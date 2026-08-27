// my custom system rate limited (token bucket)

package rate

import (
	"sync"
	"time"
)

type Limiter struct {
	mu 			sync.Mutex
	tokens 		float64
	capacity 	float64
	lastTime 	int64
	fillsp		int16
	
}

func New(capacity float64, fillsp int16) *Limiter {
	return &Limiter {
		lastTime: time.Now().Unix(),
		capacity: capacity,
		fillsp: fillsp,
		tokens: capacity,
	} 
}

func (lim *Limiter) Allow() bool {

	lim.mu.Lock()
	defer lim.mu.Unlock()
	// TODO make update limiter

	return false
}
