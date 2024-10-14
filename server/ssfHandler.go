package server

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strings"

	"github.com/gorilla/sessions"
	"github.com/gorilla/websocket"
	"github.com/lestrrat-go/jwx/v2/jwa"

	"github.com/emanor-okta/go-scim/types"
	"github.com/emanor-okta/go-scim/types/ssf"
	"github.com/emanor-okta/go-scim/utils"
)

// const TokenCall = "client_id=%s&client_secret=%s&grant_type=authorization_code&redirect_uri=%s&code=%s"
// const AuthorizeCall = "client_id=%s&response_type=code&scope=%s&redirect_uri=%s&state=%s"

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     nil, //func(r *http.Request) bool { return true },
}

var (
	key   = []byte("super-secret-key")
	store = sessions.NewCookieStore(key)
)

var wsConn *websocket.Conn
var wsClientConnected bool
var oauthConfig types.OauthConfig
var tpl *template.Template
var sessionsMap map[string]ssf.SsfReceiverAppData

func init() {
	// hardcode for now
	oauthConfig = types.OauthConfig{
		Issuer:       "https://emanor-oie.oktapreview.com/oauth2/default",
		ClientId:     "0oa2cpl777xczKzL21d7",
		ClientSecret: "Klf03TzBqEuayATkPGy7VgTqyNDKIPsIYNd9TEKo",
		Scopes:       "openid profile email",
		RedirectURI:  "http://localhost:9999/ssf/receiver/app/callback",
	}

	tpl = template.Must(template.ParseGlob("server/web/templates/*"))
	sessionsMap = make(map[string]ssf.SsfReceiverAppData, 0)
	wsClientConnected = false
}

func handleSSFReq(res http.ResponseWriter, req *http.Request) {
	log.Printf("handleSSFReq: Received SSF Request:\n%+v\n", req)

	if req.Method == http.MethodPost {
		// POST
		b, err := getBody(req)
		fmt.Printf("handleSSFReq: RAW JWT Received:\n%s\n", string(b))
		if err != nil {
			log.Printf("handleSSFReq: Error getting POST body: %v\n", err)
			res.WriteHeader(http.StatusBadRequest)
			return
		}

		jwtParts := strings.Split(string(b), ".")
		if len(jwtParts) < 3 {
			log.Printf("handleSSFReq: Invalid JWT Received")
			fmt.Printf("%+v\n", jwtParts)
			res.WriteHeader(http.StatusUnauthorized)
			return
		}

		decodedHeader, _ := base64.RawStdEncoding.DecodeString(jwtParts[0])
		decodedBody, _ := base64.RawStdEncoding.DecodeString(jwtParts[1])
		var secEvtTokenJWTHeader ssf.SecEvtTokenJWTHeader
		if err := json.Unmarshal(decodedHeader, &secEvtTokenJWTHeader); err != nil {
			log.Printf("handleSSFReq: Error decoding JWT Header: %s, err: %v\n", jwtParts[0], err)
			res.WriteHeader(http.StatusForbidden)
			return
		}

		var secEvtTokenJWTBody ssf.SecEvtTokenJWTBody
		if err := json.Unmarshal(decodedBody, &secEvtTokenJWTBody); err != nil {
			log.Printf("handleSSFReq: Error decoding JWT Body: %s, err: %v\n", jwtParts[1], err)
			res.WriteHeader(http.StatusForbidden)
			return
		}

		// Verify JWT
		key, ok := utils.GetKeyForIDFromIssuer(secEvtTokenJWTHeader.Kid, secEvtTokenJWTBody.Iss)
		if !ok {
			res.WriteHeader(http.StatusUnauthorized)
			return
		}

		ok = utils.VerifyJwt(b, key, jwa.RS256) // Just assume RS256, don't think Okta will use anything else
		if !ok {
			res.WriteHeader(http.StatusUnauthorized)
			return
		}

		part, err := json.MarshalIndent(secEvtTokenJWTBody, "", "  ")
		if err != nil {
			fmt.Println(err)
		}
		fmt.Printf("handleSSFReq: %s\n", part)
		fmt.Printf("handleSSFReq: %+v\n", secEvtTokenJWTBody)
		parseEvents(secEvtTokenJWTBody.Events)

		// send to admin app to display
		sendSSFRecieverWebSocketMessage(secEvtTokenJWTBody)
	}

	res.WriteHeader(http.StatusAccepted)
}

