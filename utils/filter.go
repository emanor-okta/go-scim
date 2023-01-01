package utils

import (
	v2 "github.com/emanor-okta/go-scim/types/v2"
)

type ReqFilter interface {
	// Filters for /scim/v2/Users {POST}
	UserPostRequest([]byte) []byte
	UserPostResponse([]byte) []byte

	// Filters for /scim/v2/Users {GET} (?filter=username eq <username> AND ?startIndex=<?>&count=<?>)
	// UserGetResponse([]interface{}) []interface{}
	UserGetResponse(*v2.ListResponse)

	// Filters for /scim/v2/Users/<ID> {PUT}
	UserIdPutRequest([]byte) []byte
	UserIdPutResponse([]byte) []byte

	// Filters for /scim/v2/Users/<ID> {PATCH}
	UserIdPatchRequest(ops *v2.PatchOp)
	UserIdPatchResponse([]byte) []byte

	// Filters for /scim/v2/Users/<ID> {GET}
	UserIdGetResponse(string) string

	// Filter for /scim/v2/Groups {GET} (?filter=displayName eq <group name> AND ?startIndex=<?>&count=<?>)
	// GroupsGetResponse([]interface{})
	GroupsGetResponse(*v2.ListResponse)

	// Filters for /scim/v2/Groups {POST}
	GroupsPostRequest(map[string]interface{})
	GroupsPostResponse([]byte) []byte

	// Filters for /scim/v2/Groups/<ID> {GET}
	GroupsIdGetResponse(interface{}) interface{}

	// Filters for /scim/v2/Groups/<ID> {PUT}
	GroupsIdPutRequest(map[string]interface{})
	GroupsIdPutResponse([]byte) []byte

	// Filters for /scim/v2/Groups/<ID> {PATCH}
	GroupsIdPatchRequest(ops *v2.PatchOp)
}
