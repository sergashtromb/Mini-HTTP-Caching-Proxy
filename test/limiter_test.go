package test

import (
	"fmt"
	"mini_http_caching_proxy/rate"
	"testing"
	"time"
)

func TestDefaultOneLimiterAllow(t *testing.T) {
	
	lim := rate.NewLimiter(10, 0)

	date := make([]bool, 11)

	for i := range 11 {
		date[i] = lim.Allow()
	}

	if date[10] {
		t.Errorf("lim.Allow(), cap-10, iter-11, 11 iter is true want false")
	}

}

func TestParallerWorkLimiter(t *testing.T) {
	
	lim := rate.NewLimiter(10, 0.1)

	mp := make(map[string]int)

	mp["1.0.0.0"] = 10
	mp["2.0.0.0"] = 16
	mp["3.0.0.0"] = 10
	mp["4.0.0.0"] = 29
	mp["5.0.0.0"] = 80

	for key, val := range mp {
		go func() {

			tiket := time.NewTicker(time.Minute)

			for {

				select {
				case <- tiket.C:

					intTik := time.NewTicker(time.Duration(val) * time.Millisecond)
					for {
						select {
						case <- intTik.C:

							res := lim.Allow()
							fmt.Errorf("%s tried %t\n", key, res)

						default:
							break
						}
					}

				default:
					break
				}

			}

		}()
	}


}