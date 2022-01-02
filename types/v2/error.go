package v2

const (
	ERROR_SCHEMA = `urn:ietf:params:scim:api:messages:2.0:Error`
)

type Error struct {
	Schemas []string `json:"schemas"`
	Detail  string   `json:"detail"`
	Status  int      `json:"status"`
}
