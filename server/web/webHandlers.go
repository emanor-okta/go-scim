package web

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"path"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"

	"github.com/emanor-okta/go-scim/filters"
	messageLogs "github.com/emanor-okta/go-scim/server/log"
	v2 "github.com/emanor-okta/go-scim/types/v2"
	"github.com/emanor-okta/go-scim/utils"
)

const (
	items_per_page          = 25
	items_per_page_messages = 100
)

var tpl *template.Template
var config *utils.Configuration
var wsConn *websocket.Conn
var manualFilter filters.ManualFilter
var filterMutex sync.Mutex
var filterId int

type PagePagination struct {
	Pagination   []int
	CurrentPage  int
	NextPage     int
	PreviousPage int
	PageCount    int
}

type UserTmpl struct {
	Username string
	Id       string
	Json     string
}

type UsersTmpl struct {
	Users []UserTmpl
	Count int
	PP    PagePagination
	Error error
}

type GroupTmpl struct {
	GroupName string
	Id        string
	Json      string
}

type GroupsTmpl struct {
	Groups []GroupTmpl
	Count  int
	PP     PagePagination
	Error  error
}

// type MessageTmpl struct {
// TimeStamp string
// Method    string
// Url       string
// Json      string

// }

type MessagessTmpl struct {
	Messages           []messageLogs.Message
	Count              int
	PP                 PagePagination
	Error              error
	Enabled            bool
	ProxyEnabled       bool
	ProxyPort          int
	ProxyOrigin        string
	SNI                string
	ProxySwitchEnabled bool
}

type ProxyFilterURLsTmpl struct {
	URLs []utils.ProxyFilterURL
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     nil, //func(r *http.Request) bool { return true },
}

func StartWebServer(c *utils.Configuration) {
	config = c
	tpl = template.Must(template.ParseGlob("server/web/templates/*"))

	http.HandleFunc("/messages", handleMessages)
	http.HandleFunc("/proxy", handleProxyMessages)
	http.HandleFunc("/users", handleUsers)
	http.HandleFunc("/users/update", handleUpdateUser)
	http.HandleFunc("/users/delete", handleDeleteUser)
	http.HandleFunc("/groups", handleGroups)
	http.HandleFunc("/groups/update", handleUpdateGroup)
	http.HandleFunc("/groups/delete", handleDeleteGroup)
	http.HandleFunc("/filters/ws", handleWebSocketUpgrade)
	http.HandleFunc("/filters", handleFilters)
	http.HandleFunc("/proxyfilter", handleProxyFilters)
	http.HandleFunc("/filters/toggle", handleToggleFilter)
	http.HandleFunc("/proxyfilter/toggle", handleProxyToggleFilter)
	// http.HandleFunc("/config", handleConfig)
	http.HandleFunc("/js/ws.js", handleJavascript)
	http.HandleFunc("/js/ui.js", handleJavascript)
	http.HandleFunc("/raw/user.json", handleRawJSON)
	http.HandleFunc("/raw/group.json", handleRawJSON)
	http.HandleFunc("/redis/flush", handleFlush)
	http.HandleFunc("/messages/flush", handleFlush)
	http.HandleFunc("/messages/toggle", handleToggleMessageLogging)
	http.HandleFunc("/proxy/toggle", handleToggleProxyLogging)
	http.HandleFunc("/har/generate", handleHarGeneration)

	// fmt.Printf("Starting Web Console on %v\n", config.Server.Web_address)
	// if err := http.ListenAndServe(config.Server.Web_address, nil); err != nil {
	// 	log.Fatalf("Server startup failed: %s\n", err)
	// }

	//TEST
	// m := map[filterType]filter{}
	// i := []instruction{}
	// i = append(i, instruction{jsonPath: ".key2.inner_key2[1]", op: modify, value: "nothing"})
	// i = append(i, instruction{jsonPath: ".", op: delete})
	// i = append(i, instruction{jsonPath: ".keyArray[5].inner_keyO.arr[99]", op: modify, value: "arrayVal"})
	// i = append(i, instruction{jsonPath: ".", op: modify, value: "{\"key\": \"val1\"}"})
	// m[UserPostRequest] = filter{fType: UserPostRequest, instructions: i}
	// GenerateSource(m)
	/*
		type instruction struct {
		jsonPath string
		op       opType
		value    interface{}
		}

		type filter struct {
			fType        filterType
			instructions []instruction
		}
	*/
}

