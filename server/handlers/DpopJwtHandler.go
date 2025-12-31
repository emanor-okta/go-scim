package handlers

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httputil"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/lestrrat-go/jwx/v2/jwa"
	"github.com/lestrrat-go/jwx/v2/jwk"
	"github.com/lestrrat-go/jwx/v2/jws"

	"github.com/gorilla/websocket"

	"github.com/emanor-okta/go-scim/utils"
)

type JwtPayload struct {
	Htm   string `json:"htm,omitempty"`
	Htu   string `json:"htu,omitempty"`
	Iat   int64  `json:"iat,omitempty"`
	Nonce string `json:"nonce,omitempty"`
	Jti   string `json:"jti,omitempty"`
	Ath   string `json:"ath,omitempty"`
}

type TokenResponse struct {
	TokenType        string `json:"token_type,omitempty"`
	Scope            string `json:"scope,omitempty"`
	ExpiresIn        int64  `json:"expires_in,omitempty"`
	AccessToken      string `json:"access_token,omitempty"`
	IdToken          string `json:"id_token,omitempty"`
	Error            string `json:"error,omitempty"`
	ErrorDescription string `json:"error_description,omitempty"`
}

type AssertionPayload struct {
	Aud string `json:"aud,omitempty"`
	Iss string `json:"iss,omitempty"`
	Sub string `json:"sub,omitempty"`
	Exp int64  `json:"exp,omitempty"`
}

type loggingTransport struct{}

const (
	auth_code_payload          = "grant_type=%s&redirect_uri=%s&client_id=%s&code=%s"
	client_credentials_payload = "grant_type=%s&scope=%s&client_assertion_type=%s&client_assertion=%s"
)

func (s *loggingTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	bytes, _ := httputil.DumpRequestOut(r, true)

	resp, err := http.DefaultTransport.RoundTrip(r)
	// err is returned after dumping the response
	respBytes, _ := httputil.DumpResponse(resp, true)
	bytes = append(bytes, respBytes...)

	fmt.Printf("\n%s\n", bytes)
	debug = fmt.Sprintf("\n%s\n\n%s", debug, bytes)

	return resp, err
}

// var flowParams utils.Dpop
var privkey, pubkey jwk.Key
var jwtPayload JwtPayload

var result string
var debug string

func init() {
	// auth code flow - start callback server
	// http.HandleFunc("/dpop/callback", handleCallbackReq)
	// http.HandleFunc("/dpop/generate_dpop", handleGenerateDpop)
	// http.HandleFunc("/dpop", handleDpop)
}

func generate() (string, error) {
	result = ""
	debug = ""
	// flowParams = utils.Config.Dpop
	//flowParams = parseCommandLineArgs()
	if utils.Config.Dpop.FlowType == "jwt" {
		assertion, err := generateAssertion(utils.Config.Dpop)
		if err != nil {
			return "", err
		}

		result = fmt.Sprintf(`{"jwt":"%s"}`, assertion)
		fmt.Printf("JWT Credential:\n%s\n", result)
	} else if utils.Config.Dpop.Port == "" {
		// client credentials or auth code was provided
		if utils.Config.Dpop.ApiEndpoint == "" {
			// no DPoP m2m, get access_token
			tokenPayload, err := generateTokenPayload(utils.Config.Dpop)
			if err != nil {
				return "", err
			}

			tokenResponse, _, err := tokenCall("POST", getHtu(utils.Config.Dpop.Issuer), "", []byte(""), tokenPayload)
			if err != nil {
				return "", err
			}

			bytes, _ := json.Marshal(tokenResponse)
			result = string(bytes)
		} else {
			// DPoP m2m or web (with auth code provided), start token calls
			var err error
			result, err = getTokens()
			if err != nil {
				return "", err
			}
			// reader := bufio.NewReader(os.Stdin)
			// for {
			// 	fmt.Print("Press Enter to generate new DPoP or 'q' to quit: ")
			// 	input, _ := reader.ReadString('\n')
			// 	fmt.Println(input)
			// 	if strings.TrimSpace(input) == "q" {
			// 		return
			// 	}
			// 	fmt.Printf("\n\nDPoP: %s\n\n", generateDpop())
			// }
		}
	} else {
		// auth code flow - start callback server
		// http.HandleFunc("/callback", handleCallbackReq)
		// http.HandleFunc("/generate_dpop", handleGenerateDpop)
		// if err := http.ListenAndServe(fmt.Sprintf(":%s", flowParams.Port), nil); err != nil {
		// 	log.Fatalf("\nError, Server startup failed: %s\n", err)
		// }
	}

	return result, nil
}

