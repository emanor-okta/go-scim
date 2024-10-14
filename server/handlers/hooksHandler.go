package handlers

import (
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"

	"github.com/gorilla/websocket"

	"github.com/emanor-okta/go-scim/utils"
)

const (
	token          = "com.okta.oauth2.tokens.transform"
	saml           = "com.okta.saml.tokens.transform"
	passwordImport = "com.okta.user.credential.password.import"
	registration   = "com.okta.user.pre-registration"
	userImport     = "com.okta.import.transform"
	telephony      = "com.okta.telephony.provider"
	_logPrefix     = "server.handlers.hooksHandler."
)

type wsPayload struct {
	Request  map[string]interface{} `json:"request,omitempty"`
	Response map[string]interface{} `json:"response,omitempty"`
	Type     string                 `json:"type,omitempty"`
}

var wsClientConnected bool
var wsConn *websocket.Conn

var tpl *template.Template

func init() {
	wsClientConnected = false
	tpl = template.Must(template.ParseGlob("server/web/templates/*"))
}

/*
Server Handlers
*/
func HandleHookRequest(res http.ResponseWriter, req *http.Request) {
	log.Printf("%shandleHookRequest: Received Hook Request:\n", _logPrefix)
	// fmt.Printf("%+v\n", req)

	if req.Method == http.MethodGet {
		verificationValue := req.Header.Get("x-okta-verification-challenge")
		if verificationValue != "" {
			data := struct {
				Verification string `json:"verification,omitempty"`
			}{Verification: verificationValue}
			b, _ := json.Marshal(data)

			_, err := res.Write(b)
			if err != nil {
				log.Printf("%shandleHookRequest: Error sending verification response, error: %+v\n", _logPrefix, err)
			}
		}
	} else if req.Method == http.MethodPost {
		body, err := utils.GetBody(req)
		if err != nil {
			log.Printf("%shandleHookRequest: Error: %+v\n", _logPrefix, err)
			res.WriteHeader(http.StatusOK)
			return
		}

		m := map[string]interface{}{}
		err = json.Unmarshal(body, &m)
		if err != nil {
			log.Printf("%shandleHookRequest: Unmarshal Error: %+v\n", _logPrefix, err)
			res.WriteHeader(http.StatusOK)
			return
		}
		// fmt.Printf("%+v\n", m)
		eventType := m["eventType"].(string)
		fmt.Printf("EventType: %+v\n", eventType)
		switch eventType {
		case token:
			sendToUI(m, []byte(utils.Config.Hooks.Token), "Token Inline Hook", res)
		case saml:
			sendToUI(m, []byte(utils.Config.Hooks.Saml), "SAML Inline Hook", res)
		case passwordImport:
			sendToUI(m, []byte(utils.Config.Hooks.Password), "Password Import Inline Hook", res)
		case registration:
			sendToUI(m, []byte(utils.Config.Hooks.Registration), "Registration Inline Hook", res)
		case userImport:
			sendToUI(m, []byte(utils.Config.Hooks.UserImport), "User Import Inline Hook", res)
		case telephony:
			sendToUI(m, []byte(utils.Config.Hooks.Telephony), "Telephony Inline Hook", res)
		default:
			sendToUI(m, []byte("{}"), fmt.Sprintf("Event Hook %s", eventType), res)
		}

	} else {
		log.Printf("%shandleHookRequest: Unexpected HTTP Method: %s\n", _logPrefix, req.Method)
		res.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func HandleInlineHookRequest(res http.ResponseWriter, req *http.Request) {
	err := tpl.ExecuteTemplate(res, "inlinehook.gohtml", struct {
		utils.Services
		utils.Hooks
	}{utils.Config.Services, utils.Config.Hooks})
	if err != nil {
		log.Printf("handleSSFReciever: %+v\n", err)
	}
}

func HandleHookWebSocketUpgrade(res http.ResponseWriter, req *http.Request) {
	wsConn = utils.HandleWebSocketUpgrade(res, req, &wsClientConnected)
	if wsClientConnected && wsConn != nil {
		go utils.WsPingOnlyReader(wsConn, &wsClientConnected)
	}
}

func HandleHookConfig(res http.ResponseWriter, req *http.Request) {
	hookType := req.URL.Query().Get("type")
	if req.Method == http.MethodPost {
		body, err := utils.GetBody(req)
		if err != nil || hookType == "" {
			log.Printf("%sHandleHookConfig: HookType:%s, Error: %+v\n", _logPrefix, hookType, err)
			res.WriteHeader(http.StatusBadRequest)
			return
		}
		// fmt.Println(string(body))
		switch hookType {
		case "token":
			utils.Config.Hooks.Token = string(body)
		case "saml":
			utils.Config.Hooks.Saml = string(body)
		case "password":
			utils.Config.Hooks.Password = string(body)
		case "registration":
			utils.Config.Hooks.Registration = string(body)
		case "import":
			utils.Config.Hooks.UserImport = string(body)
		case "telephony":
			utils.Config.Hooks.Telephony = string(body)
		default:
			log.Printf("%sHandleHookConfig: HookType:%s Invalid\n", _logPrefix, hookType)
		}
	} else if req.Method == http.MethodGet {
		switch hookType {
		case "token":
			res.Write([]byte(utils.Config.Hooks.Token))
		case "saml":
			res.Write([]byte(utils.Config.Hooks.Saml))
		case "password":
			res.Write([]byte(utils.Config.Hooks.Password))
		case "registration":
			res.Write([]byte(utils.Config.Hooks.Registration))
		case "import":
			res.Write([]byte(utils.Config.Hooks.UserImport))
		case "telephony":
			res.Write([]byte(utils.Config.Hooks.Telephony))
		default:
			log.Printf("%sHandleHookConfig: HookType:%s Invalid\n", _logPrefix, hookType)
		}
	} else {
		res.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
}

/*
Helpers
*/
func sendToUI(hookRequest map[string]interface{}, hookResponse []byte, hookType string, res http.ResponseWriter) {
	if wsClientConnected {
		m := map[string]interface{}{}
		err := json.Unmarshal(hookResponse, &m)
		if err != nil {
			log.Printf("%ssendToUI: Error UnMarshal Hook Response: %s\n", _logPrefix, err)
		}

		payload := wsPayload{
			Request:  hookRequest,
			Response: m,
			Type:     hookType,
		}
		err = wsConn.WriteJSON(payload)
		if err != nil {
			log.Printf("%ssendToUI: Error sending WS message: %s\n", _logPrefix, err)
		}
	}

	b, _ := json.MarshalIndent(hookRequest, "", "  ")
	fmt.Printf("%s Hook Received:\nRequest:%+v\nResponse:%+v\n", hookType, string(b), string(hookResponse))
	_, err := res.Write(hookResponse)
	if err != nil {
		log.Fatalf("%+v\n", err)
	}
}

func LoadDefaultResponses() {
	v, _ := os.ReadFile("./server/web/raw/hooks/token.json")
	utils.Config.Hooks.Token = string(v)
	v, _ = os.ReadFile("./server/web/raw/hooks/password.json")
	utils.Config.Hooks.Password = string(v)
	v, _ = os.ReadFile("./server/web/raw/hooks/registration.json")
	utils.Config.Hooks.Registration = string(v)
	v, _ = os.ReadFile("./server/web/raw/hooks/saml.json")
	utils.Config.Hooks.Saml = string(v)
	v, _ = os.ReadFile("./server/web/raw/hooks/telephony.json")
	utils.Config.Hooks.Telephony = string(v)
	v, _ = os.ReadFile("./server/web/raw/hooks/import.json")
	utils.Config.Hooks.UserImport = string(v)
}