func handleMessages(res http.ResponseWriter, req *http.Request) {
	fmt.Println("Returning Messages")
	getMessages(res, req, "messages.gohtml")
}

func handleToggleMessageLogging(res http.ResponseWriter, req *http.Request) {
	state, err := strconv.ParseBool(req.URL.Query().Get("enabled"))
	if err != nil {
		log.Printf("handleToggleMessageLogging.ParseBool() failed: %v\n", err)
		res.WriteHeader(500)
		res.Write(nil)
		return
	}

	log.Printf("Setting Message Logging to %v\n", state)
	config.Server.Log_messages = state
	res.WriteHeader(200)
	res.Write(nil)
}

func handleUsers(res http.ResponseWriter, req *http.Request) {
	fmt.Println("Returning Users")
	totalUserCount, err := utils.GetUserCount()
	if err != nil {
		usersTmpl := UsersTmpl{Error: err}
		tpl.ExecuteTemplate(res, "users.gohtml", usersTmpl)
		return
	}
	fmt.Println(totalUserCount)

	// if GET /scim/v2/users filter is set disable so below call to /scim/v2/users does not send GET mesg to
	// the browser blocking the page naviagtion to /users
	filterUsersResponse := config.WebMessageFilter.UserGetResponse
	config.WebMessageFilter.UserGetResponse = false
	page := req.URL.Query().Get("page")
	start, current := getPaginationPage(page, items_per_page)
	usersTmpl := getUsers(start, fmt.Sprintf("%d", items_per_page), req)
	usersTmpl.PP = computePagePagination(current, int(totalUserCount), items_per_page)
	if usersTmpl.Error != nil {
		usersTmpl := UsersTmpl{Error: usersTmpl.Error}
		tpl.ExecuteTemplate(res, "users.gohtml", usersTmpl)
		return
	}

	// after call to GET /scim/v2/users set filter back to it's original value
	config.WebMessageFilter.UserGetResponse = filterUsersResponse
	err = tpl.ExecuteTemplate(res, "users.gohtml", usersTmpl)
	if err != nil {
		log.Printf("Render Error: \"users.gohtml\": %v\n", err)
	}
}

func handleGroups(res http.ResponseWriter, req *http.Request) {
	fmt.Println("Returning Groups")
	totalGroupCount, err := utils.GetGroupCount()
	if err != nil {
		groupsTmpl := GroupsTmpl{Error: err}
		tpl.ExecuteTemplate(res, "groups.gohtml", groupsTmpl)
	}

	// if GET /scim/v2/groups filter is set disable so below call to /scim/v2/groups does not send GET mesg to
	// the browser blocking the page naviagtion to /groups
	filterGroupsResponse := config.WebMessageFilter.GroupsGetResponse
	config.WebMessageFilter.GroupsGetResponse = false

	page := req.URL.Query().Get("page")
	start, current := getPaginationPage(page, items_per_page)
	groupsTmpl := getGroups(start, fmt.Sprintf("%d", items_per_page), req)
	groupsTmpl.PP = computePagePagination(current, int(totalGroupCount), items_per_page)
	if groupsTmpl.Error != nil {
		groupsTmpl := UsersTmpl{Error: groupsTmpl.Error}
		tpl.ExecuteTemplate(res, "groups.gohtml", groupsTmpl)
		return
	}

	// after call to GET /scim/v2/groups set filter back to it's original value
	config.WebMessageFilter.GroupsGetResponse = filterGroupsResponse
	err = tpl.ExecuteTemplate(res, "groups.gohtml", groupsTmpl)
	if err != nil {
		log.Printf("Render Error: \"groups.gohtml\": %v\n", err)
	}
}

/*
SCIM Filters
*/
func handleFilters(res http.ResponseWriter, req *http.Request) {
	fmt.Println("Returning Filters")
	filterMutex.Lock()
	manualFilter = filters.ManualFilter{Config: config, WsConn: nil, ReqMap: make(map[string]chan interface{}, 0)}
	*config.ReqFilter = &manualFilter
	filterId++
	filterMutex.Unlock()

	err := tpl.ExecuteTemplate(res, "filters.gohtml", config.WebMessageFilter)
	if err != nil {
		fmt.Printf("%+v\n", err)
	}
}

