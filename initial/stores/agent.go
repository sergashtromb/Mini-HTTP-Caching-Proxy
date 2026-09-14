package stores

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"mini_http_caching_proxy/config"
	"mini_http_caching_proxy/domain"
)

type Agent struct {
	store   domain.CacheStore
	queries map[string]Sources
	mu      sync.Mutex
}

type Sources struct {
	src []chan []byte
	res chan Result
}

type Result struct {
	Value []byte
	Err   error
}

func NewAgent(ctx context.Context, isRamCache bool, cnf *config.Config) (*Agent, error) {
	var store domain.CacheStore
	if isRamCache {
		store = NewRamCacheStore(cnf, time.Duration(cnf.ShardStoreConfig.QtShard*int(time.Minute)), cnf.ShardStoreConfig.QtShard)
	} else {
		flStore, err := NewFileCacheStore(ctx, time.Duration(cnf.ShardStoreConfig.TimeForDel*int(time.Minute)), cnf.ShardStoreConfig.QtShard, 256*Mbyte, cnf.TmpPath)
		if err != nil {
			slog.Error("Failed init agent for cache store", "err", err)
			return nil, err
		}
		store = flStore
	}

	return &Agent{
		store:   store,
		queries: make(map[string]Sources),
	}, nil
}

// TEST: create a test-bench function
func (ag *Agent) Get(ctx context.Context, key string, cn chan []byte) {
	ag.mu.Lock()

	src, ok := ag.queries[key]
	if !ok {

		src := Sources{
			src: make([]chan []byte, 0),
			res: make(chan Result, 1),
		}

		src.src = append(src.src, cn)

		go func() {
			val, err := ag.store.Get(key)
			src.res <- Result{
				Value: val,
				Err:   err,
			}
		}()

	} else {
		src.src = append(src.src, cn)
	}
	ag.mu.Unlock()

	select {
	case <-ctx.Done():
		delete(ag.queries, key)
		return
	case res, ok := <-src.res:

		if !ok {
			delete(ag.queries, key)
			return
		}

		for _, val := range src.src {
			val <- res.Value
		}
		delete(ag.queries, key)
	}
}

// TODO: add set func
