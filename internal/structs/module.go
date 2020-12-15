package structs

type IntelligenceElementType string

const (
	IP     IntelligenceElementType = "IP"
	DOMAIN IntelligenceElementType = "DOMAIN"
	URL    IntelligenceElementType = "URL"
	RANGE  IntelligenceElementType = "RANGE"
)

type Module struct {
	ServiceName          string             `json:"module_service_name"`
	DisplayName          string             `json:"module_display_name"`
	IconURL 			 string 			`json:"icon_url"`
	Type                 string             `json:"module_type"`
	Description          string             `json:"module_description"`
	InboundRoute         string             `json:"inbound_route"`
	InternalIP           string             `json:"internal_ip"`
	InternalPort         string             `json:"internal_port"`
	Configured           bool               `json:"configured"`
	Configurable         bool               `json:"configurable"`
	AcceptedElementTypes ModuleElementTypes `json:"accepted_element_types"`
	InternalEndpoints    []InternalEndpoint `json:"internal_endpoints"`
}

type ModuleElementTypes struct {
	ElementTypes []IntelligenceElementType `json:"element_types"`
}

type InternalEndpoint struct {
	Secure      bool     `json:"secure"`
	Endpoint    string   `json:"endpoint"`
	HttpMethods []Method `json:"http_methods"`
}

type Method struct {
	Method string `json:"method"`
}