/*
Proxy Filters
*/
func handleProxyFilters(res http.ResponseWriter, req *http.Request) {
	fmt.Println("Returning Proxy Filters")
	filterMutex.Lock()
	manualFilter = filters.ManualFilter{Config: config, WsConn: nil, ReqMap: make(map[string]chan interface{}, 0)}
	*config.ReqFilter = &manualFilter
	filterId++
	filterMutex.Unlock()

	Tmpl := ProxyFilterURLsTmpl{}
	//Tmpl.URLs = []ProxyFilterURL{}
	urls := []utils.ProxyFilterURL{}
	for _, v := range config.ProxyMessageFilter.FilterURLs {
		urls = append(urls, v)
	}
	// Tmpl.URLs = config.ProxyMessageFilter.FilterURLs
	Tmpl.URLs = urls

	// Tmpl.ResponseURLs = []ProxyFilterURL{}
	// for _, v := range config.WebMessageFilter.
	fmt.Println(config.ProxyMessageFilter.FilterMessages)
	// if config.ProxyMessageFilter.FilterMessages {
	// 	for key, value := range config.ProxyMessageFilter.FilterURLs {
	// 		Tmpl.URLs = append(Tmpl.URLs, getProxyFilterURL(key, value))
	// 	}
	// 	// for key, value := range config.ProxyMessageFilter.ResponseMessages {
	// 	// 	Tmpl.ResponseURLs = append(Tmpl.ResponseURLs, getProxyFilterURL(key, value))
	// 	// }
	// }
	fmt.Printf("%+v\n", Tmpl)
	err := tpl.ExecuteTemplate(res, "proxyfilter.gohtml", Tmpl)
	if err != nil {
		fmt.Printf("%+v\n", err)
	}
}

// func handleConfig(res http.ResponseWriter, req *http.Request) {
// 	fmt.Println("Returning Config")
// 	tpl.ExecuteTemplate(res, "config.gohtml", nil)
// }

func handleUpdateUser(res http.ResponseWriter, req *http.Request) {
	handleUpdate(res, req, fmt.Sprintf("%s%s/goscim/scim/v2/Users", getScheme(req.TLS, req.Host), req.Host))
}

func handleDeleteUser(res http.ResponseWriter, req *http.Request) {
	id := req.URL.Query().Get("id")
	err := sendDeleteToScim(fmt.Sprintf("%s%s/goscim/scim/v2/Users/%s", getScheme(req.TLS, req.Host), req.Host, id))
	if err != nil {
		log.Printf("handleDeleteUser() error: %v\n", err)
		res.WriteHeader(http.StatusInternalServerError)
		res.Write([]byte(err.Error()))
		return
	}

	res.WriteHeader(200)
}

func handleUpdateGroup(res http.ResponseWriter, req *http.Request) {
	handleUpdate(res, req, fmt.Sprintf("%s%s/goscim/scim/v2/Groups", getScheme(req.TLS, req.Host), req.Host))
}

func handleUpdate(res http.ResponseWriter, req *http.Request, url string) {
	id := req.URL.Query().Get("id")
	b, err := getBody(req)
	if err != nil {
		log.Printf("handleUpdateGroup error: %v\n", err)
		res.WriteHeader(http.StatusBadRequest)
		res.Write([]byte(err.Error()))
		return
	}

	method := "POST"
	if id != "" {
		url = fmt.Sprintf("%s/%s", url, id)
		method = "PUT"
	}

	err = sendUpdateToScim(url, method, string(b))
	if err != nil {
		log.Printf("handleUpdateGroup() error: %v\n", err)
		res.WriteHeader(http.StatusBadRequest)
		res.Write([]byte(err.Error()))
		return
	}

	res.WriteHeader(200)
}

func handleDeleteGroup(res http.ResponseWriter, req *http.Request) {
	id := req.URL.Query().Get("id")
	err := sendDeleteToScim(fmt.Sprintf("%s%s/goscim/scim/v2/Groups/%s", getScheme(req.TLS, req.Host), req.Host, id))
	if err != nil {
		log.Printf("handleDeleteGroup() error: %v\n", err)
		res.WriteHeader(http.StatusInternalServerError)
		res.Write([]byte(err.Error()))
		return
	}

	res.WriteHeader(200)
}

