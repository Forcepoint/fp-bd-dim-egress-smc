package server

import (
	"encoding/json"
	"fmt"
	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"log"
	"main/internal/channel"
	conf "main/internal/config"
	"main/internal/smc"
	"main/internal/structs"
	"net/http"
	"reflect"
)

func health(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		w.WriteHeader(http.StatusOK)
	} else {
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func config(session *smc.Session) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" {

			// Set response headers
			w.Header().Set("Content-Type", "application/json")

			// Get config
			data := conf.GetConfig()

			// Return json response
			if err := json.NewEncoder(w).Encode(data); err != nil {
				logrus.Error("There was an error encoding the config for response: ", err)
				return
			}

		} else if r.Method == "POST" {
			// Get posted config data.
			var config structs.PostedModuleConfig
			err := json.NewDecoder(r.Body).Decode(&config)
			if err != nil {
				logrus.Error("There was an error decoding the POST config: ", err)
				w.WriteHeader(http.StatusBadRequest)
				return
			}

			// Store current configuration.
			blocklistDuration := viper.GetString("blocklist_duration")
			smcEndpoint := viper.GetString("smc_endpoint")
			smcPort := viper.GetString("smc_port")
			smcAPIKey := viper.GetString("smc_api_key")

			// Handle posted config data.
			values := reflect.Indirect(reflect.ValueOf(config.Values))
			numFields := values.Type().NumField()
			for i := 0; i < numFields; i++ {
				field := values.Type().Field(i).Tag.Get("json")
				value := values.Field(i)
				viper.Set(field, value.String())
			}
			if err := viper.WriteConfig(); err != nil {
				logrus.Error("error writing config", err)
			}

			// Validate configuration.
			valid, errStr := conf.ValidateConfig(session)
			if !valid {
				viper.Set("blocklist_duration", blocklistDuration)
				viper.Set("smc_endpoint", smcEndpoint)
				viper.Set("smc_port", smcPort)
				viper.Set("smc_api_key", smcAPIKey)
				if err := viper.WriteConfig(); err != nil {
					logrus.Error("error writing config", err)
				}
				w.WriteHeader(http.StatusBadRequest)
				w.Write([]byte(errStr))
			} else {
				conf.RegisterModule(session.LoggedIn)
				w.WriteHeader(http.StatusAccepted)
			}
		}
	})
}

func run(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		// Create struct for element sent.
		var request structs.Request

		// Parse request.
		err := json.NewDecoder(r.Body).Decode(&request)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		channel.Requests <- request

		w.WriteHeader(http.StatusAccepted)

	} else {
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func RunServer(smcSession *smc.Session) {
	// Create Router for Server
	router := mux.NewRouter().StrictSlash(true)

	// Create Routes
	router.HandleFunc("/health", health).Methods("OPTIONS", "GET")
	router.Handle("/config", config(smcSession)).Methods("OPTIONS", "GET", "POST")
	router.HandleFunc("/run", run).Methods("OPTIONS", "POST")

	fmt.Println("Starting Server")

	// Start Server
	log.Fatal(http.ListenAndServe("0.0.0.0:8080", router))
}
