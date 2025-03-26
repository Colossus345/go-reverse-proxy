package main

import (
	"github.com/Colossus345/go-reverse-proxy/internal/config"
	connectionhandler "github.com/Colossus345/go-reverse-proxy/internal/connection_handler"
)

func main() {
	conf := config.NewConfig()

	connectionhandler.SetupServer(conf)

}
