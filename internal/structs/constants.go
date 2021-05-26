package structs

type ListElementType string
type BatchStatus = string
type ElementType int

const (
	IP     ListElementType = "IP"
	DOMAIN ListElementType = "DOMAIN"
	URL    ListElementType = "URL"
	RANGE  ListElementType = "RANGE"

	Success BatchStatus = "success"
	Failed  BatchStatus = "failed"

	Text ElementType = iota + 1
	Select
	Radio
	Number
	Password
	Disabled
)
