package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type PrefixPair struct {
	Primary   string `yaml:"primary"`
	Secondary string `yaml:"secondary"`
}

type PrefixConfig struct {
	Para1    PrefixPair `yaml:"para1"`
	Para2    PrefixPair `yaml:"para2"`
	ULA      string     `yaml:"ula"`
	Third    string     `yaml:"third"`
	AltThird string     `yaml:"alt-third"`
}

type ServerConfig struct {
	IPv4 string `yaml:"ipv4"`
}

type APIConfig struct {
	Key    string `yaml:"key"`
	Listen string `yaml:"listen"`
}

type DatabaseConfig struct {
	MaxConnections     int    `yaml:"max_connections"`
	ConnectionLifetime string `yaml:"connection_lifetime"`
}

type Config struct {
	Prefixes PrefixConfig   `yaml:"prefixes"`
	Server   ServerConfig   `yaml:"server"`
	API      APIConfig      `yaml:"api"`
	Database DatabaseConfig `yaml:"database"`
}

var GlobalConfig Config

func LoadConfig(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("błąd odczytu pliku konfiguracyjnego: %w", err)
	}

	if err := yaml.Unmarshal(data, &GlobalConfig); err != nil {
		return fmt.Errorf("błąd parsowania konfiguracji: %w", err)
	}

	return nil
}
