package v2

const (
	USER_SCHEMA = "urn:ietf:params:scim:schemas:core:2.0:User"
)

type CoreUser struct {
	// Mandatory values
	Schemas  []string `json:"schemas"`
	Id       string   `json:"id"`
	UserName string   `json:"userName"`
	Meta     Meta     `json:"meta"`

	//optionals
	ExternalId        string `json:"externalId,omitempty"`
	Name              string `json:"name,omitempty"`
	DisplayName       string `json:"displayName,omitempty"`
	NickName          string `json:"nickName,omitempty"`
	ProfileUrl        string `json:"profileUrl,omitempty"`
	UserType          string `json:"userType,omitempty"`
	Title             string `json:"title,omitempty"`
	PreferredLanguage string `json:"preferredLanguage,omitempty"`
	Locale            string `json:"locale,omitempty"`
	Timezone          string `json:"timezone,omitempty"`
	Active            bool   `json:"active,omitempty"`
	Password          string `json:"password,omitempty"`

	Emails       []Email       `json:"emails,omitempty"`
	Addresses    []Address     `json:"addresses,omitempty"`
	PhoneNumbers []PhoneNumber `json:"phoneNumbers,omitempty"`
	Ims          []Im          `json:"ims,omitempty"`
	Photos       []Photo       `json:"photos,omitempty"`
	UserGroups   []UserGroup   `json:"groups,omitempty"`
}

type Email struct {
	Value   string `json:"value"`
	Type    string `json:"type"`
	Primary bool   `json:"primary,omitempty"`
}

type Address struct {
	Type          string `json:"type"`
	StreetAddress string `json:"streetAddress,omitempty"`
	Locality      string `json:"locality,omitempty"`
	Region        string `json:"region,omitempty"`
	PostalCode    string `json:"postalCode,omitempty"`
	Country       string `json:"country,omitempty"`
	Formatted     string `json:"formatted,omitempty"`
	Primary       bool   `json:"primary,omitempty"`
}

type PhoneNumber struct {
	Value string `json:"value"`
	Type  string `json:"type"`
}

type Im struct {
	Value string `json:"value"`
	Type  string `json:"type"`
}

type Photo struct {
	Value string `json:"value"`
	Type  string `json:"type"`
}

type UserGroup struct {
	Value   string `json:"value"`
	Ref     string `json:"$ref"`
	Display string `json:"display"`
}
