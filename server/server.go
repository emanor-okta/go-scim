package server

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"

	v2 "github.com/emanor-okta/go-scim/types/v2"
	"github.com/emanor-okta/go-scim/utils"
)

const (
	NOT_FOUND           = "not_found"
	USER_ALREADY_EXISTS = "User already exists in the database."
	GROUPALREADY_EXISTS = "Group already exists in the database."
)

var debugHeaders bool
var debugBody bool
var enableGroups bool

func StartServer(config *utils.Configuration) {
	debugHeaders = config.Server.Debug_headers
	debugBody = config.Server.Debug_body
	enableGroups = config.Scim.Enable_groups
	log.Printf("starting server at %v\n", config.Server.Address)

	http.HandleFunc("/scim/v2/Users", handleUsers)
	http.HandleFunc("/scim/v2/Users/", handleUser)
	http.HandleFunc("/scim/v2/Groups", handleGroups)
	http.HandleFunc("/scim/v2/Groups/", handleGroup)

	http.HandleFunc("/scim/v1/Users", handleUsers)
	http.HandleFunc("/scim/v1/Users/", handleUser)
	http.HandleFunc("/scim/v1/Groups", handleGroups)
	http.HandleFunc("/scim/v1/Groups/", handleGroup)

	if err := http.ListenAndServe(config.Server.Address, nil); err != nil {
		log.Fatalf("Server startup failed: %s\n", err)
	}
}

// may need to revisit if Okta makes concurrent requests printing could be broken across requests
func printHeaders(req *http.Request) {
	log.Println("> Request Headers")
	for k, v := range req.Header {
		fmt.Printf("%v : %v\n", k, v)
	}
	fmt.Println("")
}

func printBody(body []byte) {
	log.Println("> Request Body")
	fmt.Println(string(body))
	fmt.Println("")
}

func getBody(req *http.Request) ([]byte, error) {
	b, err := io.ReadAll(req.Body)
	if err != nil {
		log.Printf("Error reading Json Data: %v\n", err)
		return nil, err
	}

	defer req.Body.Close()
	return b, nil
}

func buildListResponse(docs []interface{}) v2.ListResponse {
	lr := v2.ListResponse{}
	lr.Schemas = append(lr.Schemas, v2.LIST_SCHEMA)
	lr.StartIndex = 1
	lr.TotalResults = len(docs)
	lr.ItemsPerPage = lr.TotalResults
	lr.Resources = []interface{}{}

	for _, v := range docs {
		if v == nil {
			continue
		}
		var m map[string]interface{}
		json.Unmarshal([]byte(fmt.Sprintf("%v", v)), &m)
		lr.Resources = append(lr.Resources, m)
	}
	return lr
}

func buildGroupListResponse(groups []map[string]interface{}) v2.ListResponse {
	lr := v2.ListResponse{}
	lr.Schemas = append(lr.Schemas, v2.LIST_SCHEMA)
	lr.StartIndex = 1
	lr.TotalResults = len(groups)
	lr.ItemsPerPage = lr.TotalResults
	lr.Resources = []interface{}{}

	for _, v := range groups {
		if v == nil {
			continue
		}
		lr.Resources = append(lr.Resources, v)
	}
	return lr
}

func handleErrorForKeyLookup(res *http.ResponseWriter, err error) {
	if err.Error() == NOT_FOUND {
		(*res).WriteHeader(http.StatusNotFound)
	} else {
		(*res).WriteHeader(http.StatusInternalServerError)
	}
	(*res).Write(nil)
}

func handleEmptyListReturn(res *http.ResponseWriter, err error) {
	if err.Error() == NOT_FOUND {
		lr := buildListResponse([]interface{}{})
		j, err := json.Marshal(&lr)
		if err != nil {
			log.Fatalf("Error Marshalling ListResponse: %v\n", err)
		}
		(*res).WriteHeader(http.StatusOK)
		(*res).Write(j)
	} else {
		(*res).WriteHeader(http.StatusInternalServerError)
		(*res).Write(nil)
	}
}

func handleErrorResponse(res *http.ResponseWriter, err string, status int) {
	e := v2.Error{
		Detail: err,
		Status: status,
	}
	e.Schemas = append(e.Schemas, v2.ERROR_SCHEMA)
	j, er := json.Marshal(&e)
	if er != nil {
		log.Fatalf("Error Marshalling Error: %v\n", er)
	}
	(*res).WriteHeader(status)
	(*res).Write(j)
}

func handleNotSupported(req *http.Request, res *http.ResponseWriter) {
	log.Printf("Method: %v, not supported for Path: %v\n", req.Method, req.URL.Path)
	(*res).WriteHeader(http.StatusMethodNotAllowed)
	(*res).Write(nil)
}
