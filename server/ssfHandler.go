package server

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/gorilla/sessions"
	"github.com/gorilla/websocket"
	"github.com/lestrrat-go/jwx/v2/jwa"

	"github.com/emanor-okta/go-scim/utils"
)

type SecEvtTokenJWTHeader struct {
	Kid string `json:"kid,omitempty"`
	Typ string `json:"typ,omitempty"`
	Alg string `json:"alg,omitempty"`
}

type Reason struct {
	En string `json:"en,omitempty"`
}

type EvtAttributes struct {
	Current_ip            string            `json:"current_ip,omitempty"`
	Current_user_agent    string            `json:"current_user_agent,omitempty"`
	Event_timestamp       int64             `json:"event_timestamp,omitempty"`
	Initiating_entity     string            `json:"initiating_entity,omitempty"`
	Current_level         string            `json:"current_level,omitempty"`
	Previous_level        string            `json:"previous_level,omitempty"`
	Current_ip_address    string            `json:"current_ip_address,omitempty"`
	Previous_ip_address   string            `json:"previous_ip_address,omitempty"`
	Last_known_ip         string            `json:"last_known_ip,omitempty"`
	Last_known_user_agent string            `json:"last_known_user_agent,omitempty"`
	New_value             string            `json:"new-value,omitempty"`
	Reason_admin          Reason            `json:"reason_admin,omitempty"`
	Reason_user           Reason            `json:"reason_user,omitempty"`
	Subject               map[string]string `json:"subject,omitempty"`
}

type Events struct {
	DeviceRiskChange struct {
		EvtAttributes
	} `json:"https://schemas.okta.com/secevent/okta/event-type/device-risk-change,omitempty"`
	IpChange struct {
		EvtAttributes
	} `json:"https://schemas.okta.com/secevent/okta/event-type/ip-change,omitempty"`
	UserRiskChange struct {
		EvtAttributes
	} `json:"https://schemas.okta.com/secevent/okta/event-type/user-risk-change,omitempty"`
	DeviceComplianceChange struct {
		EvtAttributes
	} `json:"https://schemas.openid.net/secevent/caep/event-type/device-compliance-change,omitempty"`
	SessionRevoked struct {
		EvtAttributes
	} `json:"https://schemas.openid.net/secevent/caep/event-type/session-revoked,omitempty"`
	IdentifierChanged struct {
		EvtAttributes
	} `json:"https://schemas.openid.net/secevent/risc/event-type/identifier-changed,omitempty"`
}

type SecEvtTokenJWTBody struct {
	Iss    string `json:"iss,omitempty"`
	Aud    string `json:"aud,omitempty"`
	Jti    string `json:"jti,omitempty"`
	Iat    int64  `json:"iat,omitempty"`
	Events Events `json:"events,omitempty"`
}

type OauthConfig struct {
	Issuer       string `json:"issuer,omitempty"`
	ClientId     string `json:"client_id,omitempty"`
	ClientSecret string `json:"client_secret,omitempty"`
	Scopes       string `json:"scopes,omitempty"`
	RedirectURI  string `json:"redirect_url,omitempty"`
}

type TokenReponse struct {
	AccessToken string `json:"access_token,omitempty"`
	IdToken     string `json:"id_token,omitempty"`
}

type SsfReceiverAppData struct {
	TokenReponse
	OauthConfig
	Authenticated bool
	Username,
	UUID string
}

const TokenCall = "client_id=%s&client_secret=%s&grant_type=authorization_code&redirect_uri=%s&code=%s"
const AuthorizeCall = "client_id=%s&response_type=code&scope=%s&redirect_uri=%s&state=%s"

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
var oauthConfig OauthConfig
var tpl *template.Template
var sessionsMap map[string]SsfReceiverAppData

func init() {
	// hardcode for now
	oauthConfig = OauthConfig{
		Issuer:       "https://emanor-oie.oktapreview.com/oauth2/default",
		ClientId:     "0oa2cpl777xczKzL21d7",
		ClientSecret: "Klf03TzBqEuayATkPGy7VgTqyNDKIPsIYNd9TEKo",
		Scopes:       "openid profile email",
		RedirectURI:  "http://localhost:8082/ssf/receiver/app/callback",
	}

	tpl = template.Must(template.ParseGlob("server/web/templates/*"))
	sessionsMap = make(map[string]SsfReceiverAppData, 0)
}

