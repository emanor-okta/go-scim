package v2

import (
	"github.com/emanor-okta/go-scim/types/v2/extension"
)

type User struct {
	CoreUser
}

type EnterpriseUser struct {
	CoreUser
	extension.Enterprise
}

// type ScimUser interface {
// 	CreateUser() error
// }

// func (*User) CreateUser() error {

// 	return nil
// }

// func (*EnterpriseUser) CreateUser() error {

// 	return nil
// }
