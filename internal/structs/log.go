package structs

import "time"

type LogEntry struct {
	ModuleName string    `json:"module_name"`
	Level      string    `json:"level"`
	Message    string    `json:"message"`
	Caller     string    `json:"caller"`
	Time       time.Time `json:"time"`
}