func parseEvents(events ssf.Events) {
	if events.DeviceComplianceChange.Event_timestamp > 0 {
		log.Println(" !! DeviceComplianceChange is not Supported !!")
	}
	if events.DeviceRiskChange.Event_timestamp > 0 {
		log.Println(" !! DeviceRiskChange is not Supported !!")
	}
	if events.IpChange.Event_timestamp > 0 {
		log.Println(" !! IpChange is not Supported !!")
	}
	if events.UserRiskChange.Event_timestamp > 0 {
		log.Println(" !! UserRiskChange is not Supported !!")
	}
	if events.IdentifierChanged.Event_timestamp > 0 {
		log.Println(" !! IdentifierChanged is not Supported !!")
	}
	if events.SessionRevoked.Event_timestamp > 0 {
		log.Println("Received Session Revoke SecEvt")
		subject := events.SessionRevoked.Subject
		for k, v := range subject {
			fmt.Printf("subject attribute: %s, value: %s\n", k, v)
			if k == "sub" {
				// check sessions Map for user and remove if present
				for sessionId, sessionV := range sessionsMap {
					if v == sessionV.UUID {
						log.Printf("!!!!!! SECURITY EVENT !!!!!!: Session Revoked for User: %s, id: %s, removing app session\n", sessionV.Username, sessionV.UUID)
						delete(sessionsMap, sessionId)
					}
				}
			}
		}
	}
	if events.CredentialChanged.Event_timestamp > 0 {
		log.Println("Received Credential Changed SecEvt")
		subject := events.CredentialChanged.Subject
		if events.CredentialChanged.Change_type != "create" {
			// change_type is "delete", "update", or "revoke"
			for k, v := range subject {
				fmt.Printf("subject attribute: %s, value: %s\n", k, v)
				if k == "sub" {
					// check sessions Map for user and remove if present
					for k2, sessionV := range sessionsMap {
						if v == sessionV.UUID {
							log.Printf("!!!!!! SECURITY EVENT !!!!!!: Credential Changed for User: %s, id: %s, Force Re-Authentication\n", sessionV.Username, sessionV.UUID)
							sessionV.ForceReAuth = true
							sessionsMap[k2] = sessionV
						}
					}
				}
			}
		}
	}
}

func handleSSFReciever(res http.ResponseWriter, req *http.Request) {
	err := tpl.ExecuteTemplate(res, "ssfreceiver.gohtml", struct{ utils.Services }{config.Services})
	if err != nil {
		log.Printf("handleSSFReciever: %+v\n", err)
	}
}

func handleSSFRecieverAppEmbed(res http.ResponseWriter, req *http.Request) {
	fmt.Println("Returning SSF Receiver App")
	session, _ := store.Get(req, "ssf-receiver-session")
	fmt.Printf("handleSSFReciever values: %+v\n", session.Values)
	_, ok := session.Values["authenticated"]
	if ok {
		ssfReceiverAppData, ok := sessionsMap[session.Values["id"].(string)]
		if ok {
			// Check if Re-Auth is needed (Credential Change Event received for user)
			if ssfReceiverAppData.ForceReAuth {
				ssfReceiverAppData.ForceReAuth = false
				delete(session.Values, "authenticated")
				session.Save(req, res)
				handleSSFRecieverOauthLoginWithExtraParams(res, req, "&prompt=login")
				return
			}

			// Authenticated
			err := tpl.ExecuteTemplate(res, "ssfreceiverAppEmbed.gohtml", ssfReceiverAppData) // just send raw token for now
			if err != nil {
				fmt.Printf("%+v\n", err)
			}
		} else {
			log.Println("handleSSFReciever: Session Cookie set as Authenticated, but no tokenResponse in Map?, redirecting")
			delete(session.Values, "authenticated")
			session.Save(req, res)
			http.Redirect(res, req, "/ssf/receiver/app/embed", http.StatusFound)
		}
	} else {
		// Need Auth
		scheme := utils.GetRequestScheme(req)
		oauthConfig.RedirectURI = fmt.Sprintf("%s://%s/ssf/receiver/app/callback", scheme, req.Host)
		ssfReceiverAppData := ssf.SsfReceiverAppData{Authenticated: false, OauthConfig: oauthConfig}
		err := tpl.ExecuteTemplate(res, "ssfreceiverAppEmbed.gohtml", ssfReceiverAppData)
		if err != nil {
			fmt.Printf("%+v\n", err)
		}
	}
}

