package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type Protocol string

const (
	TCP Protocol = "tcp"
	UDP Protocol = "udp"
)

type ServerConfig struct {
	ListenAddr string   `yaml:"listen_addr"`
	Protocol   Protocol `yaml:"protocol"`
	RemoteAddr string   `yaml:"remote_addr"`
}

type Config struct {
	Servers []ServerConfig `yaml:"servers"`
}

func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("error reading config file: %w", err)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("error parsing config file: %w", err)
	}

	// Validate configuration
	for _, server := range config.Servers {
		if server.Protocol != TCP && server.Protocol != UDP {
			return nil, fmt.Errorf("unsupported protocol: %s", server.Protocol)
		}
		if server.ListenAddr == "" {
			return nil, fmt.Errorf("listen_addr is required")
		}
		if server.RemoteAddr == "" {
			return nil, fmt.Errorf("remote_addr is required")
		}
	}

	return &config, nil
}
