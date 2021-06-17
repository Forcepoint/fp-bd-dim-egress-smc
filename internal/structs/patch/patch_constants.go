package patch

type PatchOp = string

const (
	Add     PatchOp = "add"
	Remove  PatchOp = "remove"
	Replace PatchOp = "replace"
)