// func generateDpop() (string, error) {
// 	// generates another DPoP since each API call with the access_token requires a unique DPoP sent with the request
// 	jwtPayload.Jti = uuid.NewString()
// 	signedBytes, err := signJwt(jwtPayload, jws.WithKey(jwa.RS256, privkey, jws.WithProtectedHeaders(generateJwtHeader(pubkey))))
// 	if err != nil {
// 		return "", err
// 	}

// 	return string(signedBytes), nil
// }

func getTokens() (string, error) {
	var err error
	privkey, pubkey, err = getOrGenerateDpopKey(utils.Config.Dpop.DpopPem)
	if err != nil {
		return "", err
	}

	fmt.Println("\nDPoP Private Key:")
	json.NewEncoder(os.Stdout).Encode(privkey)
	fmt.Println("\nDPoP Public Key:")
	json.NewEncoder(os.Stdout).Encode(pubkey)

	// token call 1
	jwtPayload = JwtPayload{
		Htm: "POST",
		Htu: getHtu(utils.Config.Dpop.Issuer),
		Iat: time.Now().Unix(),
	}
	dPop, err := signJwt(jwtPayload, jws.WithKey(jwa.RS256, privkey, jws.WithProtectedHeaders(generateJwtHeader(pubkey))))
	if err != nil {
		return "", err
	}

	fmt.Printf("\nDPoP JWT:\n%s\n", dPop)
	debug = fmt.Sprintf("\n%s\nDPoP JWT:\n%s\n", debug, dPop)
	tokenPayload, err := generateTokenPayload(utils.Config.Dpop)
	if err != nil {
		return "", err
	}

	resp1, nonce, err := tokenCall(jwtPayload.Htm, jwtPayload.Htu, utils.Config.Dpop.Code, dPop, tokenPayload)
	if err != nil {
		return "", err
	}

	fmt.Printf("\nToken Call Response 1: \n%+v\n", resp1)
	debug = fmt.Sprintf("\n%s\nToken Call Response 1: \n%+v\n", debug, resp1)
	fmt.Printf("\nnonce: %v\n", nonce)
	debug = fmt.Sprintf("\n%s\nnonce: %v\n", debug, nonce)
	if nonce == "" {
		log.Printf("\nExpected \"dpop-nonce\" http header but was not present")
		return "", fmt.Errorf("expected dpop-nonce http header but was not present")
	}

	// token call 2
	jwtPayload = JwtPayload{
		Htm:   "POST",
		Htu:   getHtu(utils.Config.Dpop.Issuer),
		Iat:   time.Now().Unix(),
		Nonce: nonce,
		Jti:   uuid.NewString(),
	}
	dPop, err = signJwt(jwtPayload, jws.WithKey(jwa.RS256, privkey, jws.WithProtectedHeaders(generateJwtHeader(pubkey))))
	if err != nil {
		return "", err
	}

	tokenPayload, err = generateTokenPayload(utils.Config.Dpop)
	if err != nil {
		return "", err
	}

	resp2, _, err := tokenCall(jwtPayload.Htm, jwtPayload.Htu, utils.Config.Dpop.Code, dPop, tokenPayload)
	if err != nil {
		return "", err
	}

	fmt.Printf("\nToken Call Response 2: \n%+v\n", resp2)
	debug = fmt.Sprintf("\n%s\nToken Call Response 2: \n%+v\n", debug, resp2)

	jwtPayload = JwtPayload{
		Htm: utils.Config.Dpop.ApiMethod,
		Htu: utils.Config.Dpop.ApiEndpoint,
	}
	if !strings.Contains(utils.Config.Dpop.Issuer, "/oauth2/") {
		// Add ath value for o4o
		jwtPayload.Ath = generateAth(resp2.AccessToken)
		jwtPayload.Iat = time.Now().Unix()
		jwtPayload.Jti = uuid.NewString()
	}
	dPop, err = signJwt(jwtPayload, jws.WithKey(jwa.RS256, privkey, jws.WithProtectedHeaders(generateJwtHeader(pubkey))))
	if err != nil {
		return "", err
	}

	values := getApiRequestValues(resp2.AccessToken, string(dPop))
	fmt.Println(values)
	return values, nil
}

