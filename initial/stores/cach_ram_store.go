// this file is needed to implement the cache ram store

package stores

import (
	"context"
	"log/slog"
	"mini_http_caching_proxy/config"
	"mini_http_caching_proxy/tools"
	"reflect"
	"sync"
	"time"
)

type RamCacheStore struct {
	cnf 		*config.Config
	shards 		[]ShardCache
	qt_shards	int
	timeDel		time.Duration
}

type ShardCache struct {
	dataStack map[string]Item
	rm 		  sync.RWMutex
}

type Item struct {
	Data 		interface{}
	Expiration 	int64
	DType 		reflect.Type
}

func NewRamCacheStore(cnf *config.Config, td time.Duration, qt_shards int) *RamCacheStore {
	return &RamCacheStore {
		cnf: cnf,
		shards: initShards(qt_shards),
		qt_shards: qt_shards,
		timeDel: td,
	}
}

func initShards(qt_shards int) []ShardCache {
	shards := make([]ShardCache, qt_shards)

	for i := range qt_shards {
		shards[i].dataStack = make(map[string]Item)
	}
	return shards
}

func (rcs *RamCacheStore) getShard(key string) *ShardCache {
	idx := tools.ShardIDFromStringxxxHash(key, rcs.qt_shards)
	return &rcs.shards[idx]
}

func (rcs *RamCacheStore) Set(key string, data interface{}, exp time.Duration) {

	shard := rcs.getShard(key)

	shard.rm.Lock()
	defer shard.rm.Unlock()

	item := Item {
		Data: data,
		Expiration: time.Now().Add(exp).Unix(),
		DType: reflect.TypeOf(data),
	}

	shard.dataStack[key] = item
	slog.Debug("set elem", "data", data, "idx_shard", tools.ShardIDFromStringxxxHash(key, rcs.qt_shards))
}

func (rcs *RamCacheStore) Get(key string) interface{} {

	shard := rcs.getShard(key)

	shard.rm.RLock()
	defer shard.rm.RUnlock()

	val, ok := shard.dataStack[key]	

	if !ok {
		return nil
	}

	return val.Data
}

func (rcs *RamCacheStore) GetType(key string) reflect.Type {

	shard := rcs.getShard(key)

	shard.rm.RLock()
	defer shard.rm.RUnlock()

	val, ok := shard.dataStack[key]
	if !ok {
		return nil
	}

	return val.DType
}

func (rcs *RamCacheStore) Del(key string) {

	shard := rcs.getShard(key)
	shard.rm.Lock()
	defer shard.rm.Unlock()

	delete(shard.dataStack, key)
}

func (rcs *RamCacheStore) DelWithCheckExp(key string) {

	shard := rcs.getShard(key)
	shard.rm.Lock()
	defer shard.rm.Unlock()

	val, ok := shard.dataStack[key]
	if !ok {
		return
	}

	now := time.Now()
	decs := now.Sub(time.Unix(val.Expiration, 0)).Seconds()

	if decs > 0 {
		delete(shard.dataStack, key)
	} 

}

func (rcs *RamCacheStore) DelExpiration(ctx context.Context) {

	go func() {

		ticker := time.NewTicker(rcs.timeDel)

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:

				now := time.Now()
				forDel := make([]string, 1000)

				for i := range rcs.shards {

					shard := &rcs.shards[i]
					shard.rm.RLock()

					for key := range shard.dataStack {

						item := shard.dataStack[key]
						decs := now.Sub(time.Unix(item.Expiration, 0)).Seconds()

						if decs > 0 {
							forDel = append(forDel, key)
						}

					}

					shard.rm.RUnlock()

					for _, val := range forDel {
						rcs.DelWithCheckExp(val)
					}

				}

			}
		}


	}()

}