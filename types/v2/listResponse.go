package v2

const (
	LIST_SCHEMA       = `urn:ietf:params:scim:api:messages:2.0:ListResponse`
	STR_LIST_RESPONSE = `{"schemas":["urn:ietf:params:scim:api:messages:2.0:ListResponse"],"totalResults":%v,"startIndex":%v,"itemsPerPage":%v,"resources":%v}`
)

type ListResponse struct {
	Schemas      []string      `json:"schemas,omitempty"`
	TotalResults int           `json:"totalResults"`
	StartIndex   int           `json:"startIndex"`
	ItemsPerPage int           `json:"itemsPerPage"`
	Resources    []interface{} `json:"Resources,omitempty"`
}
