package stores

import (
	//"hash/fnv"
	"bytes"
	"encoding/gob"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"

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

func (fs *FileShard) getOffset(hash_key string) int64 {

	idx_rec, ok := fs.index[hash_key]
	if !ok {
		return 0
	}

	return idx_rec.GetOffset()
}

func (fs *FileShard) writeNewRecord(rec Record) error {

	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)

	if err := enc.Encode(rec); err != nil {
		return err
	}

	if _, err := fs.file.Write(buf.Bytes()); err != nil {
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

	fs.file.

	return nil, nil
}

func (fs *FileShard) Close() error {
	err := fs.file.Close()
	return err
}
