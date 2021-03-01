package main

import (
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
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

	sesh, _, err := smc.NewSMCSession(
		viper.GetString("smc_endpoint"),
		viper.GetString("smc_port"),
		viper.GetString("smc_api_key"))

	if err != nil {
		logrus.Error(err)
	}

	// Register Module.
	config.RegisterModule(sesh.LoggedIn)
	// Handle new requests to server.
	go smc.HandleRequests(sesh)
	// Run server to handle incoming requests.
	server.RunServer(sesh)
}
