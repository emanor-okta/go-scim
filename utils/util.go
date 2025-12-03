package utils

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/lestrrat-go/jwx/v2/jwa"
	"github.com/lestrrat-go/jwx/v2/jwk"
	"github.com/lestrrat-go/jwx/v2/jwt"

	"github.com/gorilla/websocket"

	"github.com/emanor-okta/go-scim/types"
	"github.com/google/uuid"
)

const (
	_logPrefix = "utils.util."

	tokenCall     = "client_id=%s&client_secret=%s&grant_type=authorization_code&redirect_uri=%s&code=%s"
	authorizeCall = "client_id=%s&response_type=code&scope=%s&redirect_uri=%s&state=%s"
)

var stateMap map[string]types.OauthTransactionState

func init() {
	stateMap = make(map[string]types.OauthTransactionState, 0)
}

func GenerateUUID() string {
	return uuid.NewString()
}

func ConvertHeaderMap(m interface{}) map[string][]string {
	r := map[string][]string{}
	for k, v := range m.(map[string]interface{}) {
		r[k] = []string{}
		for _, v2 := range v.([]interface{}) {
			r[k] = append(r[k], v2.(string))
		}
	}
	return r
}

func GetRequestScheme(req *http.Request) string {
	scheme := "https"
	fmt.Printf("Host: %+v\n", req.Host)
	fmt.Printf("%+v", req.Header)

	if req.TLS == nil {
		if !strings.Contains(req.Host, "https://") {
			if !strings.Contains(req.Header.Get("Referer"), "https://") {
				if !strings.Contains(req.Header.Get("X-Forwarded-For"), "https://") {
					scheme = "http"
				}
			}
		}
	}

	return scheme
}

func GetRemoteAddress(r *http.Request) string {
	// if config.Server.FilterIPs {
	// Need to find correct way to do this, for now check X-Forwarded-For, followed by ReoteAddr
	addr := r.Header.Get("X-Forwarded-For")
	if addr == "" {
		addr = r.RemoteAddr
		if addr != "" {
			index := strings.LastIndex(addr, ":")
			if index > -1 {
				addr = addr[0:index]
			}
		}
	}
	// fmt.Printf("X-Forwarded-For: %s, RemoteAddr: %s\n", r.Header.Get("X-Forwarded-For"), r.RemoteAddr)
	return addr
}

func GetBody(req *http.Request) ([]byte, error) {
	b, err := io.ReadAll(req.Body)
	if err != nil {
		log.Printf("%sGetBody: Error reading Json Data: %v\n", _logPrefix, err)
		return nil, err
	}

	defer req.Body.Close()
	return b, nil
}

func GetOktaPublicIPs() map[string]string {
	req, _ := http.NewRequest("GET", Config.Server.Okta_public_ips_url, nil)
	req.Header.Set("Accept", "application/json")
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Fatalf("%sGetOktaPublicIPs: Configuration Requires IP Filtering, failed to download Okta IPs, %+v\n", _logPrefix, err)
	}

	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	var oktaIPs map[string]interface{}
	if err = json.Unmarshal(body, &oktaIPs); err != nil {
		log.Fatalf("%sGetOktaPublicIPs: Configuration Requires IP Filtering, failed to UnMarshall Okta IPs, %+v\n", _logPrefix, err)
	}

	m := map[string]string{}
	for k, v := range oktaIPs {
		ips := v.(map[string]interface{})["ip_ranges"].([]interface{})
		log.Printf("%sGetOktaPublicIPs: Adding Allowed IPs for %s\n", _logPrefix, k)
		for _, ip := range ips {
			ip_ := strings.Split(ip.(string), "/")[0]
			m[ip_] = "_"
		}
	}
	// for k, v := range m {
	// 	fmt.Printf("%s : %s\n", k, v)
	// }
	// localIps := GetLocalIps()
	// for _, ip_ := range localIps {
	// 	m[ip_] = "local-server-ip"
	// }

	// DebugAllowedIPs(m)
	return m
}

func DebugAllowedIPs(ips map[string]string) {
	s := "Allowed Public IPs:\n"
	for k, v := range ips {
		if v != "_" {
			s = fmt.Sprintf("%s  address: %s,  user: %s\n", s, k, v)
		}
	}
	fmt.Println(s)
}

func GetLocalIps() []string {
	ips := []string{}
	ifaces, err := net.Interfaces()
	if err != nil {
		log.Printf("%sGetLocalIps: Error getting interfaces %s\n", _logPrefix, err)
		return ips
	}

	for _, i := range ifaces {
		addrs, err := i.Addrs()
		if err != nil {
			log.Printf("%sGetLocalIps: Error getting address %s\n", _logPrefix, err)
			continue
		}

		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}

			if !ip.IsLoopback() {
				ips = append(ips, ip.String())
			}
		}
	}

	// Get outbound IP when connecting to gw.oktamanor.net
	conn, err := net.Dial("udp", "gw.oktamanor.net:8443")
	if err != nil {
		log.Fatal(err)
	} else {
		defer conn.Close()
		localAddr := conn.LocalAddr().(*net.UDPAddr)
		localIp := localAddr.IP.String()
		index := strings.LastIndex(localIp, ":")
		if index > -1 {
			localIp = localIp[0:index]
		}
		fmt.Println(localIp)
		ips = append(ips, localIp)
	}

	return ips
}

