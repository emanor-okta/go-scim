package server

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gorilla/sessions"
	"github.com/gorilla/websocket"
	"github.com/lestrrat-go/jwx/v2/jwa"
	"github.com/lestrrat-go/jwx/v2/jwk"
	"github.com/lestrrat-go/jwx/v2/jws"

	"github.com/emanor-okta/go-scim/types"
	"github.com/emanor-okta/go-scim/types/ssf"
	"github.com/emanor-okta/go-scim/utils"
)

// const TokenCall = "client_id=%s&client_secret=%s&grant_type=authorization_code&redirect_uri=%s&code=%s"
// const AuthorizeCall = "client_id=%s&response_type=code&scope=%s&redirect_uri=%s&state=%s"
const success = 0

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

var publicKey, privateKey jwk.Key

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

func handleGlobalLogout(res http.ResponseWriter, req *http.Request) {
	log.Printf("handleGlobalLogout: Received Logout Request:\n%+v\n", req)
	if req.Method != http.MethodPost {
		res.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	authHeaderParts := strings.Split(req.Header.Get("Authorization"), " ")
	if strings.ToLower(authHeaderParts[0]) != "bearer" {
		log.Printf("handleGlobalLogout: Received Request without bearer token:\n%+v\n", req.Header.Get("Authorization"))
		res.WriteHeader(http.StatusUnauthorized)
		return
	}

	_, result := verify(authHeaderParts[1])
	if result != success {
		res.WriteHeader(result)
		return
	}

	b, err := getBody(req)
	fmt.Printf("handleGlobalLogout: Body Received:\n%s\n", string(b))
	if err != nil {
		log.Printf("handleGlobalLogout: Error getting POST body: %v\n", err)
		res.WriteHeader(http.StatusBadRequest)
		return
	}

	// var m map[string]interface{}
	var universalLogout struct {
		Sub_id struct {
			Format string `json:"format,omitempty"`
			Sub    string `json:"sub,omitempty"`
			Iss    string `json:"iss,omitempty"`
			Email  string `json:"email,omitempty"`
		} `json:"sub_id,omitempty"`
	}
	if err = json.Unmarshal(b, &universalLogout); err != nil {
		fmt.Printf("handleGlobalLogout: Invalid Body Received:\n%s\n", string(b))
		res.WriteHeader(http.StatusBadRequest)
		return
	}

	fmt.Printf("%+v\n", universalLogout)
	// {"sub_id":{"format":"iss_sub","iss":"https://emanor-oie.oktapreview.com","sub":"00u72c3w7p0pOSvuj1d7"}}
	// {"sub_id":{"format":"email","email":"emanor.okta2@gmail.com"}}
	events := ssf.Events{}
	events.SessionRevoked.Event_timestamp = time.Now().UnixMilli() // timestamp would be in bearer JWT, just make it now
	// events.SessionRevoked.Subject = map[string]string{}
	events.SessionRevoked.Subject = ssf.Subject{}
	if universalLogout.Sub_id.Format == "email" {
		// events.SessionRevoked.Subject["sub"] = universalLogout.Sub_id.Email
		events.SessionRevoked.Subject.Sub = universalLogout.Sub_id.Email
	} else {
		// events.SessionRevoked.Subject["sub"] = universalLogout.Sub_id.Sub
		events.SessionRevoked.Subject.Sub = universalLogout.Sub_id.Sub
	}

	parseEvents(events)

	// send to admin app to display
	sendSSFRecieverWebSocketMessage(universalLogout)

	// if events.SessionRevoked.Event_timestamp > 0 {
	// 	log.Println("Received Session Revoke SecEvt")
	// 	subject := events.SessionRevoked.Subject
	// 	for k, v := range subject {
	// 		fmt.Printf("subject attribute: %s, value: %s\n", k, v)
	// 		if k == "sub" {
	// 			// check sessions Map for user and remove if present
	// 			for sessionId, sessionV := range sessionsMap {
	// 				if v == sessionV.UUID {

	res.WriteHeader(http.StatusAccepted)
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

		// jwtParts := strings.Split(string(b), ".")
		// if len(jwtParts) < 3 {
		// 	log.Printf("handleSSFReq: Invalid JWT Received")
		// 	fmt.Printf("%+v\n", jwtParts)
		// 	res.WriteHeader(http.StatusUnauthorized)
		// 	return
		// }

		// decodedHeader, _ := base64.RawStdEncoding.DecodeString(jwtParts[0])
		// decodedBody, _ := base64.RawStdEncoding.DecodeString(jwtParts[1])
		// var secEvtTokenJWTHeader ssf.SecEvtTokenJWTHeader
		// if err := json.Unmarshal(decodedHeader, &secEvtTokenJWTHeader); err != nil {
		// 	log.Printf("handleSSFReq: Error decoding JWT Header: %s, err: %v\n", jwtParts[0], err)
		// 	res.WriteHeader(http.StatusForbidden)
		// 	return
		// }

		// var secEvtTokenJWTBody ssf.SecEvtTokenJWTBody
		// if err := json.Unmarshal(decodedBody, &secEvtTokenJWTBody); err != nil {
		// 	log.Printf("handleSSFReq: Error decoding JWT Body: %s, err: %v\n", jwtParts[1], err)
		// 	res.WriteHeader(http.StatusForbidden)
		// 	return
		// }

		// // Verify JWT
		// key, ok := utils.GetKeyForIDFromIssuer(secEvtTokenJWTHeader.Kid, secEvtTokenJWTBody.Iss)
		// if !ok {
		// 	res.WriteHeader(http.StatusUnauthorized)
		// 	return
		// }

		// ok = utils.VerifyJwt(b, key, jwa.RS256) // Just assume RS256, don't think Okta will use anything else
		// if !ok {
		// 	res.WriteHeader(http.StatusUnauthorized)
		// 	return
		// }
		secEvtTokenJWTBody, result := verify(string(b))
		if result != success {
			res.WriteHeader(result)
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

func verify(jwt string) (ssf.SecEvtTokenJWTBody, int) {
	var secEvtTokenJWTHeader ssf.SecEvtTokenJWTHeader
	var secEvtTokenJWTBody ssf.SecEvtTokenJWTBody
	jwtParts := strings.Split(jwt, ".")
	if len(jwtParts) < 3 {
		log.Printf("verify: Invalid JWT Received")
		fmt.Printf("%+v\n", jwtParts)
		return secEvtTokenJWTBody, http.StatusUnauthorized
	}

	decodedHeader, _ := base64.RawStdEncoding.DecodeString(jwtParts[0])
	decodedBody, _ := base64.RawStdEncoding.DecodeString(jwtParts[1])
	if err := json.Unmarshal(decodedHeader, &secEvtTokenJWTHeader); err != nil {
		log.Printf("verify: Error decoding JWT Header: %s, err: %v\n", jwtParts[0], err)
		return secEvtTokenJWTBody, http.StatusForbidden
	}

	if err := json.Unmarshal(decodedBody, &secEvtTokenJWTBody); err != nil {
		log.Printf("verify: Error decoding JWT Body: %s, err: %v\n", jwtParts[1], err)
		return secEvtTokenJWTBody, http.StatusForbidden
	}

	// Verify JWT
	key, ok := utils.GetKeyForIDFromIssuer(secEvtTokenJWTHeader.Kid, secEvtTokenJWTBody.Iss)
	if !ok {
		return secEvtTokenJWTBody, http.StatusUnauthorized
	}

	ok = utils.VerifyJwt([]byte(jwt), key, jwa.RS256) // Just assume RS256, don't think Okta will use anything else
	if !ok {
		return secEvtTokenJWTBody, http.StatusUnauthorized
	}

	return secEvtTokenJWTBody, success
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
		fmt.Printf("subject attributes: %+v\n", events.SessionRevoked.Subject)
		for sessionId, sessionV := range sessionsMap {
			if events.SessionRevoked.Subject.Sub == sessionV.UUID {
				log.Printf("!!!!!! SECURITY EVENT !!!!!!: Session Revoked for User: %s, id: %s, removing app session\n", sessionV.Username, sessionV.UUID)
				delete(sessionsMap, sessionId)
			}
		}

		// subject := events.SessionRevoked.Subject
		// for k, v := range subject {
		// 	fmt.Printf("subject attribute: %s, value: %s\n", k, v)
		// 	if k == "sub" {
		// 		// check sessions Map for user and remove if present
		// 		for sessionId, sessionV := range sessionsMap {
		// 			if v == sessionV.UUID {
		// 				log.Printf("!!!!!! SECURITY EVENT !!!!!!: Session Revoked for User: %s, id: %s, removing app session\n", sessionV.Username, sessionV.UUID)
		// 				delete(sessionsMap, sessionId)
		// 			}
		// 		}
		// 	}
		// }
	}
	if events.CredentialChanged.Event_timestamp > 0 {
		log.Println("Received Credential Changed SecEvt")
		fmt.Printf("subject attributes: %+v\n", events.CredentialChanged.Subject)
		for sessionId, sessionV := range sessionsMap {
			if events.CredentialChanged.Subject.Sub == sessionV.UUID {
				log.Printf("!!!!!! SECURITY EVENT !!!!!!: Credential Changed for User: %s, id: %s, Force Re-Authentication\n", sessionV.Username, sessionV.UUID)
				sessionV.ForceReAuth = true
				sessionsMap[sessionId] = sessionV
			}
		}

		// subject := events.CredentialChanged.Subject
		// if events.CredentialChanged.Change_type != "create" {
		// 	// change_type is "delete", "update", or "revoke"
		// 	for k, v := range subject {
		// 		fmt.Printf("subject attribute: %s, value: %s\n", k, v)
		// 		if k == "sub" {
		// 			// check sessions Map for user and remove if present
		// 			for k2, sessionV := range sessionsMap {
		// 				if v == sessionV.UUID {
		// 					log.Printf("!!!!!! SECURITY EVENT !!!!!!: Credential Changed for User: %s, id: %s, Force Re-Authentication\n", sessionV.Username, sessionV.UUID)
		// 					sessionV.ForceReAuth = true
		// 					sessionsMap[k2] = sessionV
		// 				}
		// 			}
		// 		}
		// 	}
		// }
	}
}

func handleSSFReciever(res http.ResponseWriter, req *http.Request) {
	err := tpl.ExecuteTemplate(res, "ssfreceiver.gohtml", struct{ utils.Services }{config.Services})
	if err != nil {
		log.Printf("handleSSFReciever Error: %+v\n", err)
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
	err := tpl.ExecuteTemplate(res, "ssftransmitter.gohtml", struct{ utils.Services }{config.Services})
	if err != nil {
		log.Printf("handleSSFTransmitter Error: %+v\n", err)
	}
}

func handleGetSecurityEventType(res http.ResponseWriter, req *http.Request) {
	paths := strings.Split(strings.Split(req.RequestURI, "?")[0], "/")
	resource := paths[len(paths)-1]
	var eventJson []byte
	var err error
	switch resource {
	case "session-revoke":
		eventJson, err = os.ReadFile("./server/web/raw/securityEvents/sessionRevoke.txt")
	default:
		eventJson = []byte("{}")
	}

	if err != nil {
		log.Printf("handleGetSecurityEventType Error: %+v\n", err)
		eventJson = []byte("{}")
	}
	res.Write(eventJson)
}

func handleSendSecurityEvents(res http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		log.Printf("handleSendSecurityEvents Error, invalid Method: %s\n", req.Method)
		res.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	data, err := utils.GetBody(req)
	if err != nil {
		log.Printf("handleSendSecurityEvents Error: %+v\n", err)
		res.WriteHeader(http.StatusBadRequest)
		return
	}
	fmt.Printf("%s\n", string(data))
	//var secEvtTokenBody ssf.SecEvtTokenJWTBody
	var events map[string]interface{}
	err = json.Unmarshal(data, &events)
	if err != nil {
		log.Printf("handleSendSecurityEvents Error Unmarshalling Events: %+v\n", err)
		res.WriteHeader(http.StatusBadRequest)
		return
	}

	// secEvtJWTHeader := ssf.SecEvtTokenJWTHeader{
	// 	Kid: "ssfTransmitterKey",
	// 	Typ: "secevent+jwt",
	// 	Alg: "RS256",
	// }
	privKeyPem := `-----BEGIN PRIVATE KEY-----
MIIEvgIBADANBgkqhkiG9w0BAQEFAASCBKgwggSkAgEAAoIBAQCfVHXVaAzSeC2C

zqGmfDaixMZoQYf0amSb9Oth
-----END PRIVATE KEY-----`

	privkey, err := jwk.ParseKey([]byte(privKeyPem), jwk.WithPEM(true))
	if err != nil {
		log.Fatalf("\nfailed to parse JWK: %s\n", err)
	}

	// pubkey, err := jwk.PublicKeyOf(privkey)
	if err != nil {
		log.Fatalf("\nfailed to get public key: %s\n", err)
	}

	hdrs := jws.NewHeaders()
	hdrs.Set("typ", "secevent+jwt")
	hdrs.Set("alg", "RS256")
	hdrs.Set("kid", "ssfTransmitterKey")
	// hdrs.Set("jwk", pubkey)

	// if secEvtTokenBody.Events.SessionRevoked.Event_timestamp < 0 {
	// 	secEvtTokenBody.Events.SessionRevoked =
	// }

	signVerifyOptions := jws.WithKey(jwa.RS256, privkey, jws.WithProtectedHeaders(hdrs))
	jwtPayloadBytes, _ := json.Marshal(events)
	buf, err := jws.Sign(jwtPayloadBytes, signVerifyOptions)
	if err != nil {
		log.Fatalf("\nFailed to signJWT: %+v, with options: %+v\n", events, signVerifyOptions)
	}
	fmt.Printf("%s\n", string(buf))
	// fmt.Printf("%+v\n", secEvtTokenBody.Events.SessionRevoked.Event_timestamp)
}

func handleSSFTransmitterConfig(res http.ResponseWriter, _ *http.Request) {
	// /ssf/transmitter/.well-known/sse-configuration
	log.Printf("Returning handleSSFTransmitterConfig")
	transmitterConfig := fmt.Sprintf(`
{
    "issuer": "%s",
    "jwks_uri": "%s",
    "delivery_methods_supported": [
        "urn:ietf:rfc:8935",
        "https://schemas.openid.net/secevent/risc/delivery-method/push"
    ],
    "configuration_endpoint": "%s"
}
	`, utils.Config.Ssf.Transmitter_config.Issuer, utils.Config.Ssf.Transmitter_config.Jwks_uri, utils.Config.Ssf.Transmitter_config.Configuration_endpoint)
	res.Header().Add("content-type", "application/json")
	res.Write([]byte(transmitterConfig))
}

func handleSSFTransmitterKeys(res http.ResponseWriter, _ *http.Request) {
	// Generate keyset for signing if nil (*Okta should query keys endpoint if a Security Token is received for a key it does not have)
	// /ssf/transmitter/keys
	log.Printf("Returning handleSSFTransmitterKeys")
	if publicKey == nil {
		var err error
		privateKey, publicKey, err = utils.GenerateKey()
		if err != nil {
			log.Printf("handleSSFTransmitterKeys: Error Generating Keys, %+v\n", err)
			res.WriteHeader(http.StatusInternalServerError)
			return
		}

		uuid := utils.GenerateUUID()
		privateKey.Set("kid", uuid)
		publicKey.Set("kid", uuid)
		publicKey.Set("use", "sig")
	}

	var buf bytes.Buffer
	err := json.NewEncoder(&buf).Encode(publicKey)
	if err != nil {
		log.Printf("handleSSFTransmitterKeys: Error Encoding Key, %+v\n", err)
		res.WriteHeader(http.StatusInternalServerError)
		return
	}
	// fmt.Printf("%s\n\n", buf.String())

	hardeCoded := `
	{
		"keys": [
			{
				"kid": "ssfTransmitterKey",
				"kty": "RSA",
				"e": "AQAB",
				"use": "sig",
				"n": "n1R11WgM0ngtgtW2bHOfSX6KsGrFTieo1UnQzQK0zDcxnqiOXAb4a7lbaehulfxxmFyaR3EFd1lCgQ1HucfASyRRbxLi0ibtlQTnxwQPnLEhdEi36qeGLnSduSEDUfJHf9-f5Qs38T5gQKM7-qtbF1GJpuYI_m3CTuta1re_pzEIuVE3qDxgoPAZlvx1GhEGJHv4Bf8lWzkpi1jy3kwXROSb1xSX-enhizSTVO63p4PmRPf1T8I4x-UgyEtd_J8NYhM38GCojrP64Bjhsvf3K7AWjS2UPa0F6YMIyrU2H0QS_OpwuPmBA4gkjpqWc6hzsiQhEdt0Jc7b9L1yUS3faw"
			},
			{
				"kty": "RSA",
				"e": "AQAB",
				"use": "sig",
				"kid": "ssfTransmitterKeyNew",
				"n": "qDeXAi-rEpSGyDD59pvAXtFsxMEPeoz2VRz2TdXV8cAWsoUZilv7k8zocjHPuXOrzOCOKqbYivqnwfO8dE6xqutcYiW_WIFVzoMwPm7XE0Mj_jJpaaaNJnMRxzpM-dOOnLjjNW5xZ-Af8NbQl5OkSeNOYLYprwFx7QirbXfcEeNsAdGz4-r_in_Tzp_ZRktxUtL1eYrIpOTTTnDfP2DSBCi5oQ-xdCQMDpDJGTDC1v60Bis2Xv31EkbV5is8bcsAoBOqLTdqvVz5mN6N-HGxKSt2bnnI_hpkjBOJ5_F9gUvEAhA0Te8L-6yWJMXU5FqXaLfHpK8mPUBgX4-ZH8ZNgw"
			}
		]
	}
	`
	// 	keys := fmt.Sprintf(`
	// {
	// 	"keys": [
	// 		%s
	// 	]
	// }
	// 	`, buf.String())

	res.Header().Add("content-type", "application/json")
	// res.Write([]byte(keys))
	res.Write([]byte(hardeCoded))
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
