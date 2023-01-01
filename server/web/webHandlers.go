package web

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"path"

	"github.com/gorilla/websocket"

	"github.com/emanor-okta/go-scim/filters"
	"github.com/emanor-okta/go-scim/utils"
)

var tpl *template.Template
var config *utils.Configuration
var wsConn *websocket.Conn
var manualFilter filters.ManualFilter

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     nil, //func(r *http.Request) bool { return true },
}

func StartWebServer(c *utils.Configuration) {
	config = c
	tpl = template.Must(template.ParseGlob("server/web/templates/*"))

	http.HandleFunc("/messages", handleMessages)
	http.HandleFunc("/users", handleUsers)
	http.HandleFunc("/groups", handleGroups)
	http.HandleFunc("/filters/ws", handleWebSocketUpgrade)
	http.HandleFunc("/filters", handleFilters)
	http.HandleFunc("/config", handleConfig)
	http.HandleFunc("/js/ws.js", handleJavascript)
	// http.HandleFunc("/", handleMessages)

	// fmt.Printf("Starting Web Console on %v\n", config.Server.Web_address)
	// if err := http.ListenAndServe(config.Server.Web_address, nil); err != nil {
	// 	log.Fatalf("Server startup failed: %s\n", err)
	// }

	//TEST
	m := map[filterType]filter{}
	i := []instruction{}
	i = append(i, instruction{jsonPath: ".key2.inner_key2[1]", op: modify, value: "nothing"})
	i = append(i, instruction{jsonPath: ".", op: delete})
	i = append(i, instruction{jsonPath: ".keyArray[5].inner_keyO.arr[99]", op: modify, value: "arrayVal"})
	i = append(i, instruction{jsonPath: ".", op: modify, value: "{\"key\": \"val1\"}"})
	m[UserPostRequest] = filter{fType: UserPostRequest, instructions: i}
	GenerateSource(m)
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
	tpl.ExecuteTemplate(res, "messages.gohtml", nil)
}

func handleUsers(res http.ResponseWriter, req *http.Request) {
	fmt.Println("Returning Users")
	tpl.ExecuteTemplate(res, "users.gohtml", nil)
}

func handleGroups(res http.ResponseWriter, req *http.Request) {
	fmt.Println("Returning Groups")
	tpl.ExecuteTemplate(res, "groups.gohtml", nil)
}

func handleFilters(res http.ResponseWriter, req *http.Request) {
	fmt.Println("Returning Filters")
	tpl.ExecuteTemplate(res, "filters.gohtml", nil)
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
	manualFilter = filters.ManualFilter{Config: config, WsConn: wsConn, ReqMap: make(map[string]chan interface{}, 0)}
	*config.ReqFilter = &manualFilter
	manualFilter.Config.WebMessageFilter.UserPostRequest = true // !!!! TESTING

	log.Println("Client Connected")
	// var m map[string]interface{}
	// m = make(map[string]interface{}, 1)
	// m["id"] = 99
	// m["object"] = make(map[string]interface{})
	// m["object"].(map[string]interface{})["key1"] = "value 1"
	// m["object"].(map[string]interface{})["key2"] = [4]int{1, 2, 3, 4}
	// if err := conn.WriteJSON(m); err != nil {
	// 	log.Println(err)
	// }

	// listen indefinitely for new messages coming
	// through on our WebSocket connection
	// go writer(ws)
	go wsReader()
}

func handleConfig(res http.ResponseWriter, req *http.Request) {
	fmt.Println("Returning Config")
	tpl.ExecuteTemplate(res, "config.gohtml", nil)
}

func handleJavascript(res http.ResponseWriter, req *http.Request) {
	fmt.Println("Returning JavaScript")
	res.Header().Set("Content-Type", "application/javascript")
	fp := path.Join("server", "web", "js", "ws.js")
	http.ServeFile(res, req, fp)
	tpl.ExecuteTemplate(res, "config.gohtml", nil)
}

func wsReader() {
	for {
		// read in a message
		fmt.Println("WebSocket Reader about to block for Message")
		var m interface{}
		err := wsConn.ReadJSON(&m)
		if err != nil {
			log.Printf("wsConn.ReadJSON error: %v\n", err)
			if websocket.IsCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Println("Stopping Manual Filter, Setting Filter to Default Filter..")
				*config.ReqFilter = filters.DefaultFilter{}
				// create empty new filter to free up prior filter
				manualFilter = filters.ManualFilter{}
				return
			}
			continue
		}

		// fmt.Printf("message: %+v\n", m)
		uuid, ok := m.(map[string]interface{})["uuid"]
		if ok {
			ch := manualFilter.ReqMap[uuid.(string)]
			if ch != nil {
				ch <- m
			}
		}
	}
}
