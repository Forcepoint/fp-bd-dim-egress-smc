package main

import (
	"main/internal/config"
	"main/internal/logs"
	"main/internal/server"
	"main/internal/smc"
)

func main() {

	// Initialise logging.
	logs.InitLogrus()

	// Initialise configuration.
	config.InitConfig()

	// Register Module.
	config.RegisterModule()

	// Handle new requests to server.
	go smc.HandleRequests()

	// Run server to handle incoming requests.
	server.RunServer()
}
