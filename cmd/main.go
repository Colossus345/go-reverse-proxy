package main

import (
	"go-reverse-proxy/internal/config"
	connectionhandler "go-reverse-proxy/internal/connection_handler"
)

func main() {
	conf := config.NewConfig()

	connectionhandler.SetupServer(conf)

}
