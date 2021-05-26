package config

import (
	"fmt"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"main/internal/smc"
	"main/internal/structs"
	"net/http"
	"os"
	"strconv"
	"strings"
)

func InitConfig() {
	// Check if config file exists. If not create.
	if _, err := os.Stat("./config/config.yml"); err != nil {
		createConfig()
	}

	// Initialise viper config using config.yml file.
	viper.SetConfigName("config")
	viper.AddConfigPath("./config")

	// If unable to read in config, exit.
	if err := viper.ReadInConfig(); err != nil {
		logrus.Fatal("There was an error while trying to read the config.")
	}

	viper.WatchConfig()
}

func createConfig() {
	// Create the config
	f, err := os.Create("./config/config.yml")
	if err != nil {
		logrus.Fatal("There was an error while creating the config file.")
	}
	defer f.Close()
}

func GetConfig() structs.ModuleConfig {
	data := structs.ModuleConfig{Fields: []structs.Element{
		{
			Label:            "Requirements",
			Type:             7,
			ExpectedJsonName: "",
			Rationale:        "Forcepoint Security Management Center 6.7.3 or newer.\n\nClick the Help icon for further information on how to configure this module.",
			Value:            "",
			PossibleValues:   nil,
			Required:         false,
		}, {
			Label:            "Elements Exported",
			Type:             7,
			ExpectedJsonName: "",
			Rationale:        "IP Addresses\nIP Ranges\nDomains\nURLs",
			Value:            "",
			PossibleValues:   nil,
			Required:         false,
		}, {
			Label:            "SMC Endpoint",
			Type:             1,
			ExpectedJsonName: "smc_endpoint",
			Rationale:        "URL of the Security Management Center.",
			Value:            viper.GetString("smc_endpoint"),
			PossibleValues:   nil,
			Required:         true,
		}, {
			Label:            "SMC Port",
			Type:             4,
			ExpectedJsonName: "smc_port",
			Rationale:        "Port used by Security Management Center API service.",
			Value:            viper.GetString("smc_port"),
			PossibleValues:   []string{"1024", "49151"},
			Required:         true,
		}, {
			Label:            "SMC API Key",
			Type:             5,
			ExpectedJsonName: "smc_api_key",
			Rationale:        "API key generated in the Security Management Center.",
			Value:            viper.GetString("smc_api_key"),
			PossibleValues:   nil,
			Required:         true,
		},
	}}
	return data
}

func ValidateConfig(smcSession *smc.Session) (bool, string) {
	fmt.Println("Validating configuration.")

	// Validate that required fields aren't empty.
	smcEndpoint := viper.GetString("smc_endpoint")
	if smcEndpoint == "" {
		return false, "SMC Endpoint field is empty."
	}
	smcPort := viper.GetString("smc_port")
	if smcPort == "" {
		return false, "SMC Port field is empty."
	}
	smcAPIKey := viper.GetString("smc_api_key")
	if smcAPIKey == "" {
		return false, "SMC API Key field is empty."
	}

	// Validate SMC Endpoint value.
	validHTTP := strings.HasPrefix(smcEndpoint, "http://")
	validHTTPS := strings.HasPrefix(smcEndpoint, "https://")
	if !validHTTP && !validHTTPS {
		return false, "SMC Endpoint must be a valid address beginning with http:// or https://"
	}

	trailingSlash := strings.HasSuffix(smcEndpoint, "/")
	if trailingSlash {
		return false, "SMC Endpoint must not end with trailing /. Correct format is http://<ADDRESS_OF_SMC>."
	}

	// Validate SMC Port value.
	portAsInt, err := strconv.Atoi(smcPort)
	if err != nil {
		return false, "SMC Port entered must be a valid number."
	}

	if portAsInt <= 1023 || portAsInt > 65535 {
		return false, "SMC Port entered must be in the range 1024 - 65535."
	}

	smcSession, status, err := smc.NewSMCSession(
		smcEndpoint,
		smcPort,
		smcAPIKey)

	if err != nil {
		logrus.Error(err)
	}

	if status == http.StatusUnauthorized {
		return false, "Connection to SMC could not be created due to invalid credentials. Please validate configuration and try again."
	}
	if status != http.StatusOK {
		return false, fmt.Sprintf("Connection to SMC could not be created. Please validate configuration and try again. STATUS: %s", http.StatusText(status))
	}

	fmt.Println("Configuration Validated.")

	return true, ""
}
