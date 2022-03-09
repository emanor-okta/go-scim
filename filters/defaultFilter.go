package filters

import (
	v2 "github.com/emanor-okta/go-scim/types/v2"
)

type DefaultFilter struct {
}

func (f DefaultFilter) UserPostRequest(doc []byte) []byte {
	return doc
}

func (f DefaultFilter) UserPostResponse(doc []byte) []byte {
	return doc
}

func (f DefaultFilter) UserGetResponse(doc []interface{}) []interface{} {
	return doc
}

func (f DefaultFilter) UserIdPutRequest(doc []byte) []byte {
	return doc
}

func (f DefaultFilter) UserIdPutResponse(doc []byte) []byte {
	return doc
}

func (f DefaultFilter) UserIdPatchRequest(ops *v2.PatchOp) {
}

func (f DefaultFilter) UserIdPatchResponse(doc []byte) []byte {
	return doc
}

func (f DefaultFilter) UserIdGetResponse(doc string) string {
	return doc
}

func (f DefaultFilter) GroupsGetResponse(doc []interface{}) {
}

func (f DefaultFilter) GroupsPostRequest(m map[string]interface{}) {
}

func (f DefaultFilter) GroupsPostResponse(doc []byte) []byte {
	return doc
}

func (f DefaultFilter) GroupsIdGetResponse(g interface{}) interface{} {
	return []byte(g.(string))
}

func (f DefaultFilter) GroupsIdPutRequest(m map[string]interface{}) {
}

func (f DefaultFilter) GroupsIdPutResponse(b []byte) []byte {
	return b
}

func (f DefaultFilter) GroupsIdPatchRequest(ops *v2.PatchOp) {
}
