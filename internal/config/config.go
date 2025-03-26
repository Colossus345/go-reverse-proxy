package config

import "github.com/Colossus345/go-reverse-proxy/internal/pipeline"

const (
	TCP  Protocol = "TCP"
	UDP  Protocol = "UPD"
	WS   Protocol = "WS"
	HTTP Protocol = "HTTP"
)

type Protocol string

type Server struct {
	ListenAddr         string
	Proto              Protocol
	InboundMiddleware  *pipeline.Pipeline
	OutboundMiddleware *pipeline.Pipeline
	RemoteAddrs        []string
}

type Config struct {
	Servers []Server
}

func NewConfig() *Config {
	in, err := pipeline.New([]string{"middlewares/log.monk"})
	if err != nil {
		panic(err)
	}
	out, err := pipeline.New([]string{"middlewares/log.monk"})
	if err != nil {
		panic(err)
	}
	return &Config{
		Servers: []Server{{ListenAddr: "localhost:3000",
			RemoteAddrs: []string{"localhost:8080"},
			Proto:       TCP, InboundMiddleware: in, OutboundMiddleware: out}},
	}
}
