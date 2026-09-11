package stores

import (
	"context"
	"io"
	"log/slog"
	"maps"
	"mini_http_caching_proxy/tools"
	"os"
	"path/filepath"
	"runtime/debug"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
)

const (
	Kbyte = 1024
	Mbyte = 1024*1024
	Gbyte = 1024*1024*1024
)

type FileShard struct {
	rm 				sync.RWMutex
	index			map[string]IndexRecord
	indexComp		map[string]IndexRecord
	tmpDir			*string
	file 			*os.File
	fileComp 		*os.File
	isActiveComp 	atomic.Bool
	sizeFile		atomic.Int64
	sizeFileComp	atomic.Int64
	maxSize			int64
}

func NewFileShard(tmpDir *string, maxSize int64) (*FileShard, error) {

	fs := FileShard{
		index: make(map[string]IndexRecord),
		tmpDir: tmpDir,
		maxSize: maxSize,
	}

	fl, size, err := newCacheFile(*tmpDir)
	if err != nil {
		return nil, err
	}

	fs.file = fl
	fs.sizeFile.Store(size)

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

	isCompActive := fs.isActiveComp.Load()
	var file *os.File

	offset, size, ok := fs.getOffsetAndSize(key, isCompActive)
	if !ok {

		if isCompActive {

			offset, size, ok = fs.getOffsetAndSize(key, false)
			if !ok {
				return nil, nil
			} else {
				file = fs.file
			}

		} else {
			return nil, nil
		}

	} else {
		if isCompActive {
			file = fs.fileComp
		} else {
			file = fs.file
		}
	}

	rec, err := fs.getRecord(file, offset, size)
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

	isActComp := fs.isActiveComp.Load()

	file := fs.getFile(isActComp)

	var sizeFile int64
	if isActComp {
		sizeFile = fs.sizeFileComp.Load()
	} else {
		sizeFile = fs.sizeFile.Load()
	}

	rec, size := NewRecord(key, data)

	if sizeFile + size > fs.maxSize {
		return MemoryLimit("exceeding the allowed cache file size")
	}
	
	var idx_rec IndexRecord
	var ok bool
	if isActComp {
		idx_rec, ok = fs.indexComp[key]
	} else {
		idx_rec, ok = fs.index[key]
	}
	
	if ok {

		old_rec, err := fs.getRecord(file, idx_rec.GetOffset(), idx_rec.DateLen)
		if err != nil {
			slog.Error("failed get ", "err", err)
			return err
		}

		old_rec.Delete = 1

		if err := fs.writeNewRecord(file, *old_rec); err != nil {
			return FailedDeleteRecordToCache(err.Error())
		}
		
		if isActComp {
			fs.sizeFileComp.Add(old_rec.RecSize())
		} else {
			fs.sizeFile.Add(old_rec.RecSize())
		}	
	}
	
	var offset int64
	if isActComp {
		offset = fs.sizeFileComp.Load()
	} else {
		offset = fs.sizeFile.Load()
	}
	
	if err := fs.writeNewRecord(file, rec); err != nil {
		return FailedRecordToCache(err.Error())
	}

	if isActComp {
		fs.indexComp[key] = NewIndexRecord(offset, exp, size)
		fs.sizeFileComp.Add(size)	
	} else {
		fs.index[key] = NewIndexRecord(offset, exp, size)
		fs.sizeFile.Add(size)
	}

	return nil
}

// use only DeleteExp
func (fs *FileShard) Delete(key string) error {

	fs.rm.Lock()
	defer fs.rm.Unlock()

	isCompActive := fs.isActiveComp.Load()

	offset, size, ok := fs.getOffsetAndSize(key, isCompActive)
	if !ok {
		return LossOfRecording("delet failed, not found in index map")
	}

	file := fs.getFile(isCompActive)

	rec, err := fs.getRecord(file, offset, size)
	if err != nil {
		slog.Error("failed get ", "err", err)
		return err
	}

	rec.Delete = 1

	if err := fs.writeNewRecord(file, *rec); err != nil {
		return FailedDeleteRecordToCache(err.Error())
	}

	if isCompActive{
		fs.sizeFileComp.Add(rec.RecSize())
		delete(fs.indexComp, key)
	} else {
		fs.sizeFile.Add(rec.RecSize())
		delete(fs.index, key)
	}
	
	return nil
}

