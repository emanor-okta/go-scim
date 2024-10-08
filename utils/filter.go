package utils

import (
	v2 "github.com/emanor-okta/go-scim/types/v2"
)

type ReqFilter interface {
	/*
		SCIM filters
	*/

	// Filters for /scim/v2/Users {POST}
	UserPostRequest([]byte, string) []byte
	UserPostResponse([]byte, string) []byte

	// Filters for /scim/v2/Users {GET} (?filter=username eq <username> AND ?startIndex=<?>&count=<?>)
	// UserGetResponse([]interface{}) []interface{}
	UserGetResponse(*v2.ListResponse, string)

	// Filters for /scim/v2/Users/<ID> {PUT}
	UserIdPutRequest([]byte, string) []byte
	UserIdPutResponse([]byte, string) []byte

	// Filters for /scim/v2/Users/<ID> {PATCH}
	UserIdPatchRequest(ops *v2.PatchOp, path string)
	UserIdPatchResponse([]byte, string) []byte

	// Filters for /scim/v2/Users/<ID> {GET}
	UserIdGetResponse(string, string) string

	// Filter for /scim/v2/Groups {GET} (?filter=displayName eq <group name> AND ?startIndex=<?>&count=<?>)
	// GroupsGetResponse([]interface{})
	GroupsGetResponse(*v2.ListResponse, string)

	// Filters for /scim/v2/Groups {POST}
	GroupsPostRequest(map[string]interface{}, string)
	GroupsPostResponse([]byte, string) []byte

	// Filters for /scim/v2/Groups/<ID> {GET}
	GroupsIdGetResponse(interface{}, string) interface{}

	// Filters for /scim/v2/Groups/<ID> {PUT}
	GroupsIdPutRequest(map[string]interface{}, string)
	GroupsIdPutResponse([]byte, string) []byte

	// Filters for /scim/v2/Groups/<ID> {PATCH}
	GroupsIdPatchRequest(ops *v2.PatchOp, path string)

	/*
		Proxy filter
	*/
	FilterRequest(h map[string][]string, b []byte, path, contentType string) (map[string][]string, []byte)
}

// type ProxyFilter interface {
// 	FilterRequest(map[string][]string, []byte, string, string) (map[string][]string, []byte)
// 	/*
// 		-- Not needed, FilterRequest will be used for both request/response and all supported methods
// 		FilterResponse(http.Header, []*http.Cookie, []byte, string) []byte

// 		GetRequest(http.Header, []byte, string) []byte
// 		GetResponse(http.Header, []byte, string) []byte
// 		PostRequest(http.Header, []byte, string) []byte
// 		PostResponse(http.Header, []*http.Cookie, []byte, string) []byte
// 		PutRequest(http.Header, []byte, string) []byte
// 		PutResponse(http.Header, []byte, string) []byte
// 		OptionsRequest(http.Header, []byte, string) []byte
// 		OptionsResponse(http.Header, []byte, string) []byte
// 	*/
// }
