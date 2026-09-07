package stores

import (
	"bytes"
	"encoding/binary"
	"encoding/gob"
	"hash/fnv"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
)

type FileShard struct {
	rm 			sync.RWMutex
	index		map[string]IndexRecord
	tmpDir		*string
	file 		*os.File
	sizeFile	int64
	isFlOpen	atomic.Bool
}

func NewFileShard(tmpDir *string) (*FileShard, error) {

	fs := FileShard{
		index: make(map[string]IndexRecord),
		tmpDir: tmpDir,
	}

	if err := fs.newCacheFile(*tmpDir); err != nil {
		return nil, err
	}

	return &fs, nil
}

func (fs *FileShard) Close() error {
	err := fs.file.Close()
	return err
}

func (fs *FileShard) Get(key string) ([]byte, error) {

	fs.rm.RLock()
	defer fs.rm.RUnlock()

	hasher := fnv.New32()
	sm := hasher.Sum([]byte(key))

	offset, ok := fs.getOffset(string(sm))
	if !ok {
		return nil, nil
	}

	rec, err := fs.getRecord(offset)
	if err != nil {
		slog.Error("failed get ", "err", err)
		return nil, err
	}

	if rec == nil {
		return nil, nil
	} else {
		return *rec.GetData(), nil
	}
}

func (fs *FileShard) Set(key string, exp int64, data []byte) error {

	fs.rm.Lock()
	defer fs.rm.Unlock()

	hash := string(fnv.New32().Sum([]byte(key)))
	idx_key := NewIndexRecord(key, fs.sizeFile, exp)

	fs.index[hash] = idx_key

	rec, size := NewRecord(key, data)
	fs.sizeFile += size

	if err := fs.writeNewRecord(fs.file, rec); err != nil {
		slog.Error("failed set ", "err", err)
		return err
	}

	return nil
}

func (fs *FileShard) Delete(key string) error {

	fs.rm.Lock()
	defer fs.rm.Unlock()

	hash := string(fnv.New32().Sum([]byte(key)))
	offset, ok := fs.getOffset(hash)
	if !ok {
		return LossOfRecording("delet failed, not found in index map")
	}

	delete(fs.index, hash)
	if err := fs.setRecordForDel(offset); err != nil {
		return err
	}

	return nil
}

func (fs *FileShard) DeleteExp() {

	hashkey_for_del := make([]string, 100)

	fs.rm.RLock()

	now := time.Now()

	for key, val := range fs.index {
		if val.IsDeadFromTime(now) {
			hashkey_for_del = append(hashkey_for_del, key)
		}
	}

	fs.rm.RUnlock()

	for _, elem := range hashkey_for_del {
		if err := fs.Delete(elem); err != nil {
			slog.Error("Failed delete exp record")
		}
	}
}

func (fs *FileShard) Compose() error {

	fs.rm.Lock()
	defer fs.rm.Unlock()

	fl, err := os.OpenFile(filepath.Join(*fs.tmpDir, uuid.NewString()), os.O_RDWR|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		return err
	}

	key_for_del := make([]string, 100)

	var fl_size int64
	for key, val := range fs.index {

		off := val.GetOffset()

		rec, err := fs.getRecord(off)
		if err != nil {
			key_for_del = append(key_for_del, key)
			continue
		}

		if !rec.IsDel() {

			err = fs.writeNewRecord(fl, *rec)
			if err != nil {
				key_for_del = append(key_for_del, key)
				continue
			}

			val.offset = fl_size
			fl_size += rec.RecSize()

		} else {
			key_for_del = append(key_for_del, key)
			continue
		}
		
	}

	for _, key := range key_for_del {
		delete(fs.index, key)
	}

	fl_name := fs.file.Name()
	if err := fs.Close(); err != nil {
		return err
	}

	if err := os.Remove(fl_name); err != nil {
		return err
	}

	fs.file = fl
	fs.sizeFile = fl_size

	return nil
}

func (fs *FileShard) newCacheFile(tmpPath string) error {

	fl, err := os.OpenFile(filepath.Join(tmpPath, uuid.NewString()), os.O_RDWR|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		return err
	}

	fs.file = fl
	fs.isFlOpen.Store(true)

	stat, err := fl.Stat()
	if err != nil {
		return err
	}

	fs.sizeFile = stat.Size()

	return nil
} 

func (fs *FileShard) getOffset(hash_key string) (int64, bool) {

	idx_rec, ok := fs.index[hash_key]
	if !ok {
		return 0, false
	}

	return idx_rec.GetOffset(), true
}

func (fs *FileShard) writeNewRecord(file *os.File, rec Record) error {

	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)

	if err := enc.Encode(rec); err != nil {
		return err
	}

	if _, err := file.Write(buf.Bytes()); err != nil {
		return err
	}

	return nil
}

func (fs *FileShard) setRecordForDel(offset int64) error {

	if _, err := fs.file.WriteAt([]byte{1}, offset); err != nil {
		return err
	}

	return nil
}

func (fs *FileShard) getRecord(offset int64) (*Record, error) {

	_, err := fs.file.Seek(offset, io.SeekStart)
	if err != nil {
		slog.Error("Failed set seek in file", "err", err)
		return nil, err
	}

	del_bt := make([]byte, 1)

	_, err = fs.file.Read(del_bt)
	if err != nil && err != io.EOF {
		slog.Error("Failed read del from file", "err", err)
		return nil, err
	}

	delete := del_bt[0] == 1

	if delete {
		return nil, nil
	}

	obj_ln_bytes, key_ln_bytes := make([]byte, 4), make([]byte, 4)

	_, err = fs.file.Read(key_ln_bytes)
	if err != nil && err != io.EOF {
		slog.Error("Failed read key len from file", "err", err)
		return nil, err
	}

	key_len := binary.BigEndian.Uint32(key_ln_bytes)

	key_bytes := make([]byte, key_len)
	_, err = fs.file.Read(key_bytes) 
	if err != nil && err != io.EOF {
		slog.Error("Failed read key from file", "err", err)
		return nil, err
	}

	_, err = fs.file.Read(obj_ln_bytes)
	if err != nil && err != io.EOF {
		slog.Error("Failed read obj len from file", "err", err)
		return nil, err
	}	

	obj_len := binary.BigEndian.Uint32(obj_ln_bytes)

	obj := make([]byte, obj_len)
	_, err = fs.file.Read(obj)
	if err != nil && err != io.EOF {
		slog.Error("Failed read obj from file", "err", err)
		return nil, err
	}

	rec := NewRecordFromParams(key_len, obj_len, key_bytes, obj)

	return &rec, nil
}

