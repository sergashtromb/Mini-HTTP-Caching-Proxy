package main

import (
	//"fmt"
	"fmt"
	"log/slog"
	"mini_http_caching_proxy/config"
	"mini_http_caching_proxy/domain"
	"mini_http_caching_proxy/logger"
	"os"
)

func main() {
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

	slog.Info("Hello world par", "par", "!!!")


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