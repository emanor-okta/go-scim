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
	"github.com/emanor-okta/go-scim/server/handlers"
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
	// Used for Proxy only
	proxyMiddlewares := []types.Middleware{}

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

	// Always Init Proxy with filterIpMiddleware
	proxyMiddlewares = append(proxyMiddlewares, filterProxyIpMiddleware)
	web.InitProxy(proxyMiddlewares)

	// used for logging messages to web console - Always init and add to middleware
	messageLogs.Init()
	scimMiddlewares = append(scimMiddlewares, logMessagesMiddleware, logMessageResponseSudoMiddleware)

	c.CommonScimMiddlewares = commonScimMiddlewares

	/*
		Route Handlers defined,
		1. here
		2. web/webHandlers.go
		3. web/proxyHandler.go
		4. web/scimHandlers.go
	*/
	// SCIM Server Handlers
	if config.Services.Scim {
		http.HandleFunc("/goscim/scim/v2/Users", utils.AddMiddleware(handleUsers, scimMiddlewares...))
		http.HandleFunc("/goscim/scim/v2/Users/", utils.AddMiddleware(handleUser, scimMiddlewares...))
		http.HandleFunc("/goscim/scim/v2/Groups", utils.AddMiddleware(handleGroups, scimMiddlewares...))
		http.HandleFunc("/goscim/scim/v2/Groups/", utils.AddMiddleware(handleGroup, scimMiddlewares...))
		http.HandleFunc("/goscim/scim/v2/ServiceProviderConfig", utils.AddMiddleware(handleServiceProviderConfig, scimMiddlewares...))

		http.HandleFunc("/goscim/scim/v2/ResourceTypes", utils.AddMiddleware(handleResourceTypes, scimMiddlewares...))
		http.HandleFunc("/goscim/scim/v2/ResourceTypes/", utils.AddMiddleware(handleResourceType, scimMiddlewares...))
		http.HandleFunc("/goscim/scim/v2/Schemas", utils.AddMiddleware(handleSchemas, scimMiddlewares...))

		// hack for OPP
		// http.HandleFunc("/goscim/v2/scim/Users", utils.AddMiddleware(handleUsers, scimMiddlewares...))
		// http.HandleFunc("/goscim/v2/scim/Users/", utils.AddMiddleware(handleUser, scimMiddlewares...))
		// http.HandleFunc("/goscim/v2/scim/Groups", utils.AddMiddleware(handleGroups, scimMiddlewares...))
		// http.HandleFunc("/goscim/v2/scim/Groups/", utils.AddMiddleware(handleGroup, scimMiddlewares...))
		// http.HandleFunc("/goscim/v2/scim/ServiceProviderConfig", utils.AddMiddleware(handleServiceProviderConfig, scimMiddlewares...))
	}

	// Mock OAuth Server Handlers
	http.HandleFunc("/mock/oauth2/v1/authorize", utils.AddMiddleware(handleAuthorizeReq, commonScimMiddlewares...))
	http.HandleFunc("/mock/oauth2/v1/token", utils.AddMiddleware(handleTokenReq, commonScimMiddlewares...))

	// SSF Receiver/Transmitter Handlers  (no handlers in webHandlers.go)
	if config.Services.Ssf {
		// config.Server.Allowed_ips["[::1]"] = "blank"
		http.HandleFunc("/ssf/receiver", utils.AddMiddleware(handleSSFReq, commonScimMiddlewares...))
		http.HandleFunc("/ssf/globalLogout", utils.AddMiddleware(handleGlobalLogout, commonScimMiddlewares...))
		// http.HandleFunc("/ssf/receiver", handleSSFReq)
		http.HandleFunc("/ssf/receiver/app", utils.AddMiddleware(handleSSFReciever, commonScimMiddlewares...))
		http.HandleFunc("/ssf/receiver/app/embed", utils.AddMiddleware(handleSSFRecieverAppEmbed, commonScimMiddlewares...))
		http.HandleFunc("/ssf/receiver/app/config", utils.AddMiddleware(handleSSFRecieverConfig, commonScimMiddlewares...))
		http.HandleFunc("/ssf/receiver/app/login", utils.AddMiddleware(handleSSFRecieverOauthLogin, commonScimMiddlewares...))
		http.HandleFunc("/ssf/receiver/app/callback", utils.AddMiddleware(handleSSFRecieverOauthCallback, commonScimMiddlewares...))
		http.HandleFunc("/ssf/receiver/app/ws", utils.AddMiddleware(handleSSFRecieverWebSocketUpgrade, commonScimMiddlewares...))
		http.HandleFunc("/ssf/transmitter/app", utils.AddMiddleware(handleSSFTransmitter, commonScimMiddlewares...))
		http.HandleFunc("/ssf/transmitter/keys", utils.AddMiddleware(handleSSFTransmitterKeys, commonScimMiddlewares...))
		http.HandleFunc("/ssf/transmitter/.well-known/sse-configuration", utils.AddMiddleware(handleSSFTransmitterConfig, commonScimMiddlewares...))
		http.HandleFunc("/ssf/transmitter/event/", utils.AddMiddleware(handleGetSecurityEventType, scimMiddlewares...))
		http.HandleFunc("/ssf/transmitter/send", utils.AddMiddleware(handleSendSecurityEvents, scimMiddlewares...))
	}

	// Hooks Handlers (no handlers in webHandlers.go)
	if config.Services.Hooks {
		http.HandleFunc("/hooks/service", utils.AddMiddleware(handlers.HandleHookRequest, utils.Config.CommonScimMiddlewares...))
		http.HandleFunc("/hooks/inline", utils.AddMiddleware(handlers.HandleInlineHookRequest, utils.Config.CommonScimMiddlewares...))
		http.HandleFunc("/hooks/inline/ws", utils.AddMiddleware(handlers.HandleHookWebSocketUpgrade, utils.Config.CommonScimMiddlewares...))
		http.HandleFunc("/hooks/inline/config", utils.AddMiddleware(handlers.HandleHookConfig, utils.Config.CommonScimMiddlewares...))
		handlers.LoadDefaultResponses()
	}

	// DPoP / JWT Handlers (no handlers in webHandlers.go)
	if config.Services.Dpop {
		http.HandleFunc("/dpop/callback", utils.AddMiddleware(handlers.HandleCallbackReq, utils.Config.CommonScimMiddlewares...))
		http.HandleFunc("/dpop/generate_dpop", utils.AddMiddleware(handlers.HandleGenerateDpop, utils.Config.CommonScimMiddlewares...))
		http.HandleFunc("/dpop", utils.AddMiddleware(handlers.HandleDpop, utils.Config.CommonScimMiddlewares...))
		// http.HandleFunc("/dpop/upload_priv_key", utils.AddMiddleware(handlers.HandleDpopKeyUpload, utils.Config.CommonScimMiddlewares...))
		// http.HandleFunc("/dpop/upload_dpop_key", utils.AddMiddleware(handlers.HandleDpopKeyUpload, utils.Config.CommonScimMiddlewares...))
		http.HandleFunc("/dpop/jwt-config", utils.AddMiddleware(handlers.HandleDpopKeyUpload, utils.Config.CommonScimMiddlewares...))
		http.HandleFunc("/dpop/service-config", utils.AddMiddleware(handlers.HandleDpopKeyUpload, utils.Config.CommonScimMiddlewares...))
		http.HandleFunc("/dpop/auth-config", utils.AddMiddleware(handlers.HandleDpopKeyUpload, utils.Config.CommonScimMiddlewares...))
		http.HandleFunc("/dpop/removekey", utils.AddMiddleware(handlers.HandleDpopKeyRemoval, utils.Config.CommonScimMiddlewares...))
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
	// log.Fatal(s.ListenAndServe())
	// if err := s.ListenAndServe(); err != nil {
	// 	log.Fatalf("Server startup failed: %s\n", err)
	// }
	ch := make(chan int)
	go listen(s, ch)

	// if filtering IPs get Okta public IPs and local IPs (calls made to self)
	if config.Server.Filter_ips {
		addAllowedIps()
	}

	exit := <-ch
	log.Printf("Server shutting down: %v\n", exit)
}

