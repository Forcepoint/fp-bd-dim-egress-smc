package structs

type List struct {
	Name string `json:"name,omitempty"`
	Comment string `json:"comment,omitempty"`
	IPList []string `json:"ip,omitempty"`
}