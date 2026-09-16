// this file is needed to implement the cache ram store

package stores

import (
	"context"
	"mini_http_caching_proxy/config"
	"mini_http_caching_proxy/tools"
	"sync"
	"time"
)

type RamCacheStore struct {
	cnf 		*config.Config
	shards 		[]ShardCache
	qtShards	int
	timeDel		time.Duration
}

type ShardCache struct {
	dataStack map[string]Item
	rm 		  sync.RWMutex
}

type Item struct {
	Data 		[]byte
	Expiration 	int64
}

func NewRamCacheStore(cnf *config.Config, td time.Duration, qt_shards int) *RamCacheStore {
	return &RamCacheStore {
		cnf: cnf,
		shards: initShards(qt_shards),
		qtShards: qt_shards,
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
	idx := tools.ShardIDFromStringxxxHash(key, rcs.qtShards)
	return &rcs.shards[idx]
}

func (rcs *RamCacheStore) Set(key string, data []byte) error {

	shard := rcs.getShard(key)

	shard.rm.Lock()
	defer shard.rm.Unlock()

	item := Item {
		Data: data,
		Expiration: time.Now().Add(rcs.timeDel).Unix(),
	}

	shard.dataStack[key] = item
	return nil
}

func (rcs *RamCacheStore) SetWithExp(key string, data []byte, exp time.Duration) error {

	shard := rcs.getShard(key)

	shard.rm.Lock()
	defer shard.rm.Unlock()

	item := Item {
		Data: data,
		Expiration: time.Now().Add(exp).Unix(),
	}

	shard.dataStack[key] = item
	return nil
}

func (rcs *RamCacheStore) Get(key string) ([]byte, error) {

	shard := rcs.getShard(key)

	shard.rm.RLock()
	defer shard.rm.RUnlock()

	val, ok := shard.dataStack[key]	
	if !ok {
		return nil, nil
	}

	now := time.Now()
	decs := now.Sub(time.Unix(val.Expiration, 0)).Seconds()

	if decs > 0 {
		return nil, nil
	}

	return val.Data, nil
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
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:

				now := time.Now()
				forDel := make([]string, 0, 1000)

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