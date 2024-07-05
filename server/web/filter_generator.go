package web

import (
	"fmt"
	"reflect"
	"regexp"
	"strings"
)

/*
	Used to generate the source code for a filter so it can be compiled into a .so file and dynmically loaded as a plugin

	Json Path will use jq-ish syntax
	{
		"key1": "stringVal",
		"key2": {
			"inner_key1": true,
			"inner_key2": ["red", "blue", "green"]
		}
	}

	.                   = the entire Json object
	.key1               =  "stringVal"
	.key2               = the entire inner Json object
	.key2.inner_key2    = entire Json array
	.key2.inner_key2[1] = "blue"

	Each filter can contain multiple instructions
	Each instruction will be composed of:
	- operation type
		* modify           - (either adds new key or updates value if key already exists)
		* delete          - remove key value if it exists
		* array_insert    - insert into an array
		* array_append    - appends to an array
		* array_slice_del - delete a slice of elements from array
	- Json Path
	- value if applicable

	TODO - since plugins can't be unloaded, how it will be done each time a filter is updated to reload (means changing .so name)

	---------------------------------------
	Golang /encoding/json Unmarshall types
	- bool, for JSON booleans
	- float64, for JSON numbers
	- string, for JSON strings
	- []interface{}, for JSON arrays
	- map[string]interface{}, for JSON objects
	- nil for JSON null
*/

type opType byte

const (
	modify opType = iota
	del
	array_insert
	array_append
	array_slice_del
)

type filterType byte

const (
	UserPostRequest filterType = iota
	UserPostResponse
	UserGetResponse
	UserIdPutRequest
	UserIdPutResponse
	UserIdPatchRequest
	UserIdPatchResponse
	UserIdGetResponse
	GroupsGetResponse
	GroupsPostRequest
	GroupsPostResponse
	GroupsIdGetResponse
	GroupsIdPutRequest
	GroupsIdPutResponse
	GroupsIdPatchRequest
)

type instruction struct {
	jsonPath string
	op       opType
	value    interface{}
}

type filter struct {
	fType        filterType
	instructions []instruction
}

// var filters map[filterType]filter
var filterSrc map[filterType]string
var isArray *regexp.Regexp

const header = "package filters\nimport (\nv2 \"github.com/emanor-okta/go-scim/types/v2\"\n)\ntype DefaultFilter struct {}\n"

func init() {
	// filters = make(map[filterType]filter)
	filterSrc = make(map[filterType]string)
	isArray = regexp.MustCompile(`^.+\[[0-9]+\]$`)
}

func GenerateSource(filters map[filterType]filter) {
	filterSrc[UserPostRequest] = "func (f DefaultFilter) UserPostRequest(doc []byte) []byte {\nreturn doc\n}\n"
	filterSrc[UserPostResponse] = "func (f DefaultFilter) UserPostResponse(doc []byte) []byte {\nreturn doc\n}\n"
	filterSrc[UserGetResponse] = "func (f DefaultFilter) UserGetResponse(doc []interface{}) []interface{} {\nreturn doc\n}\n"
	filterSrc[UserIdPutRequest] = "func (f DefaultFilter) UserIdPutRequest(doc []byte) []byte {\nreturn doc\n}\n"
	filterSrc[UserIdPutResponse] = "func (f DefaultFilter) UserIdPutResponse(doc []byte) []byte {\nreturn doc\n}\n"
	filterSrc[UserIdPatchRequest] = "func (f DefaultFilter) UserIdPatchRequest(ops *v2.PatchOp) {\n}\n"
	filterSrc[UserIdPatchResponse] = "func (f DefaultFilter) UserIdPatchResponse(doc []byte) []byte {\nreturn doc\n}\n"
	filterSrc[UserIdGetResponse] = "func (f DefaultFilter) UserIdGetResponse(doc string) string {\nreturn doc\n}\n"
	filterSrc[GroupsGetResponse] = "func (f DefaultFilter) GroupsGetResponse(doc []interface{}) {\n}\n"
	filterSrc[GroupsPostRequest] = "func (f DefaultFilter) GroupsPostRequest(m map[string]interface{}) {\n}\n"
	filterSrc[GroupsPostResponse] = "func (f DefaultFilter) GroupsPostResponse(doc []byte) []byte {\nreturn doc\n}\n"
	filterSrc[GroupsIdGetResponse] = "func (f DefaultFilter) GroupsIdGetResponse(g interface{}) interface{} {\nreturn []byte(g.(string))\n}\n"
	filterSrc[GroupsIdPutRequest] = "func (f DefaultFilter) GroupsIdPutRequest(m map[string]interface{}) {\n}\n"
	filterSrc[GroupsIdPutResponse] = "func (f DefaultFilter) GroupsIdPutResponse(b []byte) []byte {\nreturn b\n}\n"
	filterSrc[GroupsIdPatchRequest] = "func (f DefaultFilter) GroupsIdPatchRequest(ops *v2.PatchOp) {\n}\n"

	s := generateUserPostRequest(filters[UserPostRequest])
	fmt.Println(s)
}