func handleFlush(res http.ResponseWriter, req *http.Request) {
	epoch := req.URL.Query().Get("epoch")
	epoch64, err := strconv.ParseInt(epoch, 10, 64)
	if err != nil {
		log.Panicf("Error converting epoch query param to int64: %v\n", err)
		epoch64 = 0
	}

	now := time.Now()
	if now.UnixMilli()-epoch64 < 30000 {
		//log.Printf("PATH: %s\n", req.URL.Path)
		if req.URL.Path == "/redis/flush" {
			log.Println("Flushing Redis")
			err := utils.FlushDB()
			if err != nil {
				res.WriteHeader(http.StatusForbidden)
			} else {
				res.WriteHeader(http.StatusOK)
			}
		} else if req.URL.Path == "/messages/flush" {
			log.Println("Flushing Messages")
			messageLogs.FlushMessages()
			res.WriteHeader(http.StatusOK)
		}
	} else {
		res.WriteHeader(http.StatusForbidden)
	}

	res.Write(nil)
}

func handleHarGeneration(res http.ResponseWriter, req *http.Request) {
	harType := req.URL.Query().Get("type")
	bytes := messageLogs.GetHar(harType)
	res.Header().Set("Content-Type", "application/octet-stream")
	res.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s.har", harType))
	res.Header().Set("Content-Length", strconv.Itoa(len(bytes)))
	res.Header().Set("Cache-Control", "private")
	res.Header().Set("Pragma", "private")
	//res.Header().Set("Expires", "Mon, 26 Jul 2024 05:00:00 GMT")
	res.WriteHeader(http.StatusOK)
	res.Write(bytes)
}

func handleWebSocketUpgrade(res http.ResponseWriter, req *http.Request) {
	// upgrade this connection to a WebSocket
	// connection
	conn, err := upgrader.Upgrade(res, req, nil)
	if err != nil {
		log.Printf("upgrader.Upgrade() err: %v\n", err)
		res.WriteHeader(http.StatusInternalServerError)
		res.Write(nil)
		return
	}

	wsConn = conn
	// manualFilter = filters.ManualFilter{Config: config, WsConn: wsConn, ReqMap: make(map[string]chan interface{}, 0)}
	// *config.ReqFilter = &manualFilter

	(*config.ReqFilter).(*filters.ManualFilter).WsConn = wsConn
	log.Println("Client Connected")
	// listen indefinitely for new messages coming
	// through on our WebSocket connection
	go wsReader()
}

func handleJavascript(res http.ResponseWriter, req *http.Request) {
	//fmt.Println(req.URL.Path)
	res.Header().Set("Content-Type", "application/javascript")
	var fp string
	if req.URL.Path == "/js/ws.js" {
		fp = path.Join("server", "web", "js", "ws.js")
	} else {
		fp = path.Join("server", "web", "js", "ui.js")
	}

	http.ServeFile(res, req, fp)
	tpl.ExecuteTemplate(res, "config.gohtml", nil)
}

func handleRawJSON(res http.ResponseWriter, req *http.Request) {
	//fmt.Println(req.URL.Path)
	res.Header().Set("Content-Type", "application/json")
	var fp string
	if req.URL.Path == "/raw/user.json" {
		fp = path.Join("server", "web", "raw", "user.json")
	} else {
		fp = path.Join("server", "web", "raw", "group.json")
	}

	http.ServeFile(res, req, fp)
	//tpl.ExecuteTemplate(res, "config.gohtml", nil)
}

func handleToggleFilter(res http.ResponseWriter, req *http.Request) {
	reqType := req.URL.Query().Get("requestType")
	state, err := strconv.ParseBool(req.URL.Query().Get("enabled"))
	if err != nil {
		log.Printf("handleToggleFilter.ParseBool() failed: %v\n", err)
		res.WriteHeader(500)
		res.Write(nil)
		return
	}

	manualFilter.ToggleFilter(reqType, state)
	res.WriteHeader(200)
	res.Write(nil)
}