func getApiRequestValues(authorization, dPop string) string {
	// return fmt.Sprintf("\n\n-------- DPoP Bound Request Values --------\nAuthorization: DPoP %s\n\nDPoP: %s\n\n", authorization, dPop)
	return fmt.Sprintf(`{"Authorization": "DPoP %s", "DPoP": "%s"}`, authorization, dPop)
}

func generateAth(accessToken string) string {
	sum := sha256.Sum256([]byte(accessToken))
	ath := base64.RawURLEncoding.EncodeToString(sum[:])
	fmt.Printf("\nATH Token:\n%s\n\nATH value:\n%s\n", accessToken, ath)
	debug = fmt.Sprintf("\n%s\nATH Token:\n%s\n\nATH value:\n%s\n", debug, accessToken, ath)
	return ath
}

// func generateKey() (jwk.Key, jwk.Key, error) {
// 	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
// 	if err != nil {
// 		log.Printf("\nError, Generating RSA Private Key: %v\n", err)
// 		return nil, nil, fmt.Errorf("error generating RSA private key: %v", err)
// 	}

// 	jwkKey, err := jwk.FromRaw(privateKey)
// 	if err != nil {
// 		log.Printf("\nError, Generating JWK Key from RSA Private Key: %v\n", err)
// 		return nil, nil, fmt.Errorf("error generating JWK key from  RSA private key: %v", err)
// 	}

// 	pubKey, err := jwkKey.PublicKey()
// 	if err != nil {
// 		log.Printf("\nError, Getting Public Key Part from JWK: %v\n", err)
// 		return nil, nil, fmt.Errorf("error getting public key part from JWK: %v", err)
// 	}

// 	return jwkKey, pubKey, nil
// }

func getKeys(keyAsPem []byte) (jwk.Key, jwk.Key, error) {
	privkey, err := jwk.ParseKey(keyAsPem, jwk.WithPEM(true))
	if err != nil {
		log.Printf("\nfailed to parse JWK: %s\n", err)
		return nil, nil, fmt.Errorf("failed to parse JWK: %s", err)
	}

	pubkey, err := jwk.PublicKeyOf(privkey)
	if err != nil {
		log.Printf("\nfailed to get public key: %s\n", err)
		return nil, nil, fmt.Errorf("failed to get public key: %s", err)
	}

	return privkey, pubkey, nil
}

func signJwt(jwtPayload JwtPayload, options ...jws.SignOption) ([]byte, error) {
	jwtPayloadBytes, _ := json.Marshal(jwtPayload)
	buf, err := jws.Sign(jwtPayloadBytes, options...)
	if err != nil {
		log.Printf("\nFailed to signJWT: %+v, with options: %+v\n", jwtPayload, options)
		return nil, fmt.Errorf("failed to sign JWT: %+v, with options: %+v", jwtPayload, options)
	}

	return buf, nil
}

