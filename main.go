package main

import (
	//"fmt"
	"context"
	"fmt"
	"log/slog"
	"mini_http_caching_proxy/config"
	"mini_http_caching_proxy/domain"
	inboxhandler "mini_http_caching_proxy/initial/inbox_handler"
	"mini_http_caching_proxy/logger"
	"mini_http_caching_proxy/rate"
	"net/http"
	"os"
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

	Middlware := inboxhandler.NewMiddleware(&cnf, globalLimiter, shardLimiter)
	Handler := inboxhandler.NewInboxHandler()

	servMux := http.NewServeMux()
	servMux.HandleFunc("/", Handler.HandleInboxReq)

	handle := Middlware.InternalHostMiddleware(servMux)

	server := &http.Server{
		Addr: ":8080",
		Handler: handle,
	}

	go func() {
		if err := server.ListenAndServe(); err != nil {
			slog.Error("Error in listen and serve ", "err", err)
			return
		}
	}()


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