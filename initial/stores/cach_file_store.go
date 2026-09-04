// this file is needed to implement the cache file store

package stores

import (
	//"log/slog"
	"mini_http_caching_proxy/config"
	//"os"
	//"path/filepath"
	"time"

	//"github.com/google/uuid"
)


type FileCacheStore struct {
	cnf 			*config.Config
	timeDel 		time.Duration
	timeComposition time.Duration
	shards 			[]*FileShard
	tmpPath			string
	qt_shard 		int
}


func NewFileCacheStore(cnf *config.Config, td, tc time.Duration, qt_sh int, tmpPath string) *FileCacheStore {
	fl_shards := newFileShards(qt_sh)

	return &FileCacheStore{
		cnf: cnf,
		timeDel: td,
		timeComposition: tc,
		shards: fl_shards,
		tmpPath: tmpPath,
		qt_shard: qt_sh,
	}

}

func newFileShards(qt_shard int) []*FileShard {
	new_fl_shard := make([]*FileShard, qt_shard)

	for i, _ := range new_fl_shard {
		new_fl_shard[i].index = make(map[string]IndexRecord)
	}

	return new_fl_shard
}


