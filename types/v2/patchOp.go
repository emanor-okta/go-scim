package v2

const (
	PATCH_SCHEMA       = "urn:ietf:params:scim:api:messages:2.0:PatchOp"
	GROUP_ADD          = "add"
	GROUP_REMOVE       = "remove"
	GROUP_REPLACE      = "replace"
	GROUP_PATH_MEMBERS = "members"
)

type Operations struct {
	Op    string      `json:"op,omitempty"`
	Path  string      `json:"path,omitempty"`
	Value interface{} `json:"value,omitempty"`
}

type ValueMulti struct {
	Display string `json:"display,omitempty"`
	Value   string `json:"value,omitempty"`
}

type ValueSingle struct {
	Active      bool   `json:"active,omitempty"`
	Password    string `json:"password,omitempty"`
	Id          string `json:"id,omitempty"`
	DisplayName string `json:"displayName,omitempty"`
}

type PatchOp struct {
	Schemas    []string     `json:"schemas,omitempty"`
	Operations []Operations `json:"Operations,omitempty"`
}
