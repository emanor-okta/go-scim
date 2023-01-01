package server

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"

	"github.com/emanor-okta/go-scim/filters"
	messageLogs "github.com/emanor-okta/go-scim/server/log"
	"github.com/emanor-okta/go-scim/server/web"
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
var debugQuery bool
var logMessages bool
var reqFilter utils.ReqFilter

func StartServer(config *utils.Configuration) {
	debugHeaders = config.Server.Debug_headers
	debugBody = config.Server.Debug_body
	debugQuery = config.Server.Debug_query
	logMessages = config.Server.Log_messages
	log.Printf("starting server at %v\n", config.Server.Address)

	middlewares := []Middleware{}
	// TODO - Add different auth middlewares
	if debugBody {
		// used to log to console
		middlewares = append(middlewares, getBodyMiddleware)
	}
	if debugHeaders {
		// used to log to console
		middlewares = append(middlewares, getHeadersMiddleware)
	}
	if logMessages {
		// used for logging messages to web console
		messageLogs.Init(config)
		middlewares = append(middlewares, logMessagesMiddleware, logMessageResponseSudoMiddleware)
	}

	// TODO move to middlewre.go
	// messageLogs.Init(config)
	// middlewares = append(middlewares, func(h http.HandlerFunc) http.HandlerFunc {
	// 	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	// 		lrw := LoggerResponseWriter{RW: w, R: r}
	// 		h.ServeHTTP(lrw, r)
	// 	})
	// })

	http.HandleFunc("/scim/v2/Users", addMiddleware(handleUsers, middlewares...))
	http.HandleFunc("/scim/v2/Users/", addMiddleware(handleUser, middlewares...))
	http.HandleFunc("/scim/v2/Groups", addMiddleware(handleGroups, middlewares...))
	http.HandleFunc("/scim/v2/Groups/", addMiddleware(handleGroup, middlewares...))

	// http.HandleFunc("/scim/v1/Users", handleUsers)
	// http.HandleFunc("/scim/v1/Users/", handleUser)
	// http.HandleFunc("/scim/v1/Groups", handleGroups)
	// http.HandleFunc("/scim/v1/Groups/", handleGroup)

	/*
	 * SET custome filter Here
	 */
	reqFilter = filters.DefaultFilter{}
	config.ReqFilter = &reqFilter

	// if running web console start it
	if config.Server.Web_console {
		web.StartWebServer(config)
	}

	if err := http.ListenAndServe(config.Server.Address, nil); err != nil {
		log.Fatalf("Server startup failed: %s\n", err)
	}
}

// TESTING - how Okta handled a SCIM server 302 all requests
// func didRedirect(res *http.ResponseWriter, req *http.Request) bool {
// 	fmt.Println(req.URL)
// 	if strings.Contains(req.URL.Path, `/v1/`) {
// 		redir := "https://c0f2-2601-644-8f00-d4e0-75c2-52f6-159f-3085.ngrok.io" + strings.Replace(req.URL.Path, `/v1`, `/v2`, 1)
// 		fmt.Println(redir)
// 		(*res).Header().Add("Location", redir)
// 		(*res).WriteHeader(http.StatusPermanentRedirect)
// 		(*res).Write(nil)
// 		return true
// 	}
// 	return false
// }

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
	// lr.TotalResults = 0
	// lr.ItemsPerPage = 100

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

func handleErrorForKeyLookup(res *http.ResponseWriter, err error, id string) {
	if err.Error() == NOT_FOUND {
		// (*res).WriteHeader(http.StatusNotFound)
		handleErrorResponse(res, fmt.Sprintf("Resource %v not found", id), http.StatusNotFound)
	} else {
		// (*res).WriteHeader(http.StatusInternalServerError)
		handleErrorResponse(res, fmt.Sprintf("Server Error: %v", err.Error()), http.StatusInternalServerError)
	}
	// (*res).Write(nil)
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
