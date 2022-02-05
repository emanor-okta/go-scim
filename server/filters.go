package server

import (
	v2 "github.com/emanor-okta/go-scim/types/v2"
)

type Filter int64

/*
 * Filters for /scim/v2/Users {POST}
 */
var UsersPostReqFilter UsersPostReq
var UsersPostResFilter UsersPostRes

type UsersPostReq interface {
	UserPostRequest(doc []byte) []byte
}
type UsersPostRes interface {
	UserPostResponse(doc []byte) []byte
}

/*
 * Filters for /scim/v2/Users/<ID> {PUT/PATCH}
 */
var UsersPutReqFilter UsersIdPutReq
var UsersPutResFilter UsersIdPutRes
var UsersPatchReqFilter UsersIdPatchReq
var UsersPatchResFilter UsersIdPatchRes

type UsersIdPutReq interface {
	UserIdPutRequest(doc []byte) []byte
}
type UsersIdPutRes interface {
	UserIdPutResponse(doc []byte) []byte
}
type UsersIdPatchReq interface {
	UserIdPatchRequest(ops *v2.PatchOp)
}
type UsersIdPatchRes interface {
	UserIdPatchResponse([]byte) []byte
}
