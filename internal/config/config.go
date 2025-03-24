package config

const (
	TCP  Protocol = "TCP"
	UDP  Protocol = "UPD"
	WS   Protocol = "WS"
	HTTP Protocol = "HTTP"
)

type Protocol string

type Config struct {
	ListenAddr  string
	Protocols   []Protocol
	RemoteAddrs []string
}

func NewConfig() *Config {
	return &Config{
		ListenAddr:  "localhost:3000",
		RemoteAddrs: []string{"localhost:8080"},
		Protocols:   []Protocol{TCP, UDP},
	}
}
