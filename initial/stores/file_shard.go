package stores

import (
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
)

const (
	Klbyte = 1024
	Mbyte = 1024*1024
	Gbyte = 1024*1024*1024
)

type FileShard struct {
	rm 			sync.RWMutex
	index		map[string]IndexRecord
	tmpDir		*string
	file 		*os.File
	sizeFile	atomic.Int64
	maxSize		int64
}

func NewFileShard(tmpDir *string, maxSize int64) (*FileShard, error) {

	fs := FileShard{
		index: make(map[string]IndexRecord),
		tmpDir: tmpDir,
		maxSize: maxSize,
	}

	if err := fs.newCacheFile(*tmpDir); err != nil {
		return nil, err
	} 

	return &fs, nil

}

// TODO create method for initialisation

func (fs *FileShard) Close() error {
	err := fs.file.Close()
	return err
}

func (fs *FileShard) Get(key string) ([]byte, error) {

	fs.rm.RLock()
	defer fs.rm.RUnlock()

	offset, size, ok := fs.getOffsetAndSize(key)
	if !ok {
		return nil, nil
	}

	rec, err := fs.getRecord(offset, size)
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

	rec, size := NewRecord(key, data)

	if fs.sizeFile.Load() > fs.maxSize + size {
		return MemoryLimit("exceeding the allowed cache file size")
	}
	
	idx_rec, ok := fs.index[key]
	
	if ok {

		old_rec, err := fs.getRecord(idx_rec.GetOffset(), idx_rec.DateLen)
		if err != nil {
			slog.Error("failed get ", "err", err)
			return err
		}

		old_rec.Delete = 1

		if err := fs.writeNewRecord(fs.file, *old_rec); err != nil {
			return FailedDeleteRecordToCache(err.Error())
		}
		fs.sizeFile.Add(old_rec.RecSize())

	}

	offset := fs.sizeFile.Load()

	if err := fs.writeNewRecord(fs.file, rec); err != nil {
		return FailedRecordToCache(err.Error())
	}

	fs.index[key] = NewIndexRecord(offset, exp, size)
	fs.sizeFile.Add(size)

	return nil
}

func (fs *FileShard) Delete(key string) error {

	fs.rm.Lock()
	defer fs.rm.Unlock()

	offset, size, ok := fs.getOffsetAndSize(key)
	if !ok {
		return LossOfRecording("delet failed, not found in index map")
	}

	rec, err := fs.getRecord(offset, size)
	if err != nil {
		slog.Error("failed get ", "err", err)
		return err
	}

	rec.Delete = 1

	if err := fs.writeNewRecord(fs.file, *rec); err != nil {
		return FailedDeleteRecordToCache(err.Error())
	}

	fs.sizeFile.Add(rec.RecSize())

	delete(fs.index, key)

	return nil
}

func (fs *FileShard) DeleteExp() error {

	fs.rm.Lock()
	defer fs.rm.Unlock()

	now := time.Now()
	for key, val := range fs.index {

		if val.IsDeadFromTime(now) {
			rec, err := fs.getRecord(val.GetOffset(), val.DateLen)
			if err != nil {
				slog.Error("failed get ", "err", err)
				continue
			}

			rec.Delete = 1

			if err := fs.writeNewRecord(fs.file, *rec); err != nil {
				return FailedDeleteRecordToCache(err.Error())
			}

			fs.sizeFile.Add(rec.RecSize())

			delete(fs.index, key)
		}
	}

	return nil
}

// TODO create method Compose() 
func (fs *FileShard) Compose() error {

	return nil
}

func (fs *FileShard) newCacheFile(tmpPath string) error {

	fl, err := os.OpenFile(filepath.Join(tmpPath, uuid.NewString()), os.O_RDWR|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		return err
	}

	fs.file = fl

	stat, err := fl.Stat()
	if err != nil {
		return err
	}

	fs.sizeFile.Store(stat.Size())

	return nil
} 

func (fs *FileShard) getOffsetAndSize(hash_key string) (int64, int64, bool) {

	idx_rec, ok := fs.index[hash_key]
	if !ok {
		return 0, 0, false
	}

	return idx_rec.GetOffset(), idx_rec.DateLen, true
}

func (fs *FileShard) writeNewRecord(file *os.File, rec Record) error {

	bt_slice := rec.ToByte()

	if _, err := file.Write(bt_slice); err != nil {
		return err
	}

	return nil
}

func (fs *FileShard) getRecord(offset int64, size int64) (*Record, error) {

	data := make([]byte, size)

	_, err := fs.file.ReadAt(data, offset)
	if err != nil && err!= io.EOF {
		return nil, err
	}

	rec := RecordFromBytesSlice(data)

	return &rec, nil
}

