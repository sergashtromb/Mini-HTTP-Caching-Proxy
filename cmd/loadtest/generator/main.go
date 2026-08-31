package main

import (
	"fmt"
	"mini_http_caching_proxy/rate"
	"sync"
	"sync/atomic"
	"time"
)


func main() {

	workers := 10000
	rpc := 10000
	var qt_allowed int64
	var qt_denided int64
	var qt_total int64
	stopChanel := make(chan struct{})
	startTime := time.Now()
	shardLimiter := rate.NewShardLimiter(2, 100, 10, 5)

	var wg sync.WaitGroup

	timer := 50 * time.Second

	for i := 0; i < workers; i++ {
		wg.Add(1)

		go func(id int) {
			defer wg.Done()
			baseIp := fmt.Sprintf("1.1.%d", workers%255)

			for {
				select{
				case <-stopChanel:
					return
				default:
					
					ip := fmt.Sprintf("%s.%d", baseIp, id)	

					allowed := shardLimiter.Allow(ip)
					atomic.AddInt64(&qt_total, 1)

					if allowed {
						atomic.AddInt64(&qt_allowed, 1)
					} else {
						atomic.AddInt64(&qt_denided, 1)
					}

					time.Sleep(time.Second / time.Duration(rpc/workers))

				}
			}

		}(i)
	}

	time.Sleep(timer)
	close(stopChanel)
	wg.Wait()


	elapsed := time.Since(startTime).Seconds()

	fmt.Printf("Всего обращений - %d\n", qt_total)
	fmt.Printf("Принято обращений - %d (%.2f%%)\n", qt_allowed, float64(qt_allowed)/float64(qt_total)*100)
	fmt.Printf("Откланено обращений - %d (%.2f%%)\n", qt_denided, float64(qt_denided)/float64(qt_total)*100)
	fmt.Printf("RPS: %.2f\n", float64(qt_total)/elapsed)

}