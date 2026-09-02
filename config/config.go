// this file is needed to load config's file

package config

import (
	"fmt"
	"os"

	"go.yaml.in/yaml/v4"
)

type Config struct {
	LogSettings 	LogSettings 		`yaml:"log_settings"`
	ShLimiter 		ShardLimiterConfig 	`yaml:"shard_limiter"`
	GlLimiter 		GlobalLimiterConfig `yaml:"global_limiter"`
	StoreCacheInRam bool 				`yaml:"store_cache_in_ram"`
	Hosts 			[]string 			`yaml:"hosts"`
	MemBuff			int 				`yaml:"mem_buff"`
}

type LogSettings struct {
	Level 		string `yaml:"level"`
	Directory 	string `yaml:"directory"`
}

type ShardLimiterConfig struct {
	Rate 		float64 `yaml:"rate"`
	QtShard 	int 	`yaml:"qt_shard"`
	Capasity 	int 	`yaml:"capasity"`
	TimeForDel	int 	`yaml:"time_for_del"`
}

type GlobalLimiterConfig struct {
	Rate 		float64 `yaml:"rate"`
	Capasity 	int 	`yaml:"capasity"`
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
			return setDefault()
		}

		if err = yaml.Unmarshal(file, &cnf); err != nil {
			fmt.Println("Error unmarshal config file err: ", err)
			return setDefault()
		}

	}
	
	return cnf

}

func setDefault() Config {	
	return Config {
		LogSettings: LogSettings {
			Level: "debug",
			Directory: "logs",
		},
		ShLimiter: ShardLimiterConfig {
			Rate: 10.0,
			QtShard: 16,
			Capasity: 100,
			TimeForDel: 5,
		},
		GlLimiter: GlobalLimiterConfig {
			Rate: 10.0,
			Capasity: 100,
		},
		StoreCacheInRam: true,
		Hosts: make([]string, 0),
	}

}
