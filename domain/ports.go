// this file is needed to define interface

package domain

import (
	"context"
	"log/slog"
	"mini_http_caching_proxy/config"
	"mini_http_caching_proxy/initial/stores"
	"time"
)

type CacheStore interface {
	Set(key string, data []byte) error
	SetWithExp(key string, data []byte, exp time.Duration) error
	Get(key string) ([]byte, error)
	Close() error
}

func GetCacheStoreFromConfig(cnf *config.Config, ctx context.Context) CacheStore {

	var cacheStore CacheStore
	
	timeForDel 	:= time.Duration(cnf.ShardStoreConfig.TimeForDel) * time.Minute
	qtShard 	:= cnf.ShardStoreConfig.QtShard

	if cnf.StoreCacheInRAM {

		ramStore := stores.NewRamCacheStore(cnf, timeForDel, qtShard)
		ramStore.DelExpiration(ctx)

		cacheStore = ramStore
		
	} else {

		fileCacheStore, err := stores.NewFileCacheStore(ctx, timeForDel, qtShard, 
			cnf.ShardStoreConfig.FileSizeStore*stores.Mbyte, cnf.TmpPath)
		if err != nil {
			slog.Error("Failed create file cache store", "err", err)
		}
		
		cacheStore = fileCacheStore
	}

	return cacheStore
}