func handleProxyToggleFilter(res http.ResponseWriter, req *http.Request) {
	log.Printf("handleProxyToggleFilter, request: %s %s\n", req.Method, req.RequestURI)
	if req.Method == http.MethodPost {
		b, err := getBody(req)
		if err != nil {
			log.Printf("handleProxyToggleFilter: Error getting POST body: %v\n", err)
			res.WriteHeader(http.StatusBadRequest)
			return
		}
		var msg utils.ProxyFilterURL
		if err := json.Unmarshal(b, &msg); err != nil {
			log.Printf("handleProxyToggleFilter: Error decoding Json Data: %v\n", err)
			res.WriteHeader(http.StatusBadRequest)
			return
		}
		config.ProxyMessageFilter.FilterURLs[msg.URL] = msg
		fmt.Printf("%+v\n", config.ProxyMessageFilter.FilterURLs)
		res.WriteHeader(http.StatusOK)
	} else if req.Method == http.MethodDelete {
		path := req.URL.Query().Get("path")
		delete(config.ProxyMessageFilter.FilterURLs, path)
	}
}

func wsReader() {
	filterMutex.Lock()
	filterId_ := filterId
	filterMutex.Unlock()

	for {
		// read in a message
		fmt.Println("WebSocket Reader about to block for Message")
		var m interface{}
		err := wsConn.ReadJSON(&m)
		if err != nil {
			log.Printf("wsConn.ReadJSON error: %v\n", err)
			if websocket.IsCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				filterMutex.Lock()
				// fmt.Printf("filterId %v, filterId_ %v\n", filterId, filterId_)
				if filterId == filterId_ {
					log.Println("Stopping Manual Filter")
					*config.ReqFilter = filters.DefaultFilter{}
					// create empty new filter to free up prior filter
					// manualFilter = filters.ManualFilter{}
				}
				filterMutex.Unlock()
				return
			}
			continue
		}

		fmt.Printf("message: %+v\n", m)
		uuid, ok := m.(map[string]interface{})["uuid"]
		// fmt.Printf("uuid: %v, v: %v\n", uuid.(string), manualFilter.ReqMap[uuid.(string)])
		if ok {
			ch := manualFilter.ReqMap[uuid.(string)]
			if ch != nil {
				ch <- m
			}
		}
	}
}

func getUsers(start, count string, req *http.Request) UsersTmpl {
	ut := UsersTmpl{}
	lr, err := getListResponseResource(fmt.Sprintf("%s%s/goscim/scim/v2/Users?startIndex=%s&count=%s", getScheme(req.TLS, req.Host), req.Host, start, count))
	if err != nil {
		ut.Error = err
		return ut
	}
	ut.Count = lr.TotalResults
	for _, v := range lr.Resources {
		userName := v.(map[string]interface{})["userName"]
		id := v.(map[string]interface{})["id"]
		user, _ := json.MarshalIndent(v, "", "  ")
		ut.Users = append(ut.Users, UserTmpl{Username: userName.(string), Id: id.(string), Json: string(user)})
	}

	return ut
}

func getGroups(start, count string, req *http.Request) GroupsTmpl {
	gt := GroupsTmpl{}
	lr, err := getListResponseResource(fmt.Sprintf("%s%s/goscim/scim/v2/Groups?startIndex=%s&count=%s", getScheme(req.TLS, req.Host), req.Host, start, count))
	if err != nil {
		gt.Error = err
		return gt
	}
	gt.Count = lr.TotalResults
	for _, v := range lr.Resources {
		displayName := v.(map[string]interface{})["displayName"]
		id := v.(map[string]interface{})["id"]
		group, _ := json.MarshalIndent(v, "", "  ")
		gt.Groups = append(gt.Groups, GroupTmpl{GroupName: displayName.(string), Id: id.(string), Json: string(group)})
	}
	return gt
}

func getListResponseResource(url string) (*v2.ListResponse, error) {
	res, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	if res.StatusCode >= 200 && res.StatusCode < 300 {
		defer res.Body.Close()
		var lResp v2.ListResponse
		b, _ := io.ReadAll(res.Body)
		err = json.Unmarshal(b, &lResp)
		if err != nil {
			return nil, err
		}
		return &lResp, nil
	} else {
		return nil, fmt.Errorf("%v", res.Status)
	}
}

