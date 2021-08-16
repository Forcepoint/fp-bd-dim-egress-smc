package structs

type SMCSystemProperties struct {
	Results []SystemProperty `json:"result,omitempty"`
}

type SystemProperty struct {
	Href string `json:"href,omitempty"`
	Name string `json:"name,omitempty"`
	Type string `json:"type,omitempty"`
}
