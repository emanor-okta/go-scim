package filters

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/gorilla/websocket"

	v2 "github.com/emanor-okta/go-scim/types/v2"
	"github.com/emanor-okta/go-scim/utils"
)

type ManualFilter struct {
	Config *utils.Configuration
	WsConn *websocket.Conn
	ReqMap map[string]chan interface{}
}

/*
 * ReqFilter Implementations
 */
func (f ManualFilter) UserPostRequest(doc []byte, path string) []byte {
	fmt.Printf("ManualFilter UserPostRequest: %v\n", f.Config.WebMessageFilter.UserPostRequest)
	if f.Config.WebMessageFilter.UserPostRequest {
		doc = f.sendByteArrayRequest("ManualFilter.UserPostRequest", doc, path)
	}

	return doc
}

func (f ManualFilter) UserPostResponse(doc []byte, path string) []byte {
	fmt.Printf("ManualFilter UserPostResponse: %v\n", f.Config.WebMessageFilter.UserPostResponse)
	if f.Config.WebMessageFilter.UserPostResponse {
		doc = f.sendByteArrayRequest("ManualFilter.UserPostResponse", doc, path)
	}

	return doc
}

func (f ManualFilter) UserGetResponse(lr *v2.ListResponse, path string) {
	fmt.Printf("ManualFilter UserGetResponse: %v\n", f.Config.WebMessageFilter.UserGetResponse)
	if f.Config.WebMessageFilter.UserGetResponse {
		f.sendListResponseRequest("ManualFilter.UserGetResponse", lr, path)
	}
}

func (f ManualFilter) UserIdPutRequest(doc []byte, path string) []byte {
	fmt.Printf("ManualFilter UserIdPutRequest: %v\n", f.Config.WebMessageFilter.UserIdPutRequest)
	if f.Config.WebMessageFilter.UserIdPutRequest {
		doc = f.sendByteArrayRequest("ManualFilter.UserIdPutRequest", doc, path)
	}

	return doc
}

func (f ManualFilter) UserIdPutResponse(doc []byte, path string) []byte {
	fmt.Printf("ManualFilter UserIdPutResponse: %v\n", f.Config.WebMessageFilter.UserIdPutResponse)
	if f.Config.WebMessageFilter.UserIdPutResponse {
		doc = f.sendByteArrayRequest("ManualFilter.UserIdPutResponse", doc, path)
	}

	return doc
}

func (f ManualFilter) UserIdPatchRequest(ops *v2.PatchOp, path string) {
	fmt.Printf("ManualFilter UserIdPatchRequest: %v\n", f.Config.WebMessageFilter.UserIdPatchRequest)
	if f.Config.WebMessageFilter.UserIdPatchRequest {
		f.sendPatchOpRequest("ManualFilter.UserIdPatchRequest", ops, path)
	}
}

/*
 * !! NOT USED, either an error or 204 is returned !!
 */
func (f ManualFilter) UserIdPatchResponse(doc []byte, path string) []byte {
	fmt.Printf("ManualFilter UserIdPatchResponse: %v\n", f.Config.WebMessageFilter.UserIdPatchResponse)
	if f.Config.WebMessageFilter.UserIdPatchResponse {
		doc = f.sendByteArrayRequest("ManualFilter.UserIdPatchResponse", doc, path)
	}

	return doc
}

func (f ManualFilter) UserIdGetResponse(doc string, path string) string {
	fmt.Printf("ManualFilter UserIdGetResponse: %v\n", f.Config.WebMessageFilter.UserIdGetResponse)
	if f.Config.WebMessageFilter.UserIdGetResponse {
		b := f.sendByteArrayRequest("ManualFilter.UserIdGetResponse", []byte(doc), path)
		doc = string(b)
	}
	return doc
}

func (f ManualFilter) GroupsGetResponse(lr *v2.ListResponse, path string) {
	fmt.Printf("ManualFilter GroupsGetResponse: %v\n", f.Config.WebMessageFilter.GroupsGetResponse)
	if f.Config.WebMessageFilter.GroupsGetResponse {
		f.sendListResponseRequest("ManualFilter.GroupsGetResponse", lr, path)
	}
}