// use only with flag isActiveComp=false
func (fs *FileShard) DeleteExp() error {

	fs.rm.Lock()
	defer fs.rm.Unlock()

	if fs.isActiveComp.Load() {
		return nil
	}

	now := time.Now()
	for key, val := range fs.index {

		if val.IsDeadFromTime(now) {
			rec, err := fs.getRecord(fs.file, val.GetOffset(), val.DateLen)
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
// FIX: проблема компактизация начинается несколько раз из за этого переопредяляет поля с новым файлом и он становится nil
func (fs *FileShard) StartCompactization(ctx context.Context) {
	fs.rm.Lock()
	if fs.isActiveComp.Load() {
		return
	}
	fs.rm.Unlock()
	slog.Error("comp start")
	tools.SafeGo(func () {
		if r := recover(); r != nil {
			slog.Error("Panic in Compactization", "r", r, "stak", debug.Stack())
		}
	}, func(ctx context.Context) error {

		if err := fs.Compactization(ctx); err != nil {
			slog.Error("rollback start")
			fs.rollbackCompact()
			return err
		}

		return nil

	}, ctx)
}

// TODO: create a test function

// Start and work only in gorutin
func (fs *FileShard) Compactization(ctx context.Context) error {

	select {
	case <- ctx.Done():
		return nil
	default:
	}

	fs.rm.Lock()

	cmpFile, cmpSize, err := newCacheFile(*fs.tmpDir)
	if err != nil {
		fs.rm.Unlock()
		slog.Error("Failed create new cache file in Compactization", "err", err)
		return err
	}

	fs.fileComp = cmpFile
	fs.sizeFileComp.Store(cmpSize)
	fs.isActiveComp.Store(true)
	fs.indexComp = make(map[string]IndexRecord)
	snapshotKeys := maps.Clone(fs.index)
	fs.rm.Unlock()

	for key, _ := range snapshotKeys {

		select {
		case <- ctx.Done():
			return InteruptedCompactization()
		default:
		}
		
		fs.rm.RLock()

		_, ok := fs.indexComp[key]
		if ok {
			fs.rm.RUnlock()
			continue
		}

		idx, ok := snapshotKeys[key]
		if !ok {
			fs.rm.RUnlock()
			continue
		}

		// read from old file, 3 - param false
		data, err := fs.getRecord(fs.file, idx.GetOffset(), idx.DateLen)
		if err != nil {
			fs.rm.RUnlock()
			slog.Error("Failed get record from old file in Compactization")
			return err
		}

		fs.rm.RUnlock()
		
		fs.rm.Lock()

		_, ok = fs.indexComp[key]
		if ok {
			fs.rm.Unlock()
			continue
		}

		offset := fs.sizeFileComp.Load()
		if err := fs.writeNewRecord(fs.fileComp, *data); err != nil {
			fs.rm.Unlock()
			slog.Error("Failed write record to new comp file in Compactization", "file", fs.fileComp, "dt", *data, "err", err)
			return err
		}

		fs.sizeFileComp.Add(data.RecSize())
		new_idx := NewIndexRecord(offset, idx.Expiration, data.RecSize())
		fs.indexComp[key] = new_idx

		fs.rm.Unlock()
	}

	fs.rm.Lock()

	old_file := fs.file
	fs.file = fs.fileComp

	if err := old_file.Close(); err != nil {
		slog.Error("Failed close old file in Compactization")
	}

	if err := os.Remove(old_file.Name()); err != nil {
		slog.Error("Failed remove old file in Compactization")
	}

	fs.sizeFile.Store(fs.sizeFileComp.Load())

	fs.fileComp = nil
	fs.sizeFileComp.Store(0)

	fs.index = maps.Clone(fs.indexComp)
	clear(fs.indexComp)

	fs.isActiveComp.Store(false)

	fs.rm.Unlock()	

	return nil
}

func (fs *FileShard) rollbackCompact() {

	fs.rm.Lock()
	defer fs.rm.Unlock()

	if fs.fileComp != nil {

		if err := fs.fileComp.Close(); err != nil {
			slog.Error("Failed close comp file", "err", err)
		}

		if err := os.Remove(fs.fileComp.Name()); err != nil {
			slog.Warn("Failed remove comp file", "err", err)
		}

		fs.fileComp = nil

	}

	fs.sizeFileComp.Store(0)
	clear(fs.indexComp)
	fs.isActiveComp.Store(false)
}

func (fs *FileShard) getFile(isActiveComp bool) *os.File {
	if isActiveComp {
		return fs.fileComp
	} else {
		return fs.file
	}
}

func newCacheFile(tmpPath string) (*os.File, int64, error) {

	fl, err := os.OpenFile(filepath.Join(tmpPath, uuid.NewString()), os.O_RDWR|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		return nil, 0, err
	}

	stat, err := fl.Stat()
	if err != nil {
		return nil, 0, err
	}

	return fl, stat.Size(), nil
} 

func (fs *FileShard) getOffsetAndSize(hash_key string, isCompActive bool) (int64, int64, bool) {

	var idx_rec IndexRecord
	var ok bool

	if isCompActive {
		idx_rec, ok = fs.indexComp[hash_key]
	} else {
		idx_rec, ok = fs.index[hash_key]
	}
	
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

func (fs *FileShard) getRecord(file *os.File, offset int64, size int64) (*Record, error) {

	data := make([]byte, size)

	_, err := file.ReadAt(data, offset)
	if err != nil && err!= io.EOF {
		return nil, err
	}

	rec := RecordFromBytesSlice(data)

	return &rec, nil
}

