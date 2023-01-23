package server

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
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
var config *utils.Configuration

func StartServer(c *utils.Configuration) {
	config = c
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

	// used for logging messages to web console - Always init and add to middleware
	messageLogs.Init()
	middlewares = append(middlewares, logMessagesMiddleware, logMessageResponseSudoMiddleware)

	http.HandleFunc("/scim/v2/Users", addMiddleware(handleUsers, middlewares...))
	http.HandleFunc("/scim/v2/Users/", addMiddleware(handleUser, middlewares...))
	http.HandleFunc("/scim/v2/Groups", addMiddleware(handleGroups, middlewares...))
	http.HandleFunc("/scim/v2/Groups/", addMiddleware(handleGroup, middlewares...))

	/*
	 * SET custome filter Here
	 */
	reqFilter = filters.DefaultFilter{}
	config.ReqFilter = &reqFilter

	// if running web console start it
	if config.Server.Web_console {
		web.StartWebServer(config)
	}

	// if err := http.ListenAndServe(config.Server.Address, nil); err != nil {
	// 	log.Fatalf("Server startup failed: %s\n", err)
	// }

	// hack to fix ngrok not reusing established connections (a guess)
	f := func(conn net.Conn, connState http.ConnState) {
		if connState == http.StateIdle {
			err := conn.Close()
			if err != nil {
				log.Printf("ConnState callback failed to close idle connection: %v\n", err)
			}
		}

	}
	s := &http.Server{
		Addr: config.Server.Address,
		// Handler:        myHandler,
		// ReadTimeout:    10 * time.Second,
		// WriteTimeout:   10 * time.Second,
		// MaxHeaderBytes: 1 << 20,
		ConnState: f,
	}
	log.Fatal(s.ListenAndServe())
	if err := s.ListenAndServe(); err != nil {
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

func handleEmptyListReturn(res *http.ResponseWriter, err error, filter *utils.ReqFilter, path string) {
	if err.Error() == NOT_FOUND {
		lr := buildListResponse([]interface{}{})
		(*filter).UserGetResponse(&lr, path)
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