func (f ManualFilter) GroupsPostRequest(m map[string]interface{}, path string) {
	fmt.Printf("ManualFilter GroupsPostRequest: %v\n", f.Config.WebMessageFilter.GroupsPostRequest)
	if f.Config.WebMessageFilter.GroupsPostRequest {
		f.sendMapRequestRequest("ManualFilter.GroupsPostRequest", m, path)
	}
}

func (f ManualFilter) GroupsPostResponse(doc []byte, path string) []byte {
	fmt.Printf("ManualFilter GroupsPostResponse: %v\n", f.Config.WebMessageFilter.GroupsPostResponse)
	if f.Config.WebMessageFilter.GroupsPostResponse {
		doc = f.sendByteArrayRequest("ManualFilter.GroupsPostResponse", doc, path)
	}

	return doc
}

func (f ManualFilter) GroupsIdGetResponse(g interface{}, path string) interface{} {
	fmt.Printf("ManualFilter GroupsIdGetResponse: %v\n", f.Config.WebMessageFilter.GroupsIdGetResponse)
	if f.Config.WebMessageFilter.GroupsIdGetResponse {
		return f.sendByteArrayRequest("ManualFilter.GroupsIdGetResponse", []byte(g.(string)), path)
	}

	return []byte(g.(string))
}

func (f ManualFilter) GroupsIdPutRequest(m map[string]interface{}, path string) {
	fmt.Printf("ManualFilter GroupsIdPutRequest: %v\n", f.Config.WebMessageFilter.GroupsIdPutRequest)
	if f.Config.WebMessageFilter.GroupsIdPutRequest {
		f.sendMapRequestRequest("ManualFilter.GroupsIdPutRequest", m, path)
	}
}

func (f ManualFilter) GroupsIdPutResponse(b []byte, path string) []byte {
	fmt.Printf("ManualFilter GroupsIdPutResponse: %v\n", f.Config.WebMessageFilter.GroupsIdPutResponse)
	if f.Config.WebMessageFilter.GroupsIdPutResponse {
		b = f.sendByteArrayRequest("ManualFilter.GroupsIdPutResponse", b, path)
	}

	return b
}

func (f ManualFilter) GroupsIdPatchRequest(ops *v2.PatchOp, path string) {
	fmt.Printf("ManualFilter GroupsIdPatchRequest: %v\n", f.Config.WebMessageFilter.GroupsIdPatchRequest)
	if f.Config.WebMessageFilter.GroupsIdPatchRequest {
		f.sendPatchOpRequest("ManualFilter.GroupsIdPatchRequest", ops, path)
	}
}

/*
 * ProxyFilter Implementations
 */
func (f ManualFilter) GetRequest(h http.Header, b []byte, path string) []byte {
	return nil
}

func (f ManualFilter) GetResponse(h http.Header, b []byte, path string) []byte {
	return nil
}

func (f ManualFilter) PostRequest(h http.Header, b []byte, path string) []byte {
	return nil
}

func (f ManualFilter) PostResponse(h http.Header, c []*http.Cookie, b []byte, path string) []byte {
	fmt.Printf(">> FILTER %s <<\n", path)
	if strings.Contains(path, "/token") {
		//values := h.Get("Set-Cookie")
		h.Del("set-cookie")
		for _, v := range c {
			fmt.Printf("  Set-Cookie -> %+v\n", v)
			if v.Name == "idx" {
				fmt.Printf("Its idx...")
				if v.SameSite != http.SameSiteNoneMode {
					fmt.Println("SameSite=none not set, setting...")
					v.SameSite = http.SameSiteStrictMode
					fmt.Printf("  Set-Cookie -> %+v\n", v)
					fmt.Printf("%+v\n", v.Unparsed)
					// need to update set-cookie header ?
					h.Add("Set-Cookie", fmt.Sprintf("%s;SameSite=None", v.Raw))
				} else {
					h.Add("Set-Cookie", v.Raw)
				}
			} else {
				h.Add("Set-Cookie", v.Raw)
			}
		}
	}
	return nil
}

func (f ManualFilter) PutRequest(h http.Header, b []byte, path string) []byte {
	return nil
}

func (f ManualFilter) PutResponse(h http.Header, b []byte, path string) []byte {
	return nil
}

func (f ManualFilter) OptionsRequest(h http.Header, b []byte, path string) []byte {
	return nil
}

