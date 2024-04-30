package config

import (
	"fmt"
	"io"
	"log"
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Server Server `yaml:"server"`
}

type Server struct {
	Port    int    `yaml:"port"`
	Address string `yaml:"address"`
	DevMode bool   `yaml:"dev_mode"`

	LoggerError *log.Logger
	LoggerInfo  *log.Logger
}

func LoadConfig(rawConfig io.Reader) (*Config, error) {
	config := struct {
		Server struct {
			Port    int    `yaml:"port"`
			Address string `yaml:"address"`
			DevMode bool   `yaml:"dev_mode"`

			Log struct {
				Error string `yaml:"error"`
				Info  string `yaml:"info"`
			} `yaml:"log"`
		} `yaml:"server"`
	}{}
	if err := yaml.NewDecoder(rawConfig).Decode(&config); err != nil {
		return nil, fmt.Errorf("failed to decode config: %w", err)
	}

	var loggerError *log.Logger
	if config.Server.Log.Error == "stderr" {
		loggerError = log.New(os.Stderr, "", log.LstdFlags)
	}

	var loggerInfo *log.Logger
	if config.Server.Log.Info == "stdout" {
		loggerInfo = log.New(os.Stdout, "", log.LstdFlags)
	}

	return &Config{
		Server: Server{
			Port:        config.Server.Port,
			Address:     config.Server.Address,
			DevMode:     config.Server.DevMode,
			LoggerError: loggerError,
			LoggerInfo:  loggerInfo,
		},
	}, nil
}
