package config

import (
	"io"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Server   Server   `yaml:"server"`
	WebAuthn WebAuthn `yaml:"webauthn"`
}

type Server struct {
	Port    int    `yaml:"port"`
	Address string `yaml:"address"`
}

type WebAuthn struct {
	Host    string   `yaml:"host"`
	Origins []string `yaml:"origins"`
}

func LoadConfig(rawConfig io.Reader) (*Config, error) {
	config := &Config{}
	if err := yaml.NewDecoder(rawConfig).Decode(config); err != nil {
		return nil, err
	}
	return config, nil
}