func (f ManualFilter) OptionsResponse(h http.Header, b []byte, path string) []byte {
	return nil
}

/*
 * Helpers
 */
func (f ManualFilter) sendByteArrayRequest(reqType string, doc []byte, path string) []byte {
	ch := make(chan interface{}, 2)
	uuid := utils.GenerateUUID()
	f.ReqMap[uuid] = ch
	var m interface{}
	err := json.Unmarshal(doc, &m)
	if err != nil {
		log.Printf("%s.Unmarshal Error: %v\n", reqType, err)
		return doc
	}

	// add a uuid which is used as a unique key in the reqMap
	m.(map[string]interface{})["uuid"] = uuid
	// add reqType (method + URL path) for web to display
	// m.(map[string]interface{})["requestType"] = f.getRequestString(reqType)
	m.(map[string]interface{})["requestType"] = path

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

func (f ManualFilter) sendPatchOpRequest(reqType string, ops *v2.PatchOp, path string) {
	b, err := json.Marshal(*ops)
	if err != nil {
		log.Printf("%s.Marshal Error: %v\n", reqType, err)
		return
	}

	b = f.sendByteArrayRequest(reqType, b, path)
	err = json.Unmarshal(b, ops)
	if err != nil {
		log.Printf("%s.UnMarshal Error: %v\n", reqType, err)
	}
}

func (f ManualFilter) sendListResponseRequest(reqType string, lr *v2.ListResponse, path string) {
	b, err := json.Marshal(*lr)
	if err != nil {
		log.Printf("%s.Marshal Error: %v\n", reqType, err)
		return
	}

	b = f.sendByteArrayRequest(reqType, b, path)
	err = json.Unmarshal(b, lr)
	if err != nil {
		log.Printf("%s.UnMarshal Error: %v\n", reqType, err)
	}
}

func (f ManualFilter) sendMapRequestRequest(reqType string, m map[string]interface{}, path string) {
	b, err := json.Marshal(m)
	if err != nil {
		log.Printf("%s.Marshal Error: %v\n", reqType, err)
		return
	}

	b = f.sendByteArrayRequest(reqType, b, path)
	err = json.Unmarshal(b, &m)
	if err != nil {
		log.Printf("%s.UnMarshal Error: %v\n", reqType, err)
	}
}

func (f ManualFilter) ToggleFilter(reqType string, state bool) {
	fmt.Printf("Setting %s to %v\n", reqType, state)
	if reqType == "UserPostRequest" {
		f.Config.WebMessageFilter.UserPostRequest = state
	} else if reqType == "UserPostResponse" {
		f.Config.WebMessageFilter.UserPostResponse = state
	} else if reqType == "UserGetResponse" {
		f.Config.WebMessageFilter.UserGetResponse = state
	} else if reqType == "UserIdPutRequest" {
		f.Config.WebMessageFilter.UserIdPutRequest = state
	} else if reqType == "UserIdPutResponse" {
		f.Config.WebMessageFilter.UserIdPutResponse = state
	} else if reqType == "UserIdPatchRequest" {
		f.Config.WebMessageFilter.UserIdPatchRequest = state
	} else if reqType == "UserIdPatchResponse" {
		f.Config.WebMessageFilter.UserIdPatchResponse = state
	} else if reqType == "UserIdGetResponse" {
		f.Config.WebMessageFilter.UserIdGetResponse = state
	} else if reqType == "GroupsGetResponse" {
		f.Config.WebMessageFilter.GroupsGetResponse = state
	} else if reqType == "GroupsPostRequest" {
		f.Config.WebMessageFilter.GroupsPostRequest = state
	} else if reqType == "GroupsPostResponse" {
		f.Config.WebMessageFilter.GroupsPostResponse = state
	} else if reqType == "GroupsIdGetResponse" {
		f.Config.WebMessageFilter.GroupsIdGetResponse = state
	} else if reqType == "GroupsIdPutRequest" {
		f.Config.WebMessageFilter.GroupsIdPutRequest = state
	} else if reqType == "GroupsIdPutResponse" {
		f.Config.WebMessageFilter.GroupsIdPutResponse = state
	} else if reqType == "GroupsIdPatchRequest" {
		f.Config.WebMessageFilter.GroupsIdPatchRequest = state
	}
}
