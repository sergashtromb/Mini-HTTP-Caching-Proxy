// this file is needed to implement the cache ram store

package stores

import (
	"mini_http_caching_proxy/config"
	"sync"
	"time"
)

type RamCacheStore struct {
	cnf 		*config.Config
	dataStack 	map[string][]byte
	rm 			sync.RWMutex
	timeDel		time.Duration
}

func NewRamCacheStore(cnf *config.Config, td time.Duration) *RamCacheStore {
	return &RamCacheStore {
		cnf: cnf,
		dataStack: make(map[string][]byte),
		timeDel: td,
	}
}

func (rcs *RamCacheStore) Add(hash string, data []byte) {

	rcs.rm.Lock()
	defer rcs.rm.Unlock()

}


