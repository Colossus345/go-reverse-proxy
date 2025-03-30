package config

import (
	"fmt"
	"os"
	"sync/atomic"

	"github.com/Colossus345/go-reverse-proxy/internal/middleware"
	"gopkg.in/yaml.v3"
)

type Protocol string

const (
	TCP  Protocol = "tcp"
	UDP  Protocol = "udp"
	HTTP Protocol = "http"
	WS   Protocol = "ws"
	GRPC Protocol = "grpc"
)

type LoadBalancingStrategy string

const (
	RoundRobin LoadBalancingStrategy = "round_robin"
	Random     LoadBalancingStrategy = "random"
)

type ServerConfig struct {
	ListenAddr            string                        `yaml:"listen_addr"`
	Protocol              Protocol                      `yaml:"protocol"`
	RemoteAddrs           []string                      `yaml:"remote_addrs"`
	LoadBalancingStrategy LoadBalancingStrategy         `yaml:"load_balancing_strategy"`
	InboundMiddlewares    []middleware.MiddlewareConfig `yaml:"inbound_middlewares"`
	OutboundMiddlewares   []middleware.MiddlewareConfig `yaml:"outbound_middlewares"`
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

		// Validate middleware configurations
		for _, mw := range server.InboundMiddlewares {
			if mw.Name == "" {
				return nil, fmt.Errorf("middleware name is required")
			}
		}
		for _, mw := range server.OutboundMiddlewares {
			if mw.Name == "" {
				return nil, fmt.Errorf("middleware name is required")
			}
		}
	}

	return &config, nil
}

// RemoteAddrSelector handles the selection of remote addresses
type RemoteAddrSelector struct {
	addresses []string
	strategy  LoadBalancingStrategy
	index     uint64
}

func NewRemoteAddrSelector(addresses []string, strategy LoadBalancingStrategy) *RemoteAddrSelector {
	return &RemoteAddrSelector{
		addresses: addresses,
		strategy:  strategy,
	}
}

func (s *RemoteAddrSelector) GetNext() string {
	if len(s.addresses) == 0 {
		return ""
	}

	switch s.strategy {
	case RoundRobin:
		index := atomic.AddUint64(&s.index, 1) % uint64(len(s.addresses))
		return s.addresses[index]
	case Random:
		index := atomic.AddUint64(&s.index, 1) % uint64(len(s.addresses))
		return s.addresses[index]
	default:
		return s.addresses[0]
	}
}