func handleSSFReq(res http.ResponseWriter, req *http.Request) {
	log.Printf("handleSSFReq: Received SSF Request:\n%+v\n", req)

	if req.Method == http.MethodPost {
		// POST
		b, err := getBody(req)
		if err != nil {
			log.Printf("handleSSFReq: Error getting POST body: %v\n", err)
			res.WriteHeader(http.StatusOK)
			return
		}

		jwtParts := strings.Split(string(b), ".")
		if len(jwtParts) < 3 {
			log.Printf("handleSSFReq: Invalid JWT Received")
			fmt.Printf("%+v\n", jwtParts)
			res.WriteHeader(http.StatusOK)
			return
		}

		decodedHeader, _ := base64.RawStdEncoding.DecodeString(jwtParts[0])
		decodedBody, _ := base64.RawStdEncoding.DecodeString(jwtParts[1])

		var secEvtTokenJWTHeader SecEvtTokenJWTHeader
		if err := json.Unmarshal(decodedHeader, &secEvtTokenJWTHeader); err != nil {
			log.Printf("handleSSFReq: Error decoding JWT Header: %s, err: %v\n", jwtParts[0], err)
			res.WriteHeader(http.StatusOK)
			return
		}

		var secEvtTokenJWTBody SecEvtTokenJWTBody
		if err := json.Unmarshal(decodedBody, &secEvtTokenJWTBody); err != nil {
			log.Printf("handleSSFReq: Error decoding JWT Body: %s, err: %v\n", jwtParts[1], err)
			res.WriteHeader(http.StatusOK)
			return
		}

		// Verify JWT
		key, ok := utils.GetKeyForIDFromIssuer(secEvtTokenJWTHeader.Kid, secEvtTokenJWTBody.Iss)
		if !ok {
			res.WriteHeader(http.StatusOK)
			return
		}

		ok = utils.VerifyJwt(b, key, jwa.RS256) // Just assume RS256, don't think Okta will use anything else
		if !ok {
			res.WriteHeader(http.StatusOK)
			return
		}

		part, err := json.MarshalIndent(secEvtTokenJWTBody, "", "  ")
		if err != nil {
			fmt.Println(err)
		}
		fmt.Printf("%s\n", part)
		fmt.Printf("%+v\n", secEvtTokenJWTBody)
		parseEvents(secEvtTokenJWTBody.Events)

		//TESTING
		sendSSFRecieverWebSocketMessage(secEvtTokenJWTBody)
	}

	res.WriteHeader(http.StatusOK)
}

func parseEvents(events Events) {
	if events.DeviceComplianceChange.Event_timestamp > 0 {

	}
	if events.DeviceRiskChange.Event_timestamp > 0 {

	}
	if events.IpChange.Event_timestamp > 0 {

	}
	if events.UserRiskChange.Event_timestamp > 0 {

	}
	if events.IdentifierChanged.Event_timestamp > 0 {

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
						log.Printf("SECURITY EVENT: Session Revoked for User: %s, id: %s, removing app session\n", sessionV.Username, sessionV.UUID)
						delete(sessionsMap, sessionId)
					}
				}
			}
		}
	}
}

