package filters

import (
	"encoding/json"
	"fmt"

	v2 "github.com/emanor-okta/go-scim/types/v2"
)

type TestReqFilter struct {
	// json string
}

func (trf TestReqFilter) UserPostRequest(doc []byte) []byte {
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

func (trf TestReqFilter) UserPostResponse(doc []byte) []byte {
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

func (trf TestReqFilter) UserIdPutRequest(doc []byte) []byte {
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

func (trf TestReqFilter) UserIdPutResponse(doc []byte) []byte {
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

func (trf TestReqFilter) UserIdPatchRequest(ops *v2.PatchOp) {
	fmt.Println("FILTER... UserIdPatchRequest")
	for i, v := range ops.Operations {
		val := v.Value.(map[string]interface{})
		if val["password"].(string) != "" {
			ops.Operations[i].Value.Password = "FilteredPAssword"
		} else {
			ops.Operations[i].Value.Active = true
		}
	}
}

func (trf TestReqFilter) UserIdPatchResponse(doc []byte) []byte {
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
