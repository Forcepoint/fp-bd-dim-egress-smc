package structs

type Blocklist struct {
	Entries []BlocklistEntry `json:"entries"`
}

type BlocklistEntry struct {
	EndpointOne BlocklistEndpoint `json:"end_point1"`
	EndpointTwo BlocklistEndpoint `json:"end_point2"`
	Duration    int               `json:"duration"`
}

type BlocklistEndpoint struct {
	AddressMode string `json:"address_mode"`
	IPNetwork   string `json:"ip_network"`
}
