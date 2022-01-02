package extension

const (
	ENTERPRISE_USER_SCHEMA = "urn:ietf:params:scim:schemas:extension:enterprise:2.0:User"
)

type Enterprise struct {
	// User `json:"user"`

	ENTERPRISE_USER_SCHEMA struct {
		EmployeeNumber string `json:"employeeNumber,omitempty"`
		Organization   string `json:"organization,omitempty"`
		CostCenter     string `json:"costCenter,omitempty"`
		Division       string `json:"division,omitempty"`
		Department     string `json:"department,omitempty"`

		Manager struct {
			Value   string `json:"value,omitempty"`
			Ref     string `json:"$ref,omitempty"`
			Display string `json:"display,omitempty"`
		} `json:"manager,omitempty"`
	} `json:"urn:ietf:params:scim:schemas:extension:enterprise:2.0:User"`
}
