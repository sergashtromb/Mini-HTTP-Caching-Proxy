// this file is needed to implement the cache file store

package stores

import (
	"context"
	"errors"
	"log/slog"
	"mini_http_caching_proxy/tools"
	"os"
	"strconv"
	"strings"
	"time"

	"golang.org/x/sync/errgroup"
	//"github.com/google/uuid"
)

type FileCacheStore struct {
	timeDel			time.Duration
	shards 			[]*FileShard
	maxSize			int64
	tmpPath			string
	qtShard 		int
}

func NewFileCacheStore(ctx context.Context, td time.Duration, qt_sh int, max_size int64, tmpPath string) (*FileCacheStore, error) {

	fcs := &FileCacheStore{
		timeDel: td,
		tmpPath: tmpPath,
		qtShard: qt_sh,
	}

	fl_shards, err := newFileShards(tmpPath, qt_sh, max_size, td)
	if err != nil {
		return nil, err
	}

	wg, errctx := errgroup.WithContext(ctx)
	for _, fs := range fl_shards {

		wg.Go(func() error {
			if err := fs.Init(errctx, ctx); err != nil {
				return err
			}
			return nil
		})
	}

	if err := wg.Wait(); err != nil {
		slog.Error("Failed init goroutins in for", "err", err)
		return nil, err
	}

	fcs.shards = fl_shards

	return fcs, nil
}

func (fcs *FileCacheStore) Get(key string) ([]byte, error) {

	fs := fcs.getShard(key)

	data, err := fs.Get(key)
	if err != nil {
		slog.Error("Failed get data from file cache store", "err", err, "key", key)
		return  nil, err
	}

	return data, nil
}

func (fcs *FileCacheStore) Set(key string, data []byte) error {

	fs := fcs.getShard(key)

	if err := fs.Set(key, time.Now().Add(fcs.timeDel).Unix(), data); err != nil {
		slog.Error("Failed set data in file cache store", "err", err, "key", key)
		return err
	}

	return nil
}

func (fcs *FileCacheStore) SetWithExp(key string, data []byte, exp time.Duration) error {

	fs := fcs.getShard(key)

	if err := fs.Set(key, time.Now().Add(exp).Unix(), data); err != nil {
		slog.Error("Failed set data in file cache store", "err", err, "key", key)
		return err
	}

	return nil
}

func (fcs *FileCacheStore) Close() error {

	var errs []error

	for _, shard := range fcs.shards {
		err := shard.Close()
		if err != nil {
			errs = append(errs, err)
		}
		
	}

	return errors.Join(errs...)
}

func newFileShards(tmp string, qt int, max_size int64, timeDorDel time.Duration) ([]*FileShard, error) {

	files := make(map[int]string)

	ent, err := os.ReadDir(tmp)
	if err != nil {
		slog.Error("Failed get cache files", "err", err)
	}

	for _, e := range ent {

		if !e.IsDir() {

			name := strings.Split(e.Name(), "_")

			if len(name) > 1 {

				num, err := strconv.Atoi(name[0])
				if err != nil {
					continue
				}

				files[num] = e.Name()
			} 
		}
	}
	
	fss := make([]*FileShard, qt)
	for i := range qt {

		flName, ok := files[i]
		if !ok {
			flName = ""
		} 

		fs, err := NewFileShard(&tmp, flName, max_size, i, timeDorDel)
		if err != nil {
			slog.Error("Failed init file shards", "err", err)
			return nil, err
		}

		fss[i] = fs
	}

	return fss, nil
}



func (fcs *FileCacheStore) getShard(key string) *FileShard {
	idx := tools.ShardIDFromStringxxxHash(key, fcs.qtShard)
	return fcs.shards[idx]
}