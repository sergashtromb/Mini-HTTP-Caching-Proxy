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

	lim.updateTokens()

	lim.mu.Lock()
	defer lim.mu.Unlock()
	
	if lim.tokens >= 1 {
		lim.tokens -= 1
		return true
	} else {
		return false
	}

}

func (lim *Limiter) updateTokens() {

	lim.mu.Lock()
	defer lim.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(time.Unix(lim.lastTime, 0)).Seconds()

	lim.tokens += elapsed * float64(lim.fillsp)

}