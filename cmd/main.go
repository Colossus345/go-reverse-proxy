package main

import (
	"flag"
	"log"

	"github.com/Colossus345/go-reverse-proxy/internal/config"
	"github.com/Colossus345/go-reverse-proxy/internal/proxy"
)

func main() {
	configPath := flag.String("config", "config.yaml", "path to configuration file")
	flag.Parse()

	cfg, err := config.LoadConfig(*configPath)
	if err != nil {
		log.Fatalf("Error loading configuration: %v", err)
	}

	proxy := proxy.New(cfg)
	if err := proxy.Start(); err != nil {
		log.Fatalf("Error starting proxy: %v", err)
	}

	// Keep the main goroutine running
	select {}
}
