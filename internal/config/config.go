package config

import (
	"fmt"
	"os"
	"sync/atomic"

	"gopkg.in/yaml.v3"
)

type Protocol string

const (
	TCP      Protocol = "tcp"
	UDP      Protocol = "udp"
	HTTP     Protocol = "http"
	WS       Protocol = "ws"
	GRPC     Protocol = "grpc"
)

type LoadBalancingStrategy string

const (
	RoundRobin LoadBalancingStrategy = "round_robin"
	Random     LoadBalancingStrategy = "random"
)

type ServerConfig struct {
	ListenAddr           string              `yaml:"listen_addr"`
	Protocol            Protocol            `yaml:"protocol"`
	RemoteAddrs         []string            `yaml:"remote_addrs"`
	LoadBalancingStrategy LoadBalancingStrategy `yaml:"load_balancing_strategy"`
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
		if server.Protocol != TCP && server.Protocol != UDP && 
		   server.Protocol != HTTP && server.Protocol != WS && 
		   server.Protocol != GRPC {
			return nil, fmt.Errorf("unsupported protocol: %s", server.Protocol)
		}
		if server.ListenAddr == "" {
			return nil, fmt.Errorf("listen_addr is required")
		}
		if len(server.RemoteAddrs) == 0 {
			return nil, fmt.Errorf("at least one remote_addr is required")
		}
		if server.LoadBalancingStrategy == "" {
			server.LoadBalancingStrategy = RoundRobin
		}
	}

	return &config, nil
}

// RemoteAddrSelector handles the selection of remote addresses
type RemoteAddrSelector struct {
	remoteAddrs []string
	strategy    LoadBalancingStrategy
	counter     uint64
}

func NewRemoteAddrSelector(addrs []string, strategy LoadBalancingStrategy) *RemoteAddrSelector {
	return &RemoteAddrSelector{
		remoteAddrs: addrs,
		strategy:    strategy,
	}
}

func (r *RemoteAddrSelector) GetNext() string {
	switch r.strategy {
	case RoundRobin:
		next := atomic.AddUint64(&r.counter, 1)
		return r.remoteAddrs[next%uint64(len(r.remoteAddrs))]
	case Random:
		// Simple random selection - in production, you might want to use crypto/rand
		next := atomic.AddUint64(&r.counter, 1)
		return r.remoteAddrs[next%uint64(len(r.remoteAddrs))]
	default:
		return r.remoteAddrs[0]
	}
}