func handleSSFReciever(res http.ResponseWriter, req *http.Request) {
	fmt.Println("Returning SSF Receiver App")
	session, _ := store.Get(req, "ssf-receiver-session")
	fmt.Printf("handleSSFReciever values: %+v\n", session.Values)
	_, ok := session.Values["authenticated"]
	if ok {
		ssfReceiverAppData, ok := sessionsMap[session.Values["id"].(string)]
		if ok {
			// Authenticated
			err := tpl.ExecuteTemplate(res, "ssfreceiver.gohtml", ssfReceiverAppData) // just send raw token for now
			if err != nil {
				fmt.Printf("%+v\n", err)
			}
		} else {
			log.Println("handleSSFReciever: Session Cookie set as Authenticated, but no tokenResponse in Map?, redirecting")
			delete(session.Values, "authenticated")
			session.Save(req, res)
			http.Redirect(res, req, "/ssf/receiver/app", http.StatusFound)
		}
	} else {
		// Need Auth
		ssfReceiverAppData := SsfReceiverAppData{Authenticated: false, OauthConfig: oauthConfig}
		err := tpl.ExecuteTemplate(res, "ssfreceiver.gohtml", ssfReceiverAppData) // just send raw token for now
		if err != nil {
			fmt.Printf("%+v\n", err)
		}
	}
}

func handleSSFTransmitter(res http.ResponseWriter, req *http.Request) {

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
	session, _ := store.Get(req, "ssf-receiver-session")
	state := utils.GenerateUUID()
	session.Values["state"] = state
	session.Save(req, res)
	reqParams := fmt.Sprintf(AuthorizeCall, oauthConfig.ClientId, oauthConfig.Scopes, oauthConfig.RedirectURI, state)
	http.Redirect(res, req, fmt.Sprintf("%s/v1/authorize?%s", oauthConfig.Issuer, reqParams), http.StatusFound)
}

func handleSSFRecieverOauthCallback(res http.ResponseWriter, req *http.Request) {
	session, _ := store.Get(req, "ssf-receiver-session")
	fmt.Printf("session: %+v\n", session)
	state, ok := session.Values["state"]
	if ok {
		// Check State
		s := req.URL.Query().Get("state")
		c := req.URL.Query().Get("code")
		if s == "" || s != state {
			fmt.Println("handleSSFRecieverOauthCallback() - Need to handle no saved state, or wrong value")
		}
		if c == "" {
			fmt.Println("handleSSFRecieverOauthCallback() - Need to handle no code")
		}
		// get Tokens
		postBody := fmt.Sprintf(TokenCall, oauthConfig.ClientId, oauthConfig.ClientSecret, oauthConfig.RedirectURI, c)
		req, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("%s/v1/token", oauthConfig.Issuer), bytes.NewBuffer([]byte(postBody)))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			fmt.Printf("handleSSFRecieverOauthCallback() - Token call Error: %+v\n", err)
		}

		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)
		if resp.StatusCode != 200 {
			fmt.Printf("handleSSFRecieverOauthCallback() - Token call Error: %+v\n", string(body))
			return
		}
		var tokenResponse TokenReponse
		if err = json.Unmarshal(body, &tokenResponse); err != nil {
			fmt.Printf("handleSSFRecieverOauthCallback() - Token Json parse Error: %+v\n", err)
			return
		}

		session.Values["authenticated"] = true
		id := session.Values["state"].(string)
		session.Values["id"] = id
		delete(session.Values, "state")
		session.Save(req, res)
		ssfReceiverAppData := SsfReceiverAppData{TokenReponse: tokenResponse, Authenticated: true}
		// won't validate token like with SecEvt JWT since that is main purpose
		jwtBody, _ := base64.RawStdEncoding.DecodeString(strings.Split(tokenResponse.IdToken, ".")[1])
		var m map[string]interface{}
		err = json.Unmarshal(jwtBody, &m)
		if err == nil {
			ssfReceiverAppData.UUID = m["sub"].(string)
			ssfReceiverAppData.Username = m["preferred_username"].(string)
		}

		sessionsMap[id] = ssfReceiverAppData
		fmt.Printf("%+v\n", session.Values)
		http.Redirect(res, req, "/ssf/receiver/app", http.StatusFound)
	} else {
		// Invalid
		fmt.Println("handleSSFRecieverOauthCallback() - Need to handle no state")
	}
}

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
				return
			}
			continue
		}

		// fmt.Printf("handleSSFReciever message: %+v\n", m)
	}
}

func sendSSFRecieverWebSocketMessage(message interface{}) {
	err := wsConn.WriteJSON(message)
	if err != nil {
		log.Printf("sendSSFRecieverWebSocketMessage Error: %v\n", err)
	}
}
