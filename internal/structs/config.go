package structs

type ModuleConfig struct {
	Fields []Element `json:"fields"`
}

type Element struct {
	Label            string      `json:"label"`
	Type             ElementType `json:"type"`
	ExpectedJsonName string      `json:"expected_json_name"`
	Rationale        string      `json:"rationale"`
	Value            string      `json:"value"`
	PossibleValues   []string    `json:"possible_values"`
	Required         bool        `json:"required"`
}

type PostedModuleConfig struct {
	Values PostedConfigValues `json:"values"`
}

type PostedConfigValues struct {
	BlocklistDuration string `json:"blocklist_duration"`
	SMCAPIKey         string `json:"smc_api_key"`
	SMCEndpoint       string `json:"smc_endpoint"`
	SMCPort           string `json:"smc_port"`
}
