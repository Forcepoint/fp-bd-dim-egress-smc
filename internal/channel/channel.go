package channel

import "main/internal/structs"

var Requests = make(chan structs.Request, 100)
