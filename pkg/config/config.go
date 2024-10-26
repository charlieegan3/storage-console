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
	Server                 Server                           `yaml:"server"`
	Database               Database                         `yaml:"database"`
	ObjectStorageProviders map[string]ObjectStorageProvider `yaml:"object_storage_providers"`
	Buckets                map[string]Bucket                `yaml:"buckets"`
}

type ObjectStorageProvider struct {
	URL       string `yaml:"url"`
	AccessKey string `yaml:"access_key"`
	SecretKey string `yaml:"secret_key"`
}

type Server struct {
	Port    int    `yaml:"port"`
	Address string `yaml:"address"`
	DevMode bool   `yaml:"dev_mode"`

	RegisterMux bool `yaml:"register_mux"`
	RunImporter bool `yaml:"run_importer"`

	LoggerError *log.Logger
	LoggerInfo  *log.Logger
}

type Database struct {
	ConnectionString string `yaml:"connection_string"`
	SchemaName       string `yaml:"schema_name"`
	MigrationsTable  string `yaml:"migrations_table"`
}

type Bucket struct {
	Provider string `yaml:"provider"`
	Default  bool   `yaml:"default"`
}

func LoadConfig(rawConfig io.Reader) (*Config, error) {
	config := struct {
		Server struct {
			Port    int    `yaml:"port"`
			Address string `yaml:"address"`
			DevMode bool   `yaml:"dev_mode"`

			RunImporter bool `yaml:"run_importer"`
			RegisterMux bool `yaml:"register_mux"`

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
		ObjectStorageProviders map[string]ObjectStorageProvider `yaml:"object_storage_providers"`
		Buckets                map[string]Bucket                `yaml:"buckets"`
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

	if len(params) > 0 {
		db.ConnectionString = fmt.Sprintf("%s?%s", db.ConnectionString, params.Encode())
	}

	db.SchemaName = config.Database.SchemaName
	db.MigrationsTable = config.Database.MigrationsTable

	return &Config{
		Server: Server{
			Port:        config.Server.Port,
			Address:     config.Server.Address,
			DevMode:     config.Server.DevMode,
			LoggerError: loggerError,
			LoggerInfo:  loggerInfo,
			RegisterMux: config.Server.RegisterMux,
			RunImporter: config.Server.RunImporter,
		},
		Database:               db,
		ObjectStorageProviders: config.ObjectStorageProviders,
		Buckets:                config.Buckets,
	}, nil
}
