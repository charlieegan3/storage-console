package config

import (
	"io"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Server Server `yaml:"server"`
}

type Server struct {
	Port    int    `yaml:"port"`
	Address string `yaml:"address"`
	DevMode bool   `yaml:"dev_mode"`
}

func LoadConfig(rawConfig io.Reader) (*Config, error) {
	config := &Config{}
	if err := yaml.NewDecoder(rawConfig).Decode(config); err != nil {
		return nil, err
	}
	return config, nil
}
