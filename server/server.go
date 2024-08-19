package server

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"strings"

	"github.com/emanor-okta/go-scim/filters"
	messageLogs "github.com/emanor-okta/go-scim/server/log"
	"github.com/emanor-okta/go-scim/server/web"
	"github.com/emanor-okta/go-scim/types"
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

// var logMessages bool
var reqFilter utils.ReqFilter
var config *utils.Configuration

func StartServer(c *utils.Configuration) {
	config = c
	debugHeaders = config.Server.Debug_headers
	debugBody = config.Server.Debug_body
	debugQuery = config.Server.Debug_query
	//logMessages = config.Server.Log_messages
	log.Printf("starting server [%s], listening on %v\n", config.Build, config.Server.Address)
	configJson, _ := json.MarshalIndent(config, "", "  ")
	fmt.Printf("Server Config:\n%v\n", string(configJson))

	// SCIM server specific (needs message logging)
	scimMiddlewares := []types.Middleware{}
	// Non SCIM (used to filter IPs)
	commonScimMiddlewares := []types.Middleware{}

	// TODO - Add different auth middlewares
	// if debugBody {
	// 	// used to log to console
	// 	scimMiddlewares = append(scimMiddlewares, getBodyMiddleware)
	// }
	// if debugHeaders {
	// 	// used to log to console
	// 	scimMiddlewares = append(scimMiddlewares, getHeadersMiddleware)
	// }

	// If filtering IPs add middleware
	if config.Server.Filter_ips {
		scimMiddlewares = append(scimMiddlewares, filterIpMiddleware)
		commonScimMiddlewares = append(commonScimMiddlewares, filterIpMiddleware)
	}

	// used for logging messages to web console - Always init and add to middleware
	messageLogs.Init()
	scimMiddlewares = append(scimMiddlewares, logMessagesMiddleware, logMessageResponseSudoMiddleware)

	// if filtering IPs get Okta public IPs
	if config.Server.Filter_ips {
		config.Server.Allowed_ips = utils.GetOktaPublicIPs()
	}

	/*
		Route Handlers defined,
		1. here
		2. web/webHandlers.go
		3. web/proxyHandler.go
	*/
	// SCIM Server Handlers
	if config.Services.Scim {
		http.HandleFunc("/goscim/scim/v2/Users", utils.AddMiddleware(handleUsers, scimMiddlewares...))
		http.HandleFunc("/goscim/scim/v2/Users/", utils.AddMiddleware(handleUser, scimMiddlewares...))
		http.HandleFunc("/goscim/scim/v2/Groups", utils.AddMiddleware(handleGroups, scimMiddlewares...))
		http.HandleFunc("/goscim/scim/v2/Groups/", utils.AddMiddleware(handleGroup, scimMiddlewares...))
	}

	// Mock OAuth Server Handlers
	http.HandleFunc("/mock/oauth2/v1/authorize", utils.AddMiddleware(handleAuthorizeReq, commonScimMiddlewares...))
	http.HandleFunc("/mock/oauth2/v1/token", utils.AddMiddleware(handleTokenReq, commonScimMiddlewares...))

	// SSF Receiver Handlers  (no handlers in webHandlers.go)
	if config.Services.Ssf {
		// config.Server.Allowed_ips["[::1]"] = "blank"
		http.HandleFunc("/ssf/receiver", utils.AddMiddleware(handleSSFReq, commonScimMiddlewares...))
		// http.HandleFunc("/ssf/receiver", handleSSFReq)
		http.HandleFunc("/ssf/receiver/app", utils.AddMiddleware(handleSSFReciever, commonScimMiddlewares...))
		http.HandleFunc("/ssf/receiver/app/embed", utils.AddMiddleware(handleSSFRecieverAppEmbed, commonScimMiddlewares...))
		http.HandleFunc("/ssf/receiver/app/config", utils.AddMiddleware(handleSSFRecieverConfig, commonScimMiddlewares...))
		http.HandleFunc("/ssf/receiver/app/login", utils.AddMiddleware(handleSSFRecieverOauthLogin, commonScimMiddlewares...))
		http.HandleFunc("/ssf/receiver/app/callback", utils.AddMiddleware(handleSSFRecieverOauthCallback, commonScimMiddlewares...))
		http.HandleFunc("/ssf/receiver/app/ws", utils.AddMiddleware(handleSSFRecieverWebSocketUpgrade, commonScimMiddlewares...))
		http.HandleFunc("/ssf/transmitter/app", utils.AddMiddleware(handleSSFTransmitter, commonScimMiddlewares...))
	}

	// // Show Authorize page for unauthorized IP
	// http.HandleFunc("/authorizeMyIp", handleShowAuthorizeMyIp)
	// http.HandleFunc("/authorizeMyIpAuthorize", handleAuthorizeMyIp)

	/*
	 * Redirect testing. Currently PUT does not follow by the client
	 */
	handleRedirect := func(req *http.Request, res http.ResponseWriter) {
		var status int
		switch {
		case req.Method == http.MethodGet:
			status = http.StatusFound //302
		case req.Method == http.MethodHead:
			status = http.StatusFound
		default:
			status = http.StatusTemporaryRedirect //307
		}
		http.Redirect(res, req, strings.Replace(req.URL.RequestURI(), "/scim/v1", "/scim/v2", 1), status)
		//return strings.Replace(req.URL.RequestURI(), "/scim/v1", "/scim/v2", 1), status
	}
	http.HandleFunc("/scim/v1/Users", utils.AddMiddleware(func(res http.ResponseWriter, req *http.Request) {
		handleRedirect(req, res)
	}, commonScimMiddlewares...))
	http.HandleFunc("/scim/v1/Users/", utils.AddMiddleware(func(res http.ResponseWriter, req *http.Request) {
		handleRedirect(req, res)
	}, commonScimMiddlewares...))
	http.HandleFunc("/scim/v1/Groups", utils.AddMiddleware(func(res http.ResponseWriter, req *http.Request) {
		handleRedirect(req, res)
	}, commonScimMiddlewares...))
	http.HandleFunc("/scimmy/scim/v1/Groups/", utils.AddMiddleware(func(res http.ResponseWriter, req *http.Request) {
		handleRedirect(req, res)
	}, commonScimMiddlewares...))
	/*
	 * end redirect testing
	 */

	/*
	 * SET custome filter Here
	 */
	reqFilter = filters.DefaultFilter{}
	config.ReqFilter = &reqFilter

	// if running web console start it
	if config.Server.Web_console {
		web.StartWebServer(config, commonScimMiddlewares)
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

/*
 * Mock OAuth Server
 */
func handleAuthorizeReq(res http.ResponseWriter, req *http.Request) {
	log.Printf("Received Mock handle Authorize Request:\n%v\n", req.RequestURI)
	s := req.URL.Query().Get("state")
	r := req.URL.Query().Get("redirect_uri")
	redir := fmt.Sprintf("%s?code=123456&state=%s", r, s)
	http.Redirect(res, req, redir, http.StatusTemporaryRedirect)
}

func handleTokenReq(res http.ResponseWriter, req *http.Request) {
	err := req.ParseForm()
	if err != nil {
		log.Printf("Error handleTokenReq.ParseForm: %s\n", err)
	}
	log.Printf("Received Mock handle Token Request:\n%v\nclient_id: %s, client_secret: %s\n",
		req.RequestURI, req.Form.Get("client_id"), req.Form.Get("client_secret"))
	// TEST - dont return new refresh
	log.Printf("Grant Type: %s\n", req.Form.Get("grant_type"))
	var tRes any
	grantType := req.Form.Get("grant_type")
	if grantType == "refresh_token" {
		tRes = struct {
			Access_token string `json:"access_token"`
			Token_type   string `json:"token_type"`
			Expires_in   int    `json:"expires_in"`
			Scope        string `json:"scope"`
		}{
			"eyJhbCI6IkhTMjU2IiwidHlwIjoiSlciLCJhbGciOiJIUzI1NiJ9.eyJzIjoiMTIzNDU2Nzg5MCIsIm4iOiJKb2huIERvZSIsImkiOjE1MTYyMzkwMjJ9.fdErMOJ0QvrD9_vnj2Ih6trMx9cyDsY-mLntzjPFpOg",
			// "abcde",
			"Bearer",
			600,
			"scim",
		}
	} else {
		//
		//tRes := struct {
		tRes = struct {
			Access_token  string `json:"access_token"`
			Token_type    string `json:"token_type"`
			Expires_in    int    `json:"expires_in"`
			Scope         string `json:"scope"`
			Refresh_token string `json:"refresh_token"`
		}{
			"eyJhbCI6IkhTMjU2IiwidHlwIjoiSlciLCJhbGciOiJIUzI1NiJ9.eyJzIjoiMTIzNDU2Nzg5MCIsIm4iOiJKb2huIERvZSIsImkiOjE1MTYyMzkwMjJ9.fdErMOJ0QvrD9_vnj2Ih6trMx9cyDsY-mLntzjPFpOg",
			// "abcde",
			"Bearer",
			600,
			"scim offline_access",
			"mock_refresh_token_value",
		}
	}
	res.Header().Add("Content-Type", "application/json")
	b, _ := json.Marshal(tRes)
	res.Write(b)
	// res.WriteHeader(http.StatusInternalServerError)
}