func AddMiddleware(h http.HandlerFunc, m ...types.Middleware) http.HandlerFunc {
	if len(m) < 1 {
		return h
	}

	middlewares := h
	for _, v := range m {
		middlewares = v(middlewares)
	}

	return middlewares
}

/*
OAuth Utils
*/
func Authorize(res http.ResponseWriter, req *http.Request, oauthConfig types.OauthConfig, callback func(http.ResponseWriter, *http.Request, types.TokenReponse)) {
	state := GenerateUUID()
	now := time.Now()
	oauthTransactionState := types.OauthTransactionState{OauthConfig: oauthConfig, State: state, AuthorizeTime: now, Callback: callback}
	stateMap[state] = oauthTransactionState
	// fmt.Printf("%+v\n", stateMap)
	reqParams := fmt.Sprintf(authorizeCall, oauthConfig.ClientId, oauthConfig.Scopes, oauthConfig.RedirectURI, state)
	http.Redirect(res, req, fmt.Sprintf("%s/v1/authorize?%s%s", oauthConfig.Issuer, reqParams, oauthConfig.ExtraParams), http.StatusFound)
}

func HandleOauthCallback(res http.ResponseWriter, req *http.Request) {
	s := req.URL.Query().Get("state")
	c := req.URL.Query().Get("code")
	transactionsState := stateMap[s]
	// fmt.Printf("%+v\n", stateMap)
	delete(stateMap, s)
	// fmt.Printf("%+v\n", transactionsState)
	if s == "" || s != transactionsState.State {
		fmt.Printf("%shandleOauthCallback() - Need to handle no saved state, or wrong value\n", _logPrefix)
	}
	if c == "" {
		fmt.Printf("%shandleOauthCallback() - Need to handle no code\n", _logPrefix)
	}
	// get Tokens
	postBody := fmt.Sprintf(tokenCall, transactionsState.OauthConfig.ClientId, transactionsState.OauthConfig.ClientSecret, transactionsState.OauthConfig.RedirectURI, c)
	tokenReq, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("%s/v1/token", transactionsState.OauthConfig.Issuer), bytes.NewBuffer([]byte(postBody)))
	tokenReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	client := &http.Client{}
	tokenRes, err := client.Do(tokenReq)
	if err != nil {
		fmt.Printf("%sHandleOauthCallback() - Token call Error: %+v\n", _logPrefix, err)
	}

	defer tokenRes.Body.Close()
	body, _ := io.ReadAll(tokenRes.Body)
	if tokenRes.StatusCode != 200 {
		fmt.Printf("%sHandleOauthCallback() - Token call Error: %+v\n", _logPrefix, string(body))
		return
	}
	var tokenResponse types.TokenReponse
	if err = json.Unmarshal(body, &tokenResponse); err != nil {
		fmt.Printf("%sHandleOauthCallback() - Token Json parse Error: %+v\n", _logPrefix, err)
		return
	}

	// Verify token
	jHeader, _, _ := GetJwtParts(tokenResponse.IdToken)
	if jHeader == nil {
		return
	}

	var m map[string]interface{}
	json.Unmarshal(jHeader, &m)
	key, ok := GetKeyForIDFromIssuer(m["kid"].(string), transactionsState.OauthConfig.Issuer)
	if !ok {
		return
	}

	valid := VerifyJwt([]byte(tokenResponse.IdToken), key, jwa.RS256)
	if !valid {
		return
	}

	// call callback with tokens
	transactionsState.Callback(res, req, tokenResponse)
}

/*
	JWT Utils
*/

func GetJwtParts(jwt string) ([]byte, []byte, []byte) {
	jwtParts := strings.Split(jwt, ".")
	if len(jwtParts) < 3 {
		log.Printf("%sGetJwtParts: Invalid JWT Received\n", _logPrefix)
		fmt.Printf("%+v\n", jwtParts)
		return nil, nil, nil
	}

	decodedHeader, _ := base64.RawStdEncoding.DecodeString(jwtParts[0])
	decodedBody, _ := base64.RawStdEncoding.DecodeString(jwtParts[1])
	decodedSignature, _ := base64.RawStdEncoding.DecodeString(jwtParts[2])

	return decodedHeader, decodedBody, decodedSignature
}

func GetKeyForIDFromIssuer(id, issuer string) (jwk.Key, bool) {
	if strings.Contains(issuer, "/oauth2/") {
		// Custom AS
		issuer = issuer + "/v1/keys"
	} else {
		// Org AS
		issuer = issuer + "/oauth2/v1/keys"
	}

	set, err := jwk.Fetch(context.Background(), issuer)
	if err != nil {
		log.Printf("getKeyForIDFromIssuer: failed to fetch JWK keys from: %s, error: %+v\n", issuer, err)
		return nil, false
	} else {
		key, ok := set.LookupKeyID(id)
		if !ok {
			log.Printf("getKeyForIDFromIssuer: failed to find key: %s\n", id)
			return nil, false
		}
		return key, true
	}
}

