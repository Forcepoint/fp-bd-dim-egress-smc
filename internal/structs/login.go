package structs

type Login struct {
	Domain string `json:"domain"`
	AuthenticationKey string `json:"authenticationkey"`
}