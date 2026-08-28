// this file is needed to load config's file

package config

import (
	"fmt"
	"os"

	"go.yaml.in/yaml/v4"
)

type Config struct {
	LogSettings 	LogSettings `yaml:"log_settings"`
	StoreCacheInRam bool 		`yaml:"store_cache_in_ram"`
	Hosts 			[]string 	`yaml:"hosts"`
}

type LogSettings struct {
	Level 		string `yaml:"level"`
	Directory 	string `yaml:"directory"`
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
			Level: "error",
			Directory: "logs",
		},
		StoreCacheInRam: true,
		Hosts: make([]string, 0),
	}

}
