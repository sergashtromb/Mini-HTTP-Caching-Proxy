package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"sync"
	"sync/atomic"
	"syscall"
	"time"
)

type Stats struct {
	sent    atomic.Int64 // попыток отправки
	ok      atomic.Int64 // 2xx
	failed  atomic.Int64 // ошибки/не-2xx
	bytesIn atomic.Int64 // принято байт
	dropped atomic.Int64 // задачи, не влезшие в очередь (воркеры не успевают)
}

// buildClient собирает http.Client, работающий через прокси.
// Поддерживаются http://, https:// и socks5://.
func buildClient(proxyURL string, timeout time.Duration, maxConns int) (*http.Client, error) {
	transport := &http.Transport{
		MaxIdleConns:        maxConns * 2,
		MaxIdleConnsPerHost: maxConns,
		MaxConnsPerHost:     maxConns,
		IdleConnTimeout:     30 * time.Second,
		DisableKeepAlives:   false,
		ForceAttemptHTTP2:   false, // HTTP/1.1 — стабильнее для прокси
	}

	if proxyURL == "" {
		return &http.Client{Transport: transport, Timeout: timeout}, nil
	}

	u, err := url.Parse(proxyURL)
	if err != nil {
		return nil, fmt.Errorf("неверный прокси %q: %w", proxyURL, err)
	}

	switch u.Scheme {
	case "http", "https":
		transport.Proxy = http.ProxyURL(u)
	default:
		return nil, fmt.Errorf("неподдерживаемая схема прокси: %s (нужно http, https или socks5)", u.Scheme)
	}

	return &http.Client{Transport: transport, Timeout: timeout}, nil
}

// sendOne отправляет ровно один и тот же GET на targetURL.
func sendOne(ctx context.Context, client *http.Client, targetURL string, stats *Stats) {
	stats.sent.Add(1)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, targetURL, nil)
	if err != nil {
		stats.failed.Add(1)
		return
	}

	resp, err := client.Do(req)
	if err != nil {
		stats.failed.Add(1)
		return
	}

	n, _ := io.Copy(io.Discard, resp.Body)
	resp.Body.Close()
	stats.bytesIn.Add(n)

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		stats.ok.Add(1)
	} else {
		stats.failed.Add(1)
	}
}

func main() {
	target := flag.String("url",
		"http://localhost:8081/?name=ivan&age=25&city=msk",
		"Полный URL (со всеми параметрами) — одинаковый для всех запросов")
	proxyURL := flag.String("proxy",
		"http://127.0.0.1:8888",
		"URL прокси: http://[user:pass@]host:port, https://..., socks5://host:port")
	workers := flag.Int("workers", 100, "Количество параллельных воркеров")
	rps := flag.Int("rps", 1000, "Целевой RPS (0 = без ограничения, максимальная скорость)")
	timeout := flag.Duration("timeout", 10*time.Second, "Таймаут одного запроса")
	statsEvery := flag.Duration("stats", 1*time.Second, "Период вывода статистики")
	flag.Parse()

	if _, err := url.ParseRequestURI(*target); err != nil {
		log.Fatalf("неверный -url: %v", err)
	}

	client, err := buildClient(*proxyURL, *timeout, *workers)
	if err != nil {
		log.Fatalf("клиент: %v", err)
	}

	stats := &Stats{}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Ctrl+C → graceful shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		log.Println("Получен сигнал — останавливаюсь...")
		cancel()
	}()

	log.Printf("Запуск:")
	log.Printf("  цель     : %s", *target)
	log.Printf("  прокси   : %s", *proxyURL)
	log.Printf("  воркеров : %d", *workers)
	if *rps > 0 {
		log.Printf("  RPS      : %d", *rps)
	} else {
		log.Printf("  RPS      : без ограничения (макс. скорость)")
	}

	var wg sync.WaitGroup

	if *rps > 0 {
		// ---- Режим с заданным RPS: диспетчер + пул воркеров ----
		interval := time.Second / time.Duration(*rps)
		if interval <= 0 {
			interval = time.Nanosecond
		}
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		// буфер на 2×воркеров — компенсирует короткие всплески
		tasks := make(chan struct{}, *workers*2)

		// производитель: кладёт ровно одну задачу за тик
		go func() {
			defer close(tasks)
			for {
				select {
				case <-ctx.Done():
					return
				case <-ticker.C:
					select {
					case tasks <- struct{}{}:
					default:
						// воркеры не успевают — очередь полна
						stats.dropped.Add(1)
					}
				}
			}
		}()

		for i := 0; i < *workers; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for range tasks {
					sendOne(ctx, client, *target, stats)
				}
			}()
		}
	} else {
		// ---- Режим без ограничения: воркеры бьют на максимум ----
		for i := 0; i < *workers; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for {
					select {
					case <-ctx.Done():
						return
					default:
						sendOne(ctx, client, *target, stats)
					}
				}
			}()
		}
	}

	// ---- Периодическая статистика ----
	statTicker := time.NewTicker(*statsEvery)
	defer statTicker.Stop()
	go func() {
		var prevSent, prevOk, prevFailed, prevBytes, prevDropped int64
		prevTime := time.Now()
		for {
			select {
			case <-ctx.Done():
				return
			case <-statTicker.C:
				now := time.Now()
				sent := stats.sent.Load()
				ok := stats.ok.Load()
				failed := stats.failed.Load()
				bytesIn := stats.bytesIn.Load()
				dropped := stats.dropped.Load()

				dt := now.Sub(prevTime).Seconds()
				rpsNow := float64(sent-prevSent) / dt
				okRate := float64(ok-prevOk) / dt
				failRate := float64(failed-prevFailed) / dt
				mbps := float64(bytesIn-prevBytes) / dt / 1024 / 1024
				dropDelta := dropped - prevDropped

				extra := ""
				if *rps > 0 && dropDelta > 0 {
					extra = fmt.Sprintf(" | drop=%d (воркеры не успевают)", dropDelta)
				}

				log.Printf("[stats] всего=%d ok=%d fail=%d | RPS=%.0f (ok=%.0f fail=%.0f) | %.2f МБ/с%s",
					sent, ok, failed, rpsNow, okRate, failRate, mbps, extra)

				prevSent, prevOk, prevFailed, prevBytes, prevDropped = sent, ok, failed, bytesIn, dropped
				prevTime = now
			}
		}
	}()

	<-ctx.Done()
	wg.Wait()

	log.Printf("Итог: отправлено=%d, успешно=%d, ошибок=%d, drop=%d, трафик=%.2f МБ",
		stats.sent.Load(), stats.ok.Load(), stats.failed.Load(), stats.dropped.Load(),
		float64(stats.bytesIn.Load())/1024/1024,
	)
}