func sendUpdateToScim(url, method, msg string) error {
	fmt.Println(url)
	req, _ := http.NewRequest(method, url, bytes.NewBuffer([]byte(msg)))
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}

	defer resp.Body.Close()
	body, _ := ioutil.ReadAll(resp.Body)
	if resp.StatusCode >= 300 {
		log.Printf("sendUpdateToScim() error: %v\n", string(body))
		return fmt.Errorf("%q", string(body))
	}

	return nil
}

func sendDeleteToScim(url string) error {
	req, _ := http.NewRequest("DELETE", url, nil)
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, _ := ioutil.ReadAll(resp.Body)
	if resp.StatusCode >= 300 {
		log.Printf("sendDeleteToScim() error: %v\n", string(body))
		return fmt.Errorf("%q", string(body))
	}

	return nil
}

func getMessages(res http.ResponseWriter, req *http.Request, template string) {
	page := req.URL.Query().Get("page")
	start, current := getPaginationPage(page, items_per_page_messages)
	i, _ := strconv.Atoi(start)
	messages, totalMessages := messageLogs.GetMessages(i-1, items_per_page_messages, template)
	// fmt.Printf("totalMessage: %v, messages: \n%+v\n", totalMessages, messages)
	messagesTmpl := MessagessTmpl{Messages: messages}
	messagesTmpl.Count = len(messages)
	messagesTmpl.PP = computePagePagination(current, totalMessages, items_per_page_messages)
	messagesTmpl.Enabled = config.Server.Log_messages
	messagesTmpl.ProxyEnabled = config.Server.Proxy_messages
	messagesTmpl.ProxyPort = config.Server.Proxy_port
	messagesTmpl.ProxyOrigin = config.Server.Proxy_origin
	messagesTmpl.SNI = config.Server.Proxy_sni
	if config.Server.Proxy_address != "" && config.Server.Proxy_port > 0 {
		messagesTmpl.ProxySwitchEnabled = true
	} else {
		messagesTmpl.ProxySwitchEnabled = false
	}

	err := tpl.ExecuteTemplate(res, template, messagesTmpl)
	if err != nil {
		log.Printf(`Render Error: "%s": %v\n`, template, err)
	}
}

func computePagePagination(currentPage, itemCount, itemsPerPage int) PagePagination {
	pp := PagePagination{CurrentPage: currentPage}
	pp.NextPage = currentPage + 1
	pp.PreviousPage = currentPage - 1

	pp.PageCount = int(itemCount / itemsPerPage)
	if int(itemCount%itemsPerPage) > 0 {
		pp.PageCount++
	}

	if pp.PageCount <= 20 {
		for i := 1; i <= pp.PageCount; i++ {
			pp.Pagination = append(pp.Pagination, i)
		}
	} else {
		for i := pp.CurrentPage; i <= pp.PageCount && i <= pp.CurrentPage+8; i++ {
			pp.Pagination = append(pp.Pagination, i)
		}
		if pp.CurrentPage+8 < pp.PageCount {
			pp.Pagination = append(pp.Pagination, pp.PageCount)
		}
		if pp.CurrentPage > 1 {
			a := []int{}
			for i := pp.CurrentPage - 1; i > 1 && i > pp.CurrentPage-9; i-- {
				a = append(a, i)
			}
			a = append(a, 1)
			for i, j := 0, len(a)-1; i < j; i, j = i+1, j-1 {
				a[i], a[j] = a[j], a[i]
			}
			pp.Pagination = append(a, pp.Pagination...)
		}
	}

	return pp
}

func getPaginationPage(page string, itemsPerPAge int) (string, int) {
	start := "1"
	current := 1
	if page == "" || page == "1" {
		page = "1"
	} else {
		i, err := strconv.Atoi(page)
		if err != nil {
			log.Printf("getPaginationPage.strconv.Atoi(page) Error: %v\n", err)
		} else {
			current = i
			i = (i-1)*itemsPerPAge + 1
			start = fmt.Sprintf("%d", i)
		}
	}

	return start, current
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

func getScheme(tls *tls.ConnectionState, host string) string {
	// is returning nil even when https - or because SSL termination and http from ALB to docker
	/*
		if tls == nil {
			return "http://"
		} else {
			return "https://"
		}
	*/
	if strings.Contains(host, "localhost") {
		return "http://"
	} else {
		return "https://"
	}
}