func mapKey(key string) string {
	return fmt.Sprintf("(map[string]interface{})[\"%s\"]", key)
}

func arrayIndex(path string) string {
	parts := strings.Split(path, "[")
	key := parts[0]
	index := strings.Split(parts[1], "]")[0]
	return fmt.Sprintf("%s.([]interface{})[%s]", mapKey(key), index)
}

func valueAsString(v interface{}) string {
	if v == nil {

	}

	switch reflect.TypeOf(v).Kind() {
	case reflect.String:
		return fmt.Sprintf("\"%s\"", v.(string))
	case reflect.Bool:
	case reflect.Float64:
		return fmt.Sprintf("%v", v)
	case reflect.Map:
		//todo
	case reflect.Array, reflect.Slice:
		//todo
	}
	return ""
}

// m["emails"].([]interface{})[0].(map[string]interface{})["value"] = "aaa@aaa.com<mailto:aaa@aaa.com>"
// .key2.inner_key2[1] = "blue"
// m.(map[string]interface{})["key2"].(map[string]interface{})["inner_key2"].([]interface{})[1] = "blue"
func generateInstruction(op opType, paths []string, value interface{}) string {
	s := "m."
	for i, v := range paths {
		fmt.Printf("%v == %v - match=%v, %v\n", i, len(paths)-1, isArray.MatchString(v), v)
		if i == len(paths)-1 {
			// last element in path
			switch op {
			case modify:
				if isArray.MatchString(v) {
					//array
					s = fmt.Sprintf("%s%s = %s\n", s, arrayIndex(v), valueAsString(value))
				} else {
					//object
					// s = fmt.Sprintf("%s%s = %s\n", s, mapKey(v), valueAsString(value))
					s = fmt.Sprintf("%s%s", s, mapKey(v))

				}
			case del:
				// s = fmt.Sprintf("delete(m, \"%s\")", s)
			case array_append:
			case array_insert:
			case array_slice_del:
			default:
				fmt.Println("TODO...")
			}
		} else {
			if isArray.MatchString(v) {
				//array
				s = fmt.Sprintf("%s%s.", s, arrayIndex(v))
			} else {
				//object
				s = fmt.Sprintf("%s%s.", s, mapKey(v))
			}
		}
	}
	return s
}

func generateUserPostRequest(f filter) string {
	/*
		var m map[string]interface{}
		json.Unmarshal(doc, &m)
		m["userName"] = "ABC@XYZ.com"
		d, err := json.Marshal(&m)
		if err != nil {
			fmt.Println(err)
		}
		return d
	*/
	s := "func (f DefaultFilter) UserPostRequest(doc []byte) []byte {\nvar m map[string]interface{}\njson.Unmarshal(doc, &m)\n"
	for _, v := range f.instructions {
		fmt.Println(v.jsonPath)
		paths := strings.Split(v.jsonPath, ".")
		fmt.Printf("%v, length:%v\n", paths, len(paths))
		if v.jsonPath == "." {
			if v.op == del {
				// delete all keys from root
				s = fmt.Sprintf("%sm = make(map[string]interface{})\n", s)
			}
			continue
		}
		s = fmt.Sprintf("%s%s\n", s, generateInstruction(v.op, paths[1:], v.value))
	}

	return fmt.Sprintf("%s}", s)
}