func generateJwtHeader(k jwk.Key) jws.Headers {
	hdrs := jws.NewHeaders()
	hdrs.Set("typ", "dpop+jwt")
	hdrs.Set("alg", "RS256")
	hdrs.Set("jwk", k)
	return hdrs
}

func generateTokenPayload(fp utils.Dpop) (*strings.Reader, error) {
	var payload string
	if fp.FlowType == "web" {
		var redirectUri string
		grantType := "authorization_code"
		if fp.RedirectURI == "" {
			// redirectUri = fmt.Sprintf("http://localhost:%s/callback", fp.Port)
			redirectUri = utils.Config.Dpop.RedirectURI
		} else {
			redirectUri = fp.RedirectURI
		}
		payload = fmt.Sprintf(auth_code_payload, grantType, redirectUri, fp.ClientId, fp.Code)
		if fp.ClientSecret != "" {
			payload = fmt.Sprintf("%s&client_secret=%s", payload, fp.ClientSecret)
		}
		if fp.CodeVerifier != "" {
			payload = fmt.Sprintf("%s&code_verifier=%s", payload, fp.CodeVerifier)
		}
	} else {
		grantType := "client_credentials"
		assertionType := "urn:ietf:params:oauth:client-assertion-type:jwt-bearer"
		assertion, err := generateAssertion(fp)
		if err != nil {
			return nil, err
		}

		payload = fmt.Sprintf(client_credentials_payload, grantType, strings.ReplaceAll(fp.Scopes, ",", " "), assertionType, assertion)
	}

	return strings.NewReader(payload), nil
}

func tokenCall(method, url, _ string, dpop []byte, payload *strings.Reader) (TokenResponse, string, error) {
	tokenResponse := TokenResponse{}
	dpopNonceHeader, tokenResp, err := httpRequest(method, url, "application/x-www-form-urlencoded", "", string(dpop), payload)
	if err != nil {
		log.Printf("Error making /token call: %+v\n", err)
		return tokenResponse, "", err
	}
	if err := json.Unmarshal([]byte(tokenResp), &tokenResponse); err != nil {
		log.Printf("\nError UnMarshalling /token Response: %+v\n", err)
		return tokenResponse, "", err
	}
	return tokenResponse, dpopNonceHeader, nil
}

func httpRequest(method, url, contentType, authorization, dpop string, payload *strings.Reader) (string, string, error) {
	var httpClient *http.Client
	if method == "" {
		return "", "", fmt.Errorf("http method not specified")
	}
	if !strings.Contains(url, "https://") {
		return "", "", fmt.Errorf("http url not specified")
	}
	if contentType == "" {
		return "", "", fmt.Errorf("http contentType not specified")
	}

	if utils.Config.Dpop.DebugNet {
		httpClient = &http.Client{Transport: &loggingTransport{}}
	} else {
		httpClient = &http.Client{}
	}
	req, err := http.NewRequest(method, url, payload)
	if err != nil {
		log.Printf("Error generating http request, %v\n", err)
		return "", "", err
	}
	req.Header.Add("Accept", "application/json")
	req.Header.Add("Content-Type", contentType)
	if dpop != "" {
		req.Header.Add("DPoP", dpop)
	}
	// fmt.Println(dpop)
	if authorization != "" {
		req.Header.Add("Authorization", authorization)
	}

	res, err := httpClient.Do(req)
	if err != nil {
		log.Printf("Error making http call, %v\n", err)
		return "", "", err
	}

	defer res.Body.Close()
	dpopNonce := ""
	if dpopNonceHeaders := res.Header.Values("dpop-nonce"); len(dpopNonceHeaders) > 0 {
		dpopNonce = dpopNonceHeaders[0]
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		log.Printf("Error reading http response body, %v\n", err)
		return "", "", err
	}

	// fmt.Println(string(body))
	return dpopNonce, string(body), nil
}