func handleSSFTransmitter(res http.ResponseWriter, req *http.Request) {
	// TODO
}

func handleSSFRecieverConfig(res http.ResponseWriter, req *http.Request) {
	if req.Method == http.MethodPost {
		// POST
		b, err := getBody(req)
		if err != nil {
			log.Printf("handleSSFRecieverConfig: Error getting POST body: %v\n", err)
			res.WriteHeader(http.StatusOK)
			return
		}
		err = json.Unmarshal(b, &oauthConfig)
		if err != nil {
			log.Printf("handleSSFRecieverConfig: Error Unmarshall body: %s, err: %+v\n", string(b), err)
			res.WriteHeader(http.StatusOK)
			return
		}
		fmt.Printf("handleSSFRecieverConfig: New App Config,\n%+v\n", oauthConfig)
	}

	res.WriteHeader(http.StatusOK)
}

func handleSSFRecieverOauthLogin(res http.ResponseWriter, req *http.Request) {
	handleSSFRecieverOauthLoginWithExtraParams(res, req, "")
}

func handleSSFRecieverOauthLoginWithExtraParams(res http.ResponseWriter, req *http.Request, extraParams string) {
	oauthConfig.ExtraParams = extraParams
	utils.Authorize(res, req, oauthConfig, Callback)
}

func Callback(res http.ResponseWriter, req *http.Request, tokenResponse types.TokenReponse) {
	id := utils.GenerateUUID()
	session, _ := store.Get(req, "ssf-receiver-session")
	v, ok := session.Values["id"]
	if ok {
		id = v.(string)
	}
	session.Values["authenticated"] = true
	session.Values["id"] = id
	session.Save(req, res)
	ssfReceiverAppData := ssf.SsfReceiverAppData{TokenReponse: tokenResponse, Authenticated: true}
	// won't validate token like with SecEvt JWT since that is main purpose
	jwtBody, _ := base64.RawStdEncoding.DecodeString(strings.Split(tokenResponse.IdToken, ".")[1])
	var m map[string]interface{}
	err := json.Unmarshal(jwtBody, &m)
	if err == nil {
		ssfReceiverAppData.UUID = m["sub"].(string)
		ssfReceiverAppData.Username = m["preferred_username"].(string)
	}

	sessionsMap[id] = ssfReceiverAppData
	fmt.Printf("%+v\n", session.Values)
	http.Redirect(res, req, "/ssf/receiver/app/embed", http.StatusFound)
}

func handleSSFRecieverOauthCallback(res http.ResponseWriter, req *http.Request) {
	utils.HandleOauthCallback(res, req)
}

/*
WebSocket Helpers
*/
func handleSSFRecieverWebSocketUpgrade(res http.ResponseWriter, req *http.Request) {
	// upgrade this connection to a WebSocket connection
	conn, err := upgrader.Upgrade(res, req, nil)
	if err != nil {
		log.Printf("upgrader.Upgrade() err: %v\n", err)
		res.WriteHeader(http.StatusInternalServerError)
		res.Write(nil)
		return
	}

	wsConn = conn
	log.Println("handleSSFReciever Web Socket Client Connected")
	wsClientConnected = true
	// listen indefinitely for new messages coming
	// through on our WebSocket connection
	go wsReader()
}

/*
Used only for Ping to keep WS open
*/
func wsReader() {
	for {
		// read in a message
		fmt.Println("handleSSFReciever WebSocket Reader about to block for Message")
		var m interface{}
		err := wsConn.ReadJSON(&m)
		if err != nil {
			log.Printf("handleSSFReciever wsConn.ReadJSON error: %v\n", err)
			if websocket.IsCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Println("handleSSFReciever wsConn.ReadJSON Client Disconnected")
				wsClientConnected = false
				return
			}
			continue
		}

		// fmt.Printf("handleSSFReciever message: %+v\n", m)
	}
}

func sendSSFRecieverWebSocketMessage(message interface{}) {
	if wsClientConnected {
		err := wsConn.WriteJSON(message)
		if err != nil {
			log.Printf("sendSSFRecieverWebSocketMessage Error: %v\n", err)
		}
	}
}
