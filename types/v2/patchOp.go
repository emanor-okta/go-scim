package v2

const (
	PATCH_SCHEMA = "urn:ietf:params:scim:api:messages:2.0:PatchOp"
)

type Operations struct {
	Op    string `json:"op,omitempty"`
	Value struct {
		Active   bool   `json:"active,omitempty"`
		Password string `json:"password,omitempty"`
	} `json:"value,omitempty"`
}

type PatchOp struct {
	Schemas    []string     `json:"schemas,omitempty"`
	Operations []Operations `json:"Operations,omitempty"`
}