func getOrGenerateDpopKey(keyAsPemFile string) (jwk.Key, jwk.Key, error) {
	if keyAsPemFile == "" {
		// return generateKey()
		return utils.GenerateKey()
	} else {
		pem, err := os.ReadFile(keyAsPemFile)
		if err != nil {
			log.Printf("\nError, Reading Key file for DPoP, generating a key instead, %+v\n", err)
			// return generateKey()
			return utils.GenerateKey()
		}
		return getKeys(pem)
	}
}

func generateAssertion(fp utils.Dpop) (string, error) {
	if fp.AssertPem == "" {
		log.Printf("\nError, Private Key used to sign Assertion not present\n")
		return "", fmt.Errorf("private key used to sign Assertion not present")
	}

	// pem, err := os.ReadFile(fp.AssertPem)
	// if err != nil {
	// 	log.Fatalf("\nError, Reading Key file for JWT Assertion, %+v\n", err)
	// }

	privkey, _, err := getKeys([]byte(fp.AssertPem))
	if err != nil {
		return "", err
	}

	assertion := AssertionPayload{
		Aud: fmt.Sprintf("%s/oauth2/v1/token", fp.Issuer),
		Iss: fp.ClientId,
		Sub: fp.ClientId,
		Exp: time.Now().Unix(),
	}
	payload, _ := json.Marshal(assertion)
	hdrs := jws.NewHeaders()
	if kid := fp.AssertKid; kid != "" {
		hdrs.Set(`kid`, kid)
	}

	jwt, err := jws.Sign([]byte(payload), jws.WithKey(jwa.RS256, privkey, jws.WithProtectedHeaders(hdrs)))
	if err != nil {
		log.Printf("\nError, failed to sign assertion: %s\n", err)
		return "", fmt.Errorf("failed to sign assertion: %s", err)
	}

	return string(jwt), nil
}

func getHtu(htu string) string {
	if strings.Contains(htu, "/oauth2/") {
		return fmt.Sprintf("%s/v1/token", utils.Config.Dpop.Issuer)
	} else {
		return fmt.Sprintf("%s/oauth2/v1/token", utils.Config.Dpop.Issuer)
	}
}

func HandleCallbackReq(res http.ResponseWriter, req *http.Request) {
	code := req.URL.Query().Get("code")
	utils.Config.Dpop.Code = code
	values, err := getTokens()
	if err != nil {
		// res.WriteHeader(http.StatusBadRequest)
		res.Write([]byte(fmt.Sprintf(`{ "error": "%s" }`, err.Error())))
		return
	}

	res.Write([]byte(values))
	sendToDpopUI([]byte(strings.TrimSpace(debug)))
}

func HandleGenerateDpop(res http.ResponseWriter, req *http.Request) {
	// dpop := generateDpop()
	// res.Write([]byte(dpop))
	result, err := generate()
	if err != nil {
		//res.WriteHeader(http.StatusBadRequest)
		res.Write([]byte(fmt.Sprintf(`{ "error": "%s" }`, err.Error())))
		return
	}
	// m := map[string]string{"result": result}
	// b, _ := json.Marshal(m)
	res.Write([]byte(result))
	sendToDpopUI([]byte(strings.TrimSpace(debug)))
}

func HandleDpop(res http.ResponseWriter, req *http.Request) {
	// dpop := generateDpop()
	// res.Write([]byte(dpop))
	// fmt.Printf("%+v\n", utils.Config.Dpop)

	// make sure one of the tabs is selected
	if utils.Config.Dpop.FlowType == "" {
		utils.Config.Dpop.FlowType = "jwt"
	}

	err := tpl.ExecuteTemplate(res, "dpopjwt.gohtml", struct {
		utils.Services
		utils.Dpop
	}{utils.Config.Services, utils.Config.Dpop})
	if err != nil {
		log.Printf("HandleDpop: %+v\n", err)
	}
}

