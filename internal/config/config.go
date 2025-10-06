package config

import (
	"os"

	"gopkg.in/yaml.v2"
)

type Config struct {
	HTTPServer HTTPServer `yaml:"http_server"`
	Binance    Binance    `yaml:"binance"`
	Repository Repository `yaml:"repository"`
}

type HTTPServer struct {
	Host string `yaml:"host"`
	Port string `yaml:"port"`
}

type Binance struct {
	URL string `yaml:"url"`
}

type Repository struct {
	
}

func LoadConfig(configPath string) *Config {
	data, err := os.ReadFile(configPath)
	if err != nil {
		panic(err)
	}
	var config Config
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		panic(err)
	}
	return &config
}
