package filters

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/gorilla/websocket"

	v2 "github.com/emanor-okta/go-scim/types/v2"
	"github.com/emanor-okta/go-scim/utils"
)

type ManualFilter struct {
	Config *utils.Configuration
	WsConn *websocket.Conn
	ReqMap map[string]chan interface{}
}

func (f ManualFilter) UserPostRequest(doc []byte) []byte {
	fmt.Printf("ManualFilter UserPostRequest: %v\n", f.Config.WebMessageFilter.UserPostRequest)
	if f.Config.WebMessageFilter.UserPostRequest {
		// ch := make(chan interface{}, 2)
		// uuid := utils.GenerateUUID()
		// f.ReqMap[uuid] = ch
		// var m interface{}
		// err := json.Unmarshal(doc, &m)
		// if err != nil {
		// 	log.Printf("ManualFilter.UserPostRequest.Unmarshal Error: %v\n", err)
		// 	return doc
		// }

		// m.(map[string]interface{})["uuid"] = uuid
		// err = f.WsConn.WriteJSON(m)
		// if err != nil {
		// 	log.Printf("ManualFilter.UserPostRequest.WriteJSON Error: %v\n", err)
		// 	return doc
		// }

		// m = <-ch
		// fmt.Printf("%+v\n", m)
		// delete(m.(map[string]interface{}), "uuid")
		// delete(f.ReqMap, uuid)
		// fmt.Printf("%+v\n", m)
		// d, err := json.Marshal(m)
		// if err != nil {
		// 	log.Printf("ManualFilter.UserPostRequest.Marshal Error: %v\n", err)
		// 	return doc
		// }

		// doc = d
		doc = f.sendByteArrayRequest("ManualFilter.UserPostRequest", doc)
	}

	return doc
}

func (f ManualFilter) UserPostResponse(doc []byte) []byte {
	fmt.Printf("ManualFilter UserPostResponse: %v\n", f.Config.WebMessageFilter.UserPostResponse)
	if f.Config.WebMessageFilter.UserPostResponse {
		doc = f.sendByteArrayRequest("ManualFilter.UserPostResponse", doc)
	}

	return doc
}

func (f ManualFilter) UserGetResponse(lr *v2.ListResponse) {
	fmt.Printf("ManualFilter UserGetResponse: %v\n", f.Config.WebMessageFilter.UserGetResponse)

}

func (f ManualFilter) UserIdPutRequest(doc []byte) []byte {
	fmt.Printf("ManualFilter UserIdPutRequest: %v\n", f.Config.WebMessageFilter.UserIdPutRequest)
	if f.Config.WebMessageFilter.UserIdPutRequest {
		doc = f.sendByteArrayRequest("ManualFilter.UserIdPutRequest", doc)
	}

	return doc
}

func (f ManualFilter) UserIdPutResponse(doc []byte) []byte {
	fmt.Printf("ManualFilter UserIdPutResponse: %v\n", f.Config.WebMessageFilter.UserIdPutResponse)
	if f.Config.WebMessageFilter.UserIdPutResponse {
		doc = f.sendByteArrayRequest("ManualFilter.UserIdPutResponse", doc)
	}

	return doc
}

func (f ManualFilter) UserIdPatchRequest(ops *v2.PatchOp) {
	fmt.Printf("ManualFilter UserIdPatchRequest: %v\n", f.Config.WebMessageFilter.UserIdPatchRequest)
}

func (f ManualFilter) UserIdPatchResponse(doc []byte) []byte {
	fmt.Printf("ManualFilter UserIdPatchResponse: %v\n", f.Config.WebMessageFilter.UserIdPatchResponse)
	if f.Config.WebMessageFilter.UserIdPatchResponse {
		doc = f.sendByteArrayRequest("ManualFilter.UserIdPatchResponse", doc)
	}

	return doc
}

func (f ManualFilter) UserIdGetResponse(doc string) string {
	fmt.Printf("ManualFilter UserIdGetResponse: %v\n", f.Config.WebMessageFilter.UserIdGetResponse)
	return doc
}

func (f ManualFilter) GroupsGetResponse(lr *v2.ListResponse) {
	fmt.Printf("ManualFilter GroupsGetResponse: %v\n", f.Config.WebMessageFilter.GroupsGetResponse)
}

func (f ManualFilter) GroupsPostRequest(m map[string]interface{}) {
	fmt.Printf("ManualFilter GroupsPostRequest: %v\n", f.Config.WebMessageFilter.GroupsPostRequest)
}

func (f ManualFilter) GroupsPostResponse(doc []byte) []byte {
	fmt.Printf("ManualFilter GroupsPostResponse: %v\n", f.Config.WebMessageFilter.GroupsPostResponse)
	if f.Config.WebMessageFilter.GroupsPostResponse {
		doc = f.sendByteArrayRequest("ManualFilter.GroupsPostResponse", doc)
	}

	return doc
}

func (f ManualFilter) GroupsIdGetResponse(g interface{}) interface{} {
	fmt.Printf("ManualFilter GroupsIdGetResponse: %v\n", f.Config.WebMessageFilter.GroupsIdGetResponse)
	return []byte(g.(string))
}

func (f ManualFilter) GroupsIdPutRequest(m map[string]interface{}) {
	fmt.Printf("ManualFilter GroupsIdPutRequest: %v\n", f.Config.WebMessageFilter.GroupsIdPutRequest)
}

func (f ManualFilter) GroupsIdPutResponse(b []byte) []byte {
	fmt.Printf("ManualFilter GroupsIdPutResponse: %v\n", f.Config.WebMessageFilter.GroupsIdPutResponse)
	if f.Config.WebMessageFilter.GroupsIdPutResponse {
		b = f.sendByteArrayRequest("ManualFilter.GroupsIdPutResponse", b)
	}

	return b
}

func (f ManualFilter) GroupsIdPatchRequest(ops *v2.PatchOp) {
	fmt.Printf("ManualFilter GroupsIdPatchRequest: %v\n", f.Config.WebMessageFilter.GroupsIdPatchRequest)
}

func (f ManualFilter) sendByteArrayRequest(reqType string, doc []byte) []byte {
	ch := make(chan interface{}, 2)
	uuid := utils.GenerateUUID()
	f.ReqMap[uuid] = ch
	var m interface{}
	err := json.Unmarshal(doc, &m)
	if err != nil {
		log.Printf("%s.Unmarshal Error: %v\n", reqType, err)
		return doc
	}

	m.(map[string]interface{})["uuid"] = uuid
	err = f.WsConn.WriteJSON(m)
	if err != nil {
		log.Printf("%s.WriteJSON Error: %v\n", reqType, err)
		return doc
	}

	m = <-ch
	fmt.Printf("%+v\n", m)
	delete(m.(map[string]interface{}), "uuid")
	delete(f.ReqMap, uuid)
	fmt.Printf("%+v\n", m)
	d, err := json.Marshal(m)
	if err != nil {
		log.Printf("%s.Marshal Error: %v\n", reqType, err)
		return doc
	}

	return d
}