func HandleDpopKeyUpload(res http.ResponseWriter, req *http.Request) {
	//bytes, err := utils.GetBody(req)
	err := req.ParseForm()
	if err != nil {
		log.Printf("HandleDpopKeyUpload: Error ParseForm, %s\n", err)
		res.WriteHeader(http.StatusBadRequest)
		return
	}

	switch req.RequestURI {
	case "/dpop/jwt-config":
		utils.Config.Dpop.FlowType = "jwt"
	case "/dpop/service-config":
		utils.Config.Dpop.FlowType = "service"
	case "/dpop/auth-config":
		utils.Config.Dpop.FlowType = "web"
	default:
		log.Printf("HandleDpopKeyUpload: unkown request URI: %s\n", req.RequestURI)
	}

	values := req.Form
	for k, v := range values {
		// fmt.Printf("%s - %s\n", k, v[0])
		switch k {
		case "issuer":
			utils.Config.Dpop.Issuer = v[0]
		case "client-id":
			utils.Config.Dpop.ClientId = v[0]
		case "priv-key-enc":
			// decodedKey, _ := base64.RawStdEncoding.DecodeString(v[0])
			if v[0] != "" {
				decodedKey, _ := base64.StdEncoding.DecodeString(v[0])
				utils.Config.Dpop.AssertPem = string(decodedKey)
			}
		case "priv-key-id":
			utils.Config.Dpop.AssertKid = v[0]
		case "scopes":
			utils.Config.Dpop.Scopes = v[0]
		case "dpop-key-enc":
			if v[0] != "" {
				decodedKey, _ := base64.StdEncoding.DecodeString(v[0])
				utils.Config.Dpop.DpopPem = string(decodedKey)
			}
		case "client-secret":
			utils.Config.Dpop.ClientSecret = v[0]
		case "service-endpoint", "auth-endpoint":
			utils.Config.Dpop.ApiEndpoint = v[0]
		case "service-method", "auth-method":
			utils.Config.Dpop.ApiMethod = v[0]
		case "auth-code":
			utils.Config.Dpop.Code = v[0]
		case "auth-code-verifier":
			utils.Config.Dpop.CodeVerifier = v[0]
		case "redirect-uri":
			utils.Config.Dpop.RedirectURI = v[0]
		case "port":
			utils.Config.Dpop.Port = v[0]
		default:
			log.Printf("HandleDpopKeyUpload: un-used request param : %s - %s\n", k, v[0])
		}
	}
	// fmt.Printf("%+v\n", utils.Config.Dpop)
	http.Redirect(res, req, "/dpop", http.StatusTemporaryRedirect)
}

func HandleDpopKeyRemoval(res http.ResponseWriter, req *http.Request) {
	keyType := req.URL.Query().Get("type")
	if keyType == "private" {
		// Assertion signing key
		utils.Config.Dpop.AssertPem = ""
		utils.Config.Dpop.AssertKid = ""
	} else {
		// DPoP Key
		utils.Config.Dpop.DpopPem = ""
	}

	res.WriteHeader(http.StatusOK)
}

/*
 * WS Support
 */
var dpopWsClientConnected bool
var dpopWsConn *websocket.Conn

func HandleDpopWebSocketUpgrade(res http.ResponseWriter, req *http.Request) {
	dpopWsConn = utils.HandleWebSocketUpgrade(res, req, &dpopWsClientConnected)
	if dpopWsClientConnected && dpopWsConn != nil {
		go utils.WsPingOnlyReader(dpopWsConn, &dpopWsClientConnected)
	}
}

/*
Helpers
*/
func sendToDpopUI(debugData []byte) {
	if dpopWsClientConnected {
		err := dpopWsConn.WriteMessage(websocket.TextMessage, debugData)
		if err != nil {
			log.Printf("sendToDpopUI: sendToUI: Error sending WS message: %s\n", err)
		}
	}
}
