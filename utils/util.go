package utils

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/lestrrat-go/jwx/v2/jwa"
	"github.com/lestrrat-go/jwx/v2/jwk"
	"github.com/lestrrat-go/jwx/v2/jwt"

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

	return addr
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
	reqParams := fmt.Sprintf(authorizeCall, oauthConfig.ClientId, oauthConfig.Scopes, oauthConfig.RedirectURI, state)
	http.Redirect(res, req, fmt.Sprintf("%s/v1/authorize?%s%s", oauthConfig.Issuer, reqParams, oauthConfig.ExtraParams), http.StatusFound)
}

func HandleOauthCallback(res http.ResponseWriter, req *http.Request) {
	s := req.URL.Query().Get("state")
	c := req.URL.Query().Get("code")
	transactionsState := stateMap[s]
	delete(stateMap, s)
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
		fmt.Printf("%shandleSSFRecieverOauthCallback() - Token call Error: %+v\n", _logPrefix, err)
	}

	defer tokenRes.Body.Close()
	body, _ := io.ReadAll(tokenRes.Body)
	if tokenRes.StatusCode != 200 {
		fmt.Printf("%shandleSSFRecieverOauthCallback() - Token call Error: %+v\n", _logPrefix, string(body))
		return
	}
	var tokenResponse types.TokenReponse
	if err = json.Unmarshal(body, &tokenResponse); err != nil {
		fmt.Printf("%shandleSSFRecieverOauthCallback() - Token Json parse Error: %+v\n", _logPrefix, err)
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