func VerifyJwt(jwtBytes []byte, key jwk.Key, alg jwa.KeyAlgorithm) bool {
	_, err := jwt.Parse(jwtBytes, jwt.WithKey(alg, key))
	if err != nil {
		fmt.Printf("verifyJwt: failed to verify JWT: %s, error: %s\n", string(jwtBytes), err)
		return false
	}

	return true
}

/*
JWK Utils
*/
func GenerateKey() (jwk.Key, jwk.Key, error) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		log.Printf("\nError, Generating RSA Private Key: %v\n", err)
		return nil, nil, fmt.Errorf("error generating RSA private key: %v", err)
	}

	jwkKey, err := jwk.FromRaw(privateKey)
	if err != nil {
		log.Printf("\nError, Generating JWK Key from RSA Private Key: %v\n", err)
		return nil, nil, fmt.Errorf("error generating JWK key from  RSA private key: %v", err)
	}

	pubKey, err := jwkKey.PublicKey()
	if err != nil {
		log.Printf("\nError, Getting Public Key Part from JWK: %v\n", err)
		return nil, nil, fmt.Errorf("error getting public key part from JWK: %v", err)
	}

	return jwkKey, pubKey, nil
}

/*
WS Utils
*/
func HandleWebSocketUpgrade(res http.ResponseWriter, req *http.Request, wsClientConnected *bool) *websocket.Conn {
	upgrader := websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin:     nil, //func(r *http.Request) bool { return true },
	}
	// upgrade this connection to a WebSocket connection
	conn, err := upgrader.Upgrade(res, req, nil)
	if err != nil {
		log.Printf("upgrader.Upgrade() err: %v\n", err)
		res.WriteHeader(http.StatusInternalServerError)
		res.Write(nil)
		return nil
	}

	// wsConn = conn
	log.Println("Web Socket Client Connected")
	*wsClientConnected = true
	return conn
	// listen indefinitely for new messages coming
	// through on our WebSocket connection
	// go wsReader()
}

/*
 * Used only for Ping to keep WS open
 */
func WsPingOnlyReader(wsConn *websocket.Conn, wsClientConnected *bool) {
	for {
		// read in a message
		log.Println("handle WebSocket Reader about to block for Message")
		var m interface{}
		err := wsConn.ReadJSON(&m)
		if err != nil {
			log.Printf("handle wsConn.ReadJSON error: %v\n", err)
			if websocket.IsCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Println("WsPingOnlyReader wsConn.ReadJSON Client Disconnected")
				*wsClientConnected = false
				return
			}
			continue
		}

		// fmt.Printf("handleSSFReciever message: %+v\n", m)
	}
}

/*
 * Populate SCIM Entitlements into Config
 */
func LoadScimEntitlements() {
	result, err := GetResourceTypes()
	if err == nil && result != "" {
		// assume Entitlement bits are stored in Redis, load from Redis
		Config.Entitlements.ResourceTypes = []byte(result)

		results, err := GetResources()
		if err == nil {
			Config.Entitlements.Resources = make(map[string][]byte)
			for k, v := range results {
				Config.Entitlements.Resources[strings.ToLower(k)] = []byte(v)
			}
		}

		result, err = GetSchemas()
		if err == nil {
			Config.Entitlements.Schemas = []byte(result)
		}
	} else {
		// either error'd loading from Redis or wasn't set, load defaults
		Config.Entitlements.ResourceTypes = LoadRawJson("./server/web/raw/entitlements/resource_types.json")
		// Config.Entitlements.Roles = LoadRawJson("./server/web/raw/entitlements/roles.json")
		Config.Entitlements.Resources = make(map[string][]byte)
		Config.Entitlements.Resources["roles"] = LoadRawJson("./server/web/raw/entitlements/roles.json")
		Config.Entitlements.Resources["entitlements"] = LoadRawJson("./server/web/raw/entitlements/entitlements.json")
		Config.Entitlements.Resources["features"] = LoadRawJson("./server/web/raw/entitlements/features.json")
		Config.Entitlements.Resources["licenses"] = LoadRawJson("./server/web/raw/entitlements/licenses.json")
		Config.Entitlements.Schemas = LoadRawJson("./server/web/raw/entitlements/schemas.json")
		// save to redis
		SetResourceTypes(string(Config.Entitlements.ResourceTypes))
		SetSchema(string(Config.Entitlements.Schemas))
		SetResource("roles", string(Config.Entitlements.Resources["roles"]))
		SetResource("entitlements", string(Config.Entitlements.Resources["entitlements"]))
		SetResource("features", string(Config.Entitlements.Resources["features"]))
		SetResource("licenses", string(Config.Entitlements.Resources["licenses"]))
	}
}

func LoadRawJson(fileName string) []byte {
	result, err := os.ReadFile(fileName)
	if err != nil {
		log.Printf("Error: LoadRawJson for file: %s, error: %v\n", fileName, err)
		result = []byte{}
	}
	return result
}