func listen(s *http.Server, ch chan int) {
	if err := s.ListenAndServe(); err != nil {
		log.Printf("Server stopped: %s\n", err)
		ch <- 0
	}
}

func addAllowedIps() {
	config.Server.Allowed_ips = utils.GetOktaPublicIPs()
	// Get local IPs
	localIps := utils.GetLocalIps()
	for _, ip_ := range localIps {
		config.Server.Allowed_ips[ip_] = "local-server-ip"
	}
	/*
		having issues getting the actual IP that will be shown as x-forwarded or remoteAddress when running
		in Docker on AWS. Use below workaround for now
	*/
	whoAmIId := utils.GenerateUUID()
	http.HandleFunc("/whoami", func(w http.ResponseWriter, r *http.Request) {
		id := r.URL.Query().Get("whoAmIId")
		fmt.Printf("%s = %s\n", id, whoAmIId)
		if id == whoAmIId {
			config.Server.Allowed_ips[utils.GetRemoteAddress(r)] = "whoami-ip"
			for i := 0; i < 10; i++ {
				whoAmIId = fmt.Sprintf("%s%s", whoAmIId, utils.GenerateUUID())
			}
			w.WriteHeader(http.StatusOK)
		} else {
			http.Redirect(w, r, "/messages", http.StatusTemporaryRedirect)
		}
	})

	req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("%s/whoami?whoAmIId=%s", config.Server.Public_address, whoAmIId), nil)
	client := &http.Client{}
	res, err := client.Do(req)
	if err == nil {
		defer res.Body.Close()
	} else {
		log.Printf("server.server.StartServer(): Error calling /whoami, %+v\n", err)
	}

	utils.DebugAllowedIPs(config.Server.Allowed_ips)
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
