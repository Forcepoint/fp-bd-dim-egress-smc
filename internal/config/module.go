package config

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/sirupsen/logrus"
	"main/internal/structs"
	"net"
	"net/http"
	"os"
)

func RegisterModule() {
	// Get environment variables
	internalToken := os.Getenv("INTERNAL_TOKEN")
	controllerSvcName := os.Getenv("CONTROLLER_SVC_NAME")
	controllerPort := os.Getenv("CONTROLLER_PORT")

	// Get JSON module details.
	data := GetModuleDetails()
	b := new(bytes.Buffer)
	json.NewEncoder(b).Encode(data)

	// Build request to register module.
	url := fmt.Sprintf("http://%s:%s/internal/register", controllerSvcName, controllerPort)
	client := &http.Client{}
	req, err := http.NewRequest("POST", url, b)
	if err != nil {
		logrus.Fatal("There was an error while building the request to register the module: ", err)
	}

	// Add required headers.
	req.Header.Set("x-internal-token", internalToken)

	// Send request to register module.
	_, err = client.Do(req)
	if err != nil {
		logrus.Fatal("There was an error sending the request to register the module: ", err)
	}
}

func GetModuleDetails() structs.Module {
	// Validate configuration
	configured, errStr := ValidateConfig()
	fmt.Println(errStr)

	// Create Methods.
	getMethod := structs.Method{Method: "GET"}
	postMethod := structs.Method{Method: "POST"}
	optionsMethod := structs.Method{Method: "OPTIONS"}

	// Create Endpoints
	runEndpoint := structs.InternalEndpoint{
		Secure:   true,
		Endpoint: "/run",
		HttpMethods: []structs.Method{
			optionsMethod,
			postMethod,
		},
	}
	healthEndpoint := structs.InternalEndpoint{
		Secure:   true,
		Endpoint: "/health",
		HttpMethods: []structs.Method{
			optionsMethod,
			getMethod,
		},
	}
	configEndpoint := structs.InternalEndpoint{
		Secure:   true,
		Endpoint: "/config",
		HttpMethods: []structs.Method{
			optionsMethod,
			getMethod,
			postMethod,
		},
	}

	// Create Module
	module := structs.Module{
		ServiceName:       "fp-smc",
		DisplayName:       "Forcepoint SMC",
		IconURL: 		   os.Getenv("ICON_URL"),
		Type:              "egress",
		Description:       "Exports intelligence from both Safelist and Blocklist of Dynamic Intelligence Manager into IP Address Lists of Forcepoint Security Management Center, to be used in Traffic inspection policies.",
		InboundRoute:      "/fp-smc",
		InternalIP:        getIP(),
		InternalPort:      "8080",
		Configured:        configured,
		Configurable: 	   true,
		InternalEndpoints: []structs.InternalEndpoint{
			runEndpoint,
			healthEndpoint,
			configEndpoint,
		},
		AcceptedElementTypes: structs.ModuleElementTypes{
			ElementTypes: []structs.IntelligenceElementType{structs.IP, structs.RANGE},
		},
	}

	return module
}

func getIP() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return ""
	}
	for _, address := range addrs {
		if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				return ipnet.IP.String()
			}
		}
	}
	return ""
}
