package v2

type Meta struct {
	ResourceType string `json:"resourceType,omitempty"`
	Created      string `json:"created,omitempty"`      // !!date??
	LastModified string `json:"lastModified,omitempty"` // !!date??
	Version      string `json:"version,omitempty"`
	Location     string `json:"location,omitempty"`
}
