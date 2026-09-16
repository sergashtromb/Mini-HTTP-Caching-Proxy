package main

import (
	//"fmt"
	"context"
	"time"
	//"crypto/tls"
	"fmt"
	"log/slog"
	"mini_http_caching_proxy/config"
	"mini_http_caching_proxy/domain"
	inboxhandler "mini_http_caching_proxy/initial/inbox_handler"
	"mini_http_caching_proxy/initial/stores"
	"mini_http_caching_proxy/logger"
	"mini_http_caching_proxy/rate"
	"net/http"
	"os"
	"sync"

	"github.com/go-chi/chi/v5"
)

func main() {

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// set start settings
	stSettings := defineStartSettings(os.Args)
	// load config or generate config file, default config.yaml
	cnf := config.Init(stSettings.ConfigName, stSettings.GenConfigFile)

	if stSettings.GenConfigFile {
		os.Exit(0)
	}

	logFile, err := logger.Init(cnf.LogSettings.Directory, cnf.LogSettings.Level)
	if err != nil {
		fmt.Println("Error load logger err:", err)
		return
	}
	defer logFile.Close()

	globalLimiter := rate.NewLimiter(float64(cnf.GlLimiter.Capasity), cnf.GlLimiter.Rate)
	shardLimiter :=	rate.NewShardLimiter(cnf.ShLimiter.QtShard, float64(cnf.ShLimiter.Capasity), 
		cnf.ShLimiter.Rate, int16(cnf.ShLimiter.TimeForDel))

	go shardLimiter.DeleteDontUseLimiters(ctx)

	var cacheStore domain.CacheStore
	
	timeForDel := time.Duration(int64(cnf.ShardStoreConfig.TimeForDel) * int64(time.Minute))
	qtShard := cnf.ShardStoreConfig.QtShard

	if cnf.StoreCacheInRAM {

		ramStore := stores.NewRamCacheStore(&cnf, timeForDel, qtShard)
		ramStore.DelExpiration(ctx)
		cacheStore = ramStore
		
	} else {
		cacheStore, err = stores.NewFileCacheStore(ctx, timeForDel, qtShard, cnf.ShardStoreConfig.FileSizeStore*stores.Mbyte, 
			cnf.TmpPath)
		if err != nil {
			slog.Error("Failed create file cache store", "err", err)
		}
	}

	Middlware := inboxhandler.NewMiddleware(&cnf, globalLimiter, shardLimiter)
	Handler := inboxhandler.NewInboxHandler(&cnf, cacheStore)

	route := chi.NewRouter()
	route.Use(Middlware.InternalHostMiddleware)
	route.HandleFunc("/", Handler.HandleInboxReq)
	route.Connect("/", Handler.HandleConnection)

	addr := fmt.Sprintf("%s:%d", cnf.Host, cnf.Port)

	server := &http.Server{
		Addr: addr,
		Handler: route,
	}
	slog.Info("Server start")
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := server.ListenAndServe(); err != nil {
			slog.Error("Error in listen and serve ", "err", err)
			return
		}
	}()

	wg.Wait()
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
