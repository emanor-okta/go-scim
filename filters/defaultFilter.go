package filters

import (
	v2 "github.com/emanor-okta/go-scim/types/v2"
)

type DefaultFilter struct {
}

func (f DefaultFilter) UserPostRequest(doc []byte, path string) []byte {
	return doc
}

func (f DefaultFilter) UserPostResponse(doc []byte, path string) []byte {
	return doc

	// fmt.Println("FILTER... UserPostResponse")
	// var m map[string]interface{}
	// json.Unmarshal(doc, &m)
	// m["id"] = "6819f362-5dc4-4804-becb-11b381e641b6"
	// m["meta"].(map[string]interface{})["location"] = "/scim/v2/Users/6819f362-5dc4-4804-becb-11b381e641b6"
	// m["active"] = false
	// d, err := json.Marshal(&m)
	// if err != nil {
	// 	fmt.Println(err)
	// }
	// return d
}

// func (f DefaultFilter) UserGetResponse(doc []interface{}) []interface{} {
// fmt.Println("FILTER... UserGetResponse")
// if len(doc) > 0 {
// 	// doc[0] = f.UserIdGetResponse(doc[0].(string))
// 	var m map[string]interface{}
// 	json.Unmarshal([]byte(f.UserIdGetResponse(doc[0].(string))), &m)
// 	m["externalId"] = "00u1qd31h9T2bGmoE1d0"
// 	b, _ := json.Marshal(m)
// 	// doc = append(doc, string(b))
// 	doc[0] = string(b)
// }
// return []interface{}{}

// return doc
// }
func (f DefaultFilter) UserGetResponse(lr *v2.ListResponse, path string) {

}

func (f DefaultFilter) UserIdPutRequest(doc []byte, path string) []byte {
	return doc
}

func (f DefaultFilter) UserIdPutResponse(doc []byte, path string) []byte {
	return doc

	// fmt.Println("FILTER... UserIdPutResponse")
	// var m map[string]interface{}
	// json.Unmarshal(doc, &m)
	// m["externalId"] = "00u1qd31h9T2bGmoE1d0"
	// m["userName"] = "FilterPUT@Response.com"
	// m["emails"].([]interface{})[0].(map[string]interface{})["value"] = "coder.joe@mail.com"
	// d, err := json.Marshal(&m)
	// if err != nil {
	// 	fmt.Println(err)
	// }
	// return d
}

func (f DefaultFilter) UserIdPatchRequest(ops *v2.PatchOp, path string) {
}

func (f DefaultFilter) UserIdPatchResponse(doc []byte, path string) []byte {
	return doc
}

func (f DefaultFilter) UserIdGetResponse(doc string, path string) string {
	return doc

	// fmt.Println("FILTER...  UserIdGetResponse")
	// var m map[string]interface{}
	// json.Unmarshal([]byte(doc), &m)
	//m["urn:ietf:params:scim:schemas:extension:enterprise:2.0:User"] = nil
	// m["name"].(map[string]interface{})["formatted"] = "First Last"
	// m["emails"].([]interface{})[0].(map[string]interface{})["primary"] = false
	// m["emails"].([]interface{})[0].(map[string]interface{})["value"] = "coder.joe@mail.com"
	// delete(m["meta"].(map[string]interface{}), "location")
	// delete(m, "displayName")
	// delete(m, "groups")
	// delete(m, "locale")
	// delete(m, "externalId")
	// delete(m, "password")
	// delete(m, "phoneNumbers")
	// m["userName"] = "aaa@aaa.com<mailto:aaa@aaa.com>"
	// m["externalId"] = "00u1qd31h9T2bGmoE1d0"
	// b, _ := json.Marshal(m)
	// return string(b)
}

// func (f DefaultFilter) GroupsGetResponse(doc []interface{}) {
// }
func (f DefaultFilter) GroupsGetResponse(lr *v2.ListResponse, path string) {

}

func (f DefaultFilter) GroupsPostRequest(m map[string]interface{}, path string) {
}

func (f DefaultFilter) GroupsPostResponse(doc []byte, path string) []byte {
	return doc

	// fmt.Println("FILTER...  GroupsPostResponse")
	// var m map[string]interface{}
	// json.Unmarshal([]byte(doc), &m)
	// m["displayName"] = ""
	// delete(m, "displayName")
	// g, _ := json.Marshal(m)
	// return g
}

func (f DefaultFilter) GroupsIdGetResponse(g interface{}, path string) interface{} {
	return []byte(g.(string))

	// fmt.Println("FILTER...  GroupsIdGetResponse")
	// var m map[string]interface{}
	// json.Unmarshal([]byte(g.(string)), &m)
	// // m["members"] = []interface{}{}
	// delete(m, "displayName")
	// g, _ = json.Marshal(m)
	// return g
}

func (f DefaultFilter) GroupsIdPutRequest(m map[string]interface{}, path string) {
}

func (f DefaultFilter) GroupsIdPutResponse(b []byte, path string) []byte {
	return b

	// fmt.Println("FILTER...  GroupsIdPutResponse")
	// var m map[string]interface{}
	// json.Unmarshal(b, &m)
	// // m["members"] = []interface{}{}
	// delete(m, "displayName")
	// g, _ := json.Marshal(m)
	// return g
}

func (f DefaultFilter) GroupsIdPatchRequest(ops *v2.PatchOp, path string) {
}
