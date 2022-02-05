package v2

const (
	TYPE_USER      = "User"
	TYPE_GROUP     = "Group"
	LOCATION_USER  = "/scim/v2/Users/"
	LOCATION_GROUP = "/scim/v2/Groups/"
)

type Meta struct {
	ResourceType string `json:"resourceType,omitempty"`
	Created      string `json:"created,omitempty"`      // !!date??
	LastModified string `json:"lastModified,omitempty"` // !!date??
	Version      string `json:"version,omitempty"`
	Location     string `json:"location,omitempty"`
}
