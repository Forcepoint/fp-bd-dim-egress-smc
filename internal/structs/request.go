package structs

import (
	"net"
	"strings"
)

type UpdateType string

const (
	ADD    UpdateType = "add"
	DELETE UpdateType = "remove"
)

type Request struct {
	UpdateType UpdateType       `json:"update_type"`
	SafeList   bool             `json:"safe_list"`
	Items      []RequestElement `json:"items"`
	Item       RequestElement   `json:"item"`
	BatchID    int              `json:"batch_id"`
}

type RequestElement struct {
	Source      string          `json:"source"`
	ServiceName string          `json:"service_name"`
	Type        ListElementType `json:"type"`
	Value       string          `json:"value"`
	BatchNumber int             `json:"batch_number"`
}

func (r *RequestElement) IsValid() bool {
	var isValid = true
	// Check if contains subnet mask
	if strings.Count(r.Value, "/") == 0 {
		isValid = net.ParseIP(r.Value) != nil
	} else if strings.Count(r.Value, "/") == 1 {
		var value = strings.Split(r.Value, "/")[0]
		isValid = net.ParseIP(value) != nil
	} else {
		isValid = false
	}

	return isValid
}
