package main

import (
	//"fmt"
	"context"
	"os/signal"
	"syscall"
	"time"

	//"crypto/tls"
	"fmt"
	"log/slog"
	"mini_http_caching_proxy/config"
	"mini_http_caching_proxy/domain"
	inboxhandler "mini_http_caching_proxy/initial/inbox_handler"
	"mini_http_caching_proxy/logger"
	"mini_http_caching_proxy/metrics"
	"mini_http_caching_proxy/rate"
	"net/http"
	"os"
	"sync"

	"github.com/go-chi/chi/v5"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	stSettings := defineStartSettings(os.Args)
	
	cnf := config.Init(stSettings.ConfigName, stSettings.GenConfigFile)

	if stSettings.GenConfigFile {
		os.Exit(0)
	}

	logger.Init(cnf.LogSettings.Directory, cnf.LogSettings.Level)

	regist := prometheus.NewRegistry()
	metr := metrics.NewMetric(regist)

	globalLimiter := rate.NewLimiter(float64(cnf.GlLimiter.Capasity), cnf.GlLimiter.Rate)
	shardLimiter :=	rate.NewShardLimiter(cnf.ShLimiter.QtShard, float64(cnf.ShLimiter.Capasity), 
		cnf.ShLimiter.Rate, int16(cnf.ShLimiter.TimeForDel))

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		shardLimiter.DeleteDontUseLimiters(ctx)
	}() 


	MetrMiddleware 	:= metrics.NewMiddleware(metr)
	Middlware 		:= inboxhandler.NewMiddleware(&cnf, globalLimiter, shardLimiter)

	route := chi.NewRouter()
	route.Use(MetrMiddleware.MetricMiddleware)
	route.Use(Middlware.InternalHostMiddleware)

	var Handler inboxhandler.Handler

	if cnf.Mode == config.ModeTranspanent {
		connManager := inboxhandler.NewConnManager(cnf.MemBuff)
		Handler = inboxhandler.NewTranspanentProxy(cnf.MemBuff, connManager)

	} else {
		cacheStore := domain.GetCacheStoreFromConfig(&cnf, ctx)
		Handler = inboxhandler.NewReveresProxy(cacheStore)
	}

	route.HandleFunc("/", Handler.InboxRequest)
	route.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
	    w.WriteHeader(http.StatusOK)
	    w.Write([]byte("ok"))
	})

	addr := fmt.Sprintf("%s:%d", cnf.Host, cnf.Port)

	server := &http.Server{
		Addr: addr,
		Handler: route,
	}

	slog.Info("Server start", "addr", addr)
	
	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("Error in listen and serve ", "err", err)
			return
		}
	}()

	http.Handle("/metrics", promhttp.HandlerFor(regist, promhttp.HandlerOpts{}))
	go func() {
		if err := http.ListenAndServe(":2525", nil); err != nil {
			slog.Error("Failed metrics server", "err", err)
			return
		}
	}()

	<-ctx.Done()

	slog.Info("Start close")

	ctxTimeout, stop := context.WithTimeout(context.Background(), 30*time.Second)
	defer stop()
	defer wg.Wait()
	
	wg.Go(func() {
		if err := server.Shutdown(ctxTimeout); err != nil {
			slog.Error("Failed server shutdown", "err", err)
		}
	})
	
	wg.Go(func() {
		if err := Handler.Shutdown(ctxTimeout); err != nil {
			slog.Error("Failed connection manager shutdown", "err", err)
		}
	})

}

func defineStartSettings(args []string) *domain.StartSettings {

	stSettings := domain.StartSettings {
		ConfigName: "config.yaml",
		GenConfigFile: false,
	}

	if len(args) <= 1 {
		return &stSettings
	}

	for _, val := range args[1:] {
		switch val {
		case "-g":
			stSettings.GenConfigFile = true
		default:
			stSettings.ConfigName = val
			break
		}
	}

	return &stSettings
}
