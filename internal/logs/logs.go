package logs

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/sirupsen/logrus"
	"main/internal/structs"
	"net/http"
	"os"
)

func InitLogrus() {
	logrus.SetLevel(logrus.InfoLevel)
	logrus.SetReportCaller(true)
	logrus.SetOutput(os.Stdout)
	logrus.AddHook(&LoggingHook{})
}

type LoggingHook struct {
}

func (h *LoggingHook) Fire(entry *logrus.Entry) error {
	go func() {
		// Get environment variables
		internalToken := os.Getenv("INTERNAL_TOKEN")
		controllerSvcName := os.Getenv("CONTROLLER_SVC_NAME")
		controllerPort := os.Getenv("CONTROLLER_PORT")

		// Create log entry.
		log := &structs.LogEntry{
			ModuleName: os.Getenv("MODULE_SVC_NAME"),
			Level:      entry.Level.String(),
			Message:    entry.Message,
			Caller:     entry.Caller.Func.Name(),
			Time:       entry.Time,
		}

		// Get JSON module details.
		b := new(bytes.Buffer)
		json.NewEncoder(b).Encode(log)

		// Build request to register module.
		url := fmt.Sprintf("http://%s:%s/internal/logevent", controllerSvcName, controllerPort)
		client := &http.Client{}
		req, err := http.NewRequest("POST", url, b)
		if err != nil {
			logrus.Error("There was an error building the request for the log event: ", err)
		}

		// Add required headers.
		req.Header.Set("x-internal-token", internalToken)

		// Send request to register module.
		resp, err := client.Do(req)
		if err != nil {
			logrus.Error("There was an error when sending the request to log event: ", err)
		}

		if resp.StatusCode != http.StatusOK {
			fmt.Println("There was an issue with logging.")
		}

	}()

	return nil
}

// Levels define on which log levels this LoggingHook would trigger
func (h *LoggingHook) Levels() []logrus.Level {
	return []logrus.Level{logrus.WarnLevel, logrus.ErrorLevel}
}
