package filters

import (
	"encoding/json"
	"fmt"

	v2 "github.com/emanor-okta/go-scim/types/v2"
)

type SampleFilter struct {
}

func (f SampleFilter) UserPostRequest(doc []byte) []byte {
	fmt.Println("FILTER... UserPostRequest")
	var m map[string]interface{}
	json.Unmarshal(doc, &m)
	m["userName"] = "ABC@XYZ.com"
	d, err := json.Marshal(&m)
	if err != nil {
		fmt.Println(err)
	}
	return d
}

func (f SampleFilter) UserPostResponse(doc []byte) []byte {
	fmt.Println("FILTER... UserPostResponse")
	var m map[string]interface{}
	json.Unmarshal(doc, &m)
	m["userName"] = "FilterPOST@Response.com"
	d, err := json.Marshal(&m)
	if err != nil {
		fmt.Println(err)
	}
	return d
}

func (f SampleFilter) UserGetResponse(doc []interface{}) []interface{} {
	fmt.Println("FILTER... UserGetResponse")
	if len(doc) > 0 {
		doc[0] = f.UserIdGetResponse(doc[0].(string))
	}
	return doc
}

func (f SampleFilter) UserIdPutRequest(doc []byte) []byte {
	fmt.Println("FILTER... UserIdPutRequest")
	var m map[string]interface{}
	json.Unmarshal(doc, &m)
	m["userName"] = "FilterPUT@Request.com"
	d, err := json.Marshal(&m)
	if err != nil {
		fmt.Println(err)
	}
	return d
}

func (f SampleFilter) UserIdPutResponse(doc []byte) []byte {
	fmt.Println("FILTER... UserIdPutResponse")
	var m map[string]interface{}
	json.Unmarshal(doc, &m)
	m["userName"] = "FilterPUT@Response.com"
	d, err := json.Marshal(&m)
	if err != nil {
		fmt.Println(err)
	}
	return d
}

func (f SampleFilter) UserIdPatchRequest(ops *v2.PatchOp) {
	fmt.Println("FILTER... UserIdPatchRequest")
	for i, v := range ops.Operations {
		val := v.Value.(map[string]interface{})
		if val["password"] != nil && val["password"].(string) != "" {
			// ops.Operations[i].Value.Password = "FilteredPAssword"
			ops.Operations[i].Value.(map[string]interface{})["password"] = "FilteredPAssword"
		} else {
			// ops.Operations[i].Value.Active = true
			fmt.Println(ops.Operations[i].Value)
			ops.Operations[i].Value.(map[string]interface{})["active"] = true
			fmt.Println(ops.Operations[i].Value)
		}
	}
}

func (f SampleFilter) UserIdPatchResponse(doc []byte) []byte {
	fmt.Println("FILTER... UserIdPatchResponse")
	var m map[string]interface{}
	json.Unmarshal(doc, &m)
	m["userName"] = "FilterPATCH@Response.com"
	d, err := json.Marshal(&m)
	if err != nil {
		fmt.Println(err)
	}
	return d
}

func (f SampleFilter) UserIdGetResponse(doc string) string {
	fmt.Println("FILTER...  UserIdGetResponse")
	var m map[string]interface{}
	json.Unmarshal([]byte(doc), &m)
	m["urn:ietf:params:scim:schemas:extension:enterprise:2.0:User"] = nil
	m["name"].(map[string]interface{})["formatted"] = "First Last"
	m["emails"].([]interface{})[0].(map[string]interface{})["value"] = "aaa@aaa.com<mailto:aaa@aaa.com>"
	delete(m["meta"].(map[string]interface{}), "location")
	delete(m, "displayName")
	delete(m, "groups")
	delete(m, "locale")
	delete(m, "externalId")
	delete(m, "password")
	delete(m, "phoneNumbers")
	m["userName"] = "aaa@aaa.com<mailto:aaa@aaa.com>"
	b, _ := json.Marshal(m)
	return string(b)
}

func (f SampleFilter) GroupsGetResponse(doc []interface{}) {
	fmt.Println("FILTER...  GroupsGetResponse")
	var m map[string]interface{}
	json.Unmarshal([]byte(doc[0].(string)), &m)
	m["displayName"] = "GroupsGetResponse Filter"
	g, _ := json.Marshal(m)
	doc[0] = string(g)
}

func (f SampleFilter) GroupsPostRequest(m map[string]interface{}) {
	fmt.Println("FILTER...  GroupsPostRequest")
	m["displayName"] = "Sample Filter Modification Before Redis Persistence"
}

func (f SampleFilter) GroupsPostResponse(doc []byte) []byte {
	fmt.Println("FILTER...  GroupsPostResponse")
	var m map[string]interface{}
	json.Unmarshal([]byte(doc), &m)
	m["displayName"] = "GroupsPostResponse Filter"
	g, _ := json.Marshal(m)
	return g
}

func (f SampleFilter) GroupsIdGetResponse(g interface{}) interface{} {
	fmt.Println("FILTER...  GroupsIdGetResponse")
	var m map[string]interface{}
	json.Unmarshal([]byte(g.(string)), &m)
	m["displayName"] = "GroupsIdGetResponse Filter"
	g, _ = json.Marshal(m)
	return g
}

func (f SampleFilter) GroupsIdPutRequest(m map[string]interface{}) {
	fmt.Println("FILTER...  GroupsIdPutRequest")
	m["displayName"] = "GroupsIdPutRequest Filter"
}

func (f SampleFilter) GroupsIdPutResponse(b []byte) []byte {
	fmt.Println("FILTER...  GroupsIdPutResponse")
	var m map[string]interface{}
	json.Unmarshal(b, &m)
	m["displayName"] = "GroupsIdPutResponse Filter"
	g, _ := json.Marshal(m)
	return g
}

func (f SampleFilter) GroupsIdPatchRequest(ops *v2.PatchOp) {
	fmt.Println("FILTER... GroupsIdPatchRequest")
	for _, v := range ops.Operations {
		if v.Op == v2.GROUP_REPLACE {
			if v.Path != v2.GROUP_PATH_MEMBERS {
				v.Value.(map[string]interface{})["displayName"] = "GroupsIdPatchRequest Filter"
			}
		}
	}
}
