package structs

type SMCList struct {
	Name     string   `json:"name,omitempty"`
	Comment  string   `json:"comment,omitempty"`
	URLEntry []string `json:"url_entry,omitempty"`
	IPList   []string `json:"ip,omitempty"`
	Key      int      `json:"key,omitempty"`
}

type SMCPatch struct {
	Op    UpdateType `json:"op"`
	Path  string     `json:"path"`
	Value string     `json:"value,omitempty"`
}
