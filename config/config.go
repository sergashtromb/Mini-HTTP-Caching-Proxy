// this file is needed to load config's file

package config

import (
	"fmt"
	"log"
	"net"
	"os"
	"strconv"
	"strings"

	"go.yaml.in/yaml/v4"
)

const (
	ModeRevers = "revers"
	ModeTranspanent = "transpanent"
)

type Config struct {
	Port 			 int 				 `yaml:"port"`
	Host 			 string 			 `yaml:"host"`
	LogSettings      LogSettings         `yaml:"log_settings"`
	ShLimiter        ShardLimiterConfig  `yaml:"shard_limiter"`
	GlLimiter        GlobalLimiterConfig `yaml:"global_limiter"`
	ShardStoreConfig ShardStoreConfig    `yaml:"shard_store_config"`
	StoreCacheInRAM  bool                `yaml:"store_cache_in_ram"`
	TmpPath          string              `yaml:"tmp_path"`
	Hosts            []string            `yaml:"hosts"`
	MemBuff          int                 `yaml:"mem_buff"`
	Mode 			 string 			 `yaml:"mode"`
}

type LogSettings struct {
	Level     string `yaml:"level"`
	Directory string `yaml:"directory"`
}

type ShardLimiterConfig struct {
	Rate       float64 `yaml:"rate"`
	QtShard    int     `yaml:"qt_shard"`
	Capasity   int     `yaml:"capasity"`
	TimeForDel int     `yaml:"time_for_del"`
}

type ShardStoreConfig struct {
	QtShard    		int 	`yaml:"qt_shard"`
	TimeForDel 		int 	`yaml:"time_for_del"`
	FileSizeStore 	int64 	`yaml:"file_size_store"`
}

type GlobalLimiterConfig struct {
	Rate     float64 `yaml:"rate"`
	Capasity int     `yaml:"capasity"`
}

func Init(filename string, genConfigFile bool) Config {
	var cnf Config

	if genConfigFile {
		cnf := setDefault()

		bytes, err := yaml.Marshal(cnf)
		if err != nil {
			fmt.Println("Error generate config file, err marshal: ", err)
			return cnf
		}

		if err = os.WriteFile(filename, bytes, 0644); err != nil {
			fmt.Println("Error generate config file, err write: ", err)
			return cnf
		}

		return cnf

	} else {
		file, err := os.ReadFile(filename)
		if err != nil {
			fmt.Println("Error load config file err: ", err)
			cnf = setDefault()
		}

		if err = yaml.Unmarshal(file, &cnf); err != nil {
			fmt.Println("Error unmarshal config file err: ", err)
			cnf = setDefault()
		}

	}

	setENVParams(&cnf)

	return cnf
}

func setENVParams(cnf *Config) {

	mode := os.Getenv("MODE")
	mode = strings.TrimSpace(mode)
	if mode != ModeRevers && mode != ModeTranspanent {
		cnf.Mode = ModeRevers
	} else {
		cnf.Mode = mode
	}

	port := os.Getenv("APP_PORT")
	if port != "" {
		int_port, err := strconv.Atoi(port)
		if err != nil {
			log.Fatal("Failed set port fron env")
		}

		cnf.Port = int_port
	}

	host := os.Getenv("HOST")
	if host != "" {
		if !IsValidIP(host) {
			log.Fatal("Failed set host from env, don't valid ip")
		}
		cnf.Host = host
	}

	logLevel := os.Getenv("LOG_LEVEL")
	if logLevel != "" {
		cnf.LogSettings.Level = logLevel
	}
	// 1 - true 0 - false
	cacheInRam := os.Getenv("CACHE_IN_RAM")
	if cacheInRam != "" {

		inRam := true

		intCacheInRam, err := strconv.Atoi(cacheInRam)
		if err != nil {
			intCacheInRam = 1	
		}
 
		inRam = (intCacheInRam == 1)
		cnf.StoreCacheInRAM = inRam
	}

	tmpPath := os.Getenv("TMP_PATH")
	if tmpPath != "" {
		cnf.TmpPath = tmpPath
	}

	// TODO add another params from config to env

	hosts := os.Getenv("LIST_HOSTS")
	if hosts != "" {
		arrHosts := strings.Split(host, ",")
		// TODO chech for arr hosts
		cnf.Hosts = arrHosts
	}

}

func IsValidIP(ipStr string) bool {
	return net.ParseIP(ipStr) != nil
}

func setDefault() Config {
	return Config{
		Port: 8888,
		Host: "0.0.0.0",
		LogSettings: LogSettings{
			Level:     "info",
			Directory: "logs",
		},
		ShLimiter: ShardLimiterConfig{
			Rate:       10.0,
			QtShard:    16,
			Capasity:   100,
			TimeForDel: 5,
		},
		GlLimiter: GlobalLimiterConfig{
			Rate:     10.0,
			Capasity: 100,
		},
		ShardStoreConfig: ShardStoreConfig{
			QtShard:    12,
			TimeForDel: 20,
			FileSizeStore: 1,
		},
		StoreCacheInRAM: true,
		Hosts:           make([]string, 0),
		TmpPath: 		 "tmp",
		MemBuff: 		 1024,
		Mode: 			 ModeRevers,	
	}
}
