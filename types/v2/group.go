package v2

const (
	GROUP_SCHEMA = `urn:ietf:params:scim:schemas:core:2.0:Group`
)

type Member struct {
	Value   string `json:"value"`
	Display string `json:"display"`
	Ref     string `json:"ref,omitempty"`
}

type Group struct {
	Schemas     []string `json:"schemas"`
	Id          string   `json:"id"`
	DisplayName string   `json:"displayName"`
	Members     []Member `json:"members"`
	Meta        Meta     `json:"meta,omitempty"`
}
