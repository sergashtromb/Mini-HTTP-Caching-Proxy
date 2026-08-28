// my custom system rate limited (token bucket)

package rate

import (
	"context"
	"sync"
	"time"
)

type Limiter struct {
	lastTime 	time.Time
	mu 			sync.Mutex
	tokens 		float64
	capacity 	float64
	rate		float64
}

func New(capacity float64, rate float64) *Limiter {
	return &Limiter {
		lastTime: time.Now(),
		capacity: capacity,
		rate: rate,
		tokens: capacity,
	} 
}

func (lim *Limiter) Allow() bool {

	lim.mu.Lock()
	defer lim.mu.Unlock()

	lim.updateTokens()
	
	if lim.tokens >=  1.0 {
		lim.tokens -= 1.0
		return true
	} else {
		return false
	}
}

func (lim *Limiter) updateTokens() {

	if lim.rate == 0.0 {
		return
	}

	now := time.Now()
	elapsed := now.Sub(lim.lastTime).Seconds()

	lim.tokens += elapsed * lim.rate

	if lim.tokens >= lim.capacity {
		lim.tokens = lim.capacity
	}

	lim.lastTime = now
}

func (lim *Limiter) Wait(ctx context.Context) error {

	for {

		select {
		case <- ctx.Done():
			return ctx.Err()
		default:

		}

		lim.mu.Lock()
		lim.updateTokens()

		if lim.tokens >= 1.0 {
			lim.tokens -= 1.0
			lim.mu.Unlock()
			return nil
		}

		if lim.rate == 0.0 {
			lim.mu.Unlock()
			return nil
		}

		diff := 1.0 - lim.tokens
		duration := time.Duration(diff / lim.rate * float64(time.Second))

		lim.mu.Unlock()

		select {
		case <- ctx.Done():
			ctx.Err()
		case <- time.After(duration):

		}
	}
}