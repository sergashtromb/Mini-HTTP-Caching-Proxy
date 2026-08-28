package rate

import (
	"context"
	"net"
	"sync"
	"time"
)

type ShardLimiter struct {
	rm 				sync.RWMutex
	shards 			[]Shard
	limCapasity 	float64
	limRate 		float64
	minForDelate 	int16
	size 			int
}

type Shard struct {
	rm 		sync.RWMutex
	data 	map[string]*Limiter
}

func NewShardLimiter(size int, limCapasity, limRate float64, minForDelate int16) *ShardLimiter {
	return &ShardLimiter{
		shards: MakeShards(make([]Shard, size), size),
		size: size,
		limCapasity: limCapasity,
		limRate: limRate,
		minForDelate: minForDelate,
	}
}

func MakeShards(shards []Shard, size int) []Shard {
	for i := range size {
		shards[i].data = make(map[string]*Limiter)
	}
	return shards
}

func (sl *ShardLimiter) Allow(ip string) bool {

	idx := sl.getShardIndexFromIp(ip)
	shard := &sl.shards[idx]

	lim := shard.getLimiter(ip)
	if lim == nil {
		shard.addLimiter(ip, sl.limCapasity, sl.limRate)
		return true
	}

	return lim.Allow()
}

func (sh *Shard) getLimiter(ip string) *Limiter {
	sh.rm.RLock()
	defer sh.rm.Unlock()

	val, ok := sh.data[ip]
	if !ok {
		return nil
	}

	return val
}

func (sh *Shard) addLimiter(ip string, capasity, rate float64) {

	sh.rm.Lock()
	defer sh.rm.Unlock()

	newlim := NewLimiter(capasity, rate)
	sh.data[ip] = newlim

}

func (sh *Shard) delLimiter(ip string) {

	sh.rm.Lock()
	defer sh.rm.Unlock()

	delete(sh.data, ip)

}

func (sl *ShardLimiter) getShardIndexFromIp(ip string) int {
	net_ip := net.ParseIP(ip).To4()
	if net_ip == nil {
		return 0
	}

	index := uint32(net_ip[0]) << 24 | uint32(net_ip[1]) << 16 | uint32(net_ip[2]) << 8 | uint32(net_ip[3])

	return int(index % uint32(sl.size))
}

func (sl *ShardLimiter) DeleteDontUseLimiters(ctx context.Context) {

	go func() {

		ticker := time.NewTicker(5 * time.Minute)
		for {

			select {
			case <- ctx.Done():
				return
			case <- ticker.C:

				now := time.Now()

				for i := range sl.size {

					shard := &sl.shards[i]

					delIp := make([]string, 0)
					shard.rm.RLock()

					for key, val := range shard.data {
						diff := now.Sub(val.lastTime).Minutes()
						if diff >= 5 {
							delIp = append(delIp, key)
						}

					}

					shard.rm.RUnlock()

					for _, val := range delIp {
						shard.delLimiter(val)
					}

				}

			}
		}
	}()

}