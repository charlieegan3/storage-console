package config

import (
	"fmt"
	"io"
	"log"
	"net/url"
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Server   Server            `yaml:"server"`
	Database Database          `yaml:"database"`
	Buckets  map[string]Bucket `yaml:"buckets"`
}

type Server struct {
	Port    int    `yaml:"port"`
	Address string `yaml:"address"`
	DevMode bool   `yaml:"dev_mode"`

	LoggerError *log.Logger
	LoggerInfo  *log.Logger
}

type Database struct {
	ConnectionString string `yaml:"connection_string"`
	SchemaName       string `yaml:"schema_name"`
	MigrationsTable  string `yaml:"migrations_table"`
}

type Bucket struct {
	URL       string `yaml:"url"`
	AccessKey string `yaml:"access_key"`
	SecretKey string `yaml:"secret_key"`
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
		Database struct {
			ConnectionString string            `yaml:"connection_string"`
			Params           map[string]string `yaml:"params"`
			SchemaName       string            `yaml:"schema_name"`
			MigrationsTable  string            `yaml:"migrations_table"`
		}
		Buckets map[string]Bucket `yaml:"buckets"`
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

	var db Database
	db.ConnectionString = config.Database.ConnectionString

	params := url.Values{}
	for k, v := range config.Database.Params {
		params.Add(k, v)
	}

	db.ConnectionString = fmt.Sprintf(
		"%s?%s",
		db.ConnectionString,
		params.Encode(),
	)
	db.SchemaName = config.Database.SchemaName
	db.MigrationsTable = config.Database.MigrationsTable

	return &Config{
		Server: Server{
			Port:        config.Server.Port,
			Address:     config.Server.Address,
			DevMode:     config.Server.DevMode,
			LoggerError: loggerError,
			LoggerInfo:  loggerInfo,
		},
		Database: db,
		Buckets:  config.Buckets,
	}, nil
}
