package structs

type Update struct {
	ServiceName   string `json:"service_name"`
	Status        string `json:"status"`
	UpdateBatchId int    `json:"update_batch_id"`
}
