package handlers

import (
	"crypto/rand"
	"crypto/rsa"
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

// type FlowParams struct {
// 	Type,
// 	Issuer,
// 	Code,
// 	CodeVerifier,
// 	RedirectURI,
// 	ClientId,
// 	ClientSecret,
// 	AssertPem,
// 	AssertKid,
// 	DpopPem,
// 	Port,
// 	ApiEndpoint,
// 	ApiMethod,
// 	Scopes string
// 	DebugNet bool
// }

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

func generate() string {
	result = ""
	debug = ""
	// flowParams = utils.Config.Dpop
	//flowParams = parseCommandLineArgs()
	if utils.Config.Dpop.FlowType == "jwt" {
		result = "{\"jwt\":\"" + generateAssertion(utils.Config.Dpop) + "\"}"
		fmt.Printf("JWT Credential:\n%s\n", result)
	} else if utils.Config.Dpop.Port == "" {
		// client credentials or auth code was provided
		if utils.Config.Dpop.ApiEndpoint == "" {
			// no DPoP m2m, get access_token
			tokenResponse, _ := tokenCall("POST", getHtu(utils.Config.Dpop.Issuer), "", []byte(""), generateTokenPayload(utils.Config.Dpop))
			bytes, _ := json.Marshal(tokenResponse)
			result = string(bytes)
		} else {
			// DPoP m2m or web (with auth code provided), start token calls
			result = getTokens()
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

	return result
}

func generateDpop() string {
	// generates another DPoP since each API call with the access_token requires a unique DPoP sent with the request
	jwtPayload.Jti = uuid.NewString()
	return string(signJwt(jwtPayload, jws.WithKey(jwa.RS256, privkey, jws.WithProtectedHeaders(generateJwtHeader(pubkey)))))
}

func getTokens() string {
	privkey, pubkey = getOrGenerateDpopKey(utils.Config.Dpop.DpopPem)

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
	dPop := signJwt(jwtPayload, jws.WithKey(jwa.RS256, privkey, jws.WithProtectedHeaders(generateJwtHeader(pubkey))))
	fmt.Printf("\nDPoP JWT:\n%s\n", dPop)
	debug = fmt.Sprintf("\n%s\nDPoP JWT:\n%s\n", debug, dPop)
	resp1, nonce := tokenCall(jwtPayload.Htm, jwtPayload.Htu, utils.Config.Dpop.Code, dPop, generateTokenPayload(utils.Config.Dpop))
	fmt.Printf("\nToken Call Response 1: \n%+v\n", resp1)
	debug = fmt.Sprintf("\n%s\nToken Call Response 1: \n%+v\n", debug, resp1)
	fmt.Printf("\nnonce: %v\n", nonce)
	debug = fmt.Sprintf("\n%s\nnonce: %v\n", debug, nonce)
	if nonce == "" {
		log.Printf("\nExpected \"dpop-nonce\" http header but was not present")
		return "\nERROR: Expected \"dpop-nonce\" http header but was not present"
	}

	// token call 2
	jwtPayload = JwtPayload{
		Htm:   "POST",
		Htu:   getHtu(utils.Config.Dpop.Issuer),
		Iat:   time.Now().Unix(),
		Nonce: nonce,
		Jti:   uuid.NewString(),
	}
	dPop = signJwt(jwtPayload, jws.WithKey(jwa.RS256, privkey, jws.WithProtectedHeaders(generateJwtHeader(pubkey))))
	resp2, _ := tokenCall(jwtPayload.Htm, jwtPayload.Htu, utils.Config.Dpop.Code, dPop, generateTokenPayload(utils.Config.Dpop))
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
	dPop = signJwt(jwtPayload, jws.WithKey(jwa.RS256, privkey, jws.WithProtectedHeaders(generateJwtHeader(pubkey))))

	values := getApiRequestValues(resp2.AccessToken, string(dPop))
	fmt.Println(values)
	return values
}

func getApiRequestValues(authorization, dPop string) string {
	return fmt.Sprintf("\n\n-------- DPoP Bound Request Values --------\nAuthorization: DPoP %s\n\nDPoP: %s\n\n", authorization, dPop)
}

func generateAth(accessToken string) string {
	sum := sha256.Sum256([]byte(accessToken))
	ath := base64.RawURLEncoding.EncodeToString(sum[:])
	fmt.Printf("\nATH Token:\n%s\n\nATH value:\n%s\n", accessToken, ath)
	debug = fmt.Sprintf("\n%s\nATH Token:\n%s\n\nATH value:\n%s\n", debug, accessToken, ath)
	return ath
}

func generateKey() (jwk.Key, jwk.Key) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		log.Fatalf("\nError, Generating RSA Private Key: %v\n", err)
	}

	jwkKey, err := jwk.FromRaw(privateKey)
	if err != nil {
		log.Fatalf("\nError, Generating JWK Key from RSA Private Key: %v\n", err)
	}

	pubKey, err := jwkKey.PublicKey()
	if err != nil {
		log.Fatalf("\nError, Getting Public Key Part from JWK: %v\n", err)
	}

	return jwkKey, pubKey
}

func getKeys(keyAsPem []byte) (jwk.Key, jwk.Key) {
	privkey, err := jwk.ParseKey(keyAsPem, jwk.WithPEM(true))
	if err != nil {
		log.Fatalf("\nfailed to parse JWK: %s\n", err)
	}

	pubkey, err := jwk.PublicKeyOf(privkey)
	if err != nil {
		log.Fatalf("\nfailed to get public key: %s\n", err)
	}

	return privkey, pubkey
}

func signJwt(jwtPayload JwtPayload, options ...jws.SignOption) []byte {
	jwtPayloadBytes, _ := json.Marshal(jwtPayload)
	buf, err := jws.Sign(jwtPayloadBytes, options...)
	if err != nil {
		log.Fatalf("\nFailed to signJWT: %+v, with options: %+v\n", jwtPayload, options)
	}

	return buf
}

func generateJwtHeader(k jwk.Key) jws.Headers {
	hdrs := jws.NewHeaders()
	hdrs.Set("typ", "dpop+jwt")
	hdrs.Set("alg", "RS256")
	hdrs.Set("jwk", k)
	return hdrs
}

func generateTokenPayload(fp utils.Dpop) *strings.Reader {
	var payload string
	if fp.FlowType == "web" {
		var redirectUri string
		grantType := "authorization_code"
		if fp.RedirectURI == "" {
			redirectUri = fmt.Sprintf("http://localhost:%s/callback", fp.Port)
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
		payload = fmt.Sprintf(client_credentials_payload, grantType, strings.ReplaceAll(fp.Scopes, ",", " "), assertionType, generateAssertion(fp))
	}

	return strings.NewReader(payload)
}

func tokenCall(method, url, _ string, dpop []byte, payload *strings.Reader) (TokenResponse, string) {
	dpopNonceHeader, tokenResp, err := httpRequest(method, url, "application/x-www-form-urlencoded", "", string(dpop), payload)
	if err != nil {
		log.Fatalf("Error making /token call: %+v\n", err)
	}
	tokenResponse := TokenResponse{}
	if err := json.Unmarshal([]byte(tokenResp), &tokenResponse); err != nil {
		log.Fatalf("\nError UnMarshalling /token Response: %+v\n", err)
	}
	return tokenResponse, dpopNonceHeader
}

func httpRequest(method, url, contentType, authorization, dpop string, payload *strings.Reader) (string, string, error) {
	var httpClient *http.Client
	if utils.Config.Dpop.DebugNet {
		httpClient = &http.Client{Transport: &loggingTransport{}}
	} else {
		httpClient = &http.Client{}
	}
	req, err := http.NewRequest(method, url, payload)
	if err != nil {
		fmt.Println(err)
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
		log.Fatal(err)
		return "", "", err
	}

	defer res.Body.Close()
	dpopNonce := ""
	if dpopNonceHeaders := res.Header.Values("dpop-nonce"); len(dpopNonceHeaders) > 0 {
		dpopNonce = dpopNonceHeaders[0]
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		fmt.Println(err)
		return "", "", err
	}

	// fmt.Println(string(body))
	return dpopNonce, string(body), nil
}

func getOrGenerateDpopKey(keyAsPemFile string) (jwk.Key, jwk.Key) {
	if keyAsPemFile == "" {
		return generateKey()
	} else {
		pem, err := os.ReadFile(keyAsPemFile)
		if err != nil {
			log.Printf("\nError, Reading Key file for DPoP, generating a key instead, %+v\n", err)
			return generateKey()
		}
		return getKeys(pem)
	}
}

func generateAssertion(fp utils.Dpop) string {
	if fp.AssertPem == "" {
		log.Fatalf("\nError, 'assertion_pem_file=<file>' option not present and needed for this flow\n")
	}

	// pem, err := os.ReadFile(fp.AssertPem)
	// if err != nil {
	// 	log.Fatalf("\nError, Reading Key file for JWT Assertion, %+v\n", err)
	// }

	privkey, _ := getKeys([]byte(fp.AssertPem))
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
		log.Fatalf("\nError, failed to sign assertion: %s\n", err)
	}

	return string(jwt)
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
	values := getTokens()
	res.Write([]byte(values))
}

func HandleGenerateDpop(res http.ResponseWriter, req *http.Request) {
	// dpop := generateDpop()
	// res.Write([]byte(dpop))
	result := generate()
	// m := map[string]string{"result": result}
	// b, _ := json.Marshal(m)
	res.Write([]byte(result))
}

func HandleDpop(res http.ResponseWriter, req *http.Request) {
	// dpop := generateDpop()
	// res.Write([]byte(dpop))
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
		fmt.Printf("%s - %s\n", k, v[0])
		switch k {
		case "issuer":
			utils.Config.Dpop.Issuer = v[0]
		case "client-id":
			utils.Config.Dpop.ClientId = v[0]
		case "priv-key-enc":
			// decodedKey, _ := base64.RawStdEncoding.DecodeString(v[0])
			decodedKey, _ := base64.StdEncoding.DecodeString(v[0])
			utils.Config.Dpop.AssertPem = string(decodedKey)
		case "priv-key-id":
			utils.Config.Dpop.AssertKid = v[0]
		case "scopes":
			utils.Config.Dpop.Scopes = v[0]
		case "dpop-key-enc":
			utils.Config.Dpop.DpopPem = v[0]
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
			log.Printf("HandleDpopKeyUpload: unknown request param: %s - %s\n", k, v[0])
		}
	}

	http.Redirect(res, req, "/dpop", http.StatusTemporaryRedirect)
}
func HandleDpopKeyUpload2(res http.ResponseWriter, req *http.Request) {
	//bytes, err := utils.GetBody(req)
	err := req.ParseMultipartForm(4096 * 4)
	if err != nil {
		log.Printf("HandleDpopKeyUpload: Error ParseMultipartForm, %s\n", err)
		res.WriteHeader(http.StatusBadRequest)
		return
	}
	fmt.Printf("%+v\n", req.MultipartForm)
	file, fileHeader, err := req.FormFile("priv_key")
	if err != nil {
		log.Printf("HandleDpopKeyUpload: Error FormFile, %s\n", err)
		res.WriteHeader(http.StatusBadRequest)
		return
	}
	bytes := make([]byte, 4096*4)
	read, err := file.Read(bytes)
	fmt.Printf("file name: %s\n%v\n", fileHeader.Filename, read)
	if err != nil || read < 1 {
		log.Printf("HandleDpopKeyUpload: Error Read, %s\n", err)
		res.WriteHeader(http.StatusBadRequest)
		return
	}
	fmt.Printf("file name: %s\n%s\n", fileHeader.Filename, string(bytes))
	if strings.Contains(req.RequestURI, "upload_priv_key") {
		utils.Config.Dpop.AssertPem = string(bytes)
	} else {
		utils.Config.Dpop.DpopPem = string(bytes)
	}
	fmt.Println(utils.Config.Dpop.AssertPem)
	res.WriteHeader(http.StatusAccepted)
}

func parseCommandLineArgs() utils.Dpop {

	flowParams := utils.Dpop{}
	if len(os.Args) < 2 {
		showHelp()
	}

	switch os.Args[1] {
	case "m2m":
		flowParams.FlowType = "service"
	case "web":
		flowParams.FlowType = "web"
	case "jwt":
		flowParams.FlowType = "jwt"
	default:
		showHelp()
	}

	for i := 2; i < len(os.Args); i++ {
		option := os.Args[i]
		if option == "-d" || option == "--debug" {
			flowParams.DebugNet = true
			continue
		}
		i = i + 1
		val := os.Args[i]
		fmt.Printf("option=%s, val=%s\n", option, val)

		switch option {
		case "-i", "--issuer":
			flowParams.Issuer = val
		case "-c", "--client-id":
			flowParams.ClientId = val
		case "-x", "--client-secret":
			flowParams.ClientSecret = val
		case "-v", "--code-verifier":
			flowParams.CodeVerifier = val
		case "-s", "--scopes":
			flowParams.Scopes = val
		case "-o", "--dpop-pem-file":
			flowParams.DpopPem = val
		case "-a", "--auth-code":
			flowParams.Code = val
		case "-r", "--redirect-uri":
			flowParams.RedirectURI = val
		case "-p", "--port":
			flowParams.Port = val
		case "-m", "--api-method":
			flowParams.ApiMethod = val
		case "-e", "--api-endpoint":
			flowParams.ApiEndpoint = val
		case "-j", "--jwt-pem-file":
			flowParams.AssertPem = val
		case "-k", "--jwt-kid":
			flowParams.AssertKid = val
		default:
			fmt.Printf("\nError, Invalid command line param supplied: %s\n", val)
		}
	}

	return flowParams
}

func showHelp() {
	fmt.Println("\nUsage:")
	fmt.Printf("%-2sgo run main.go [command]\n", "")

	fmt.Println("\nAvailable Commands:")
	fmt.Printf("  %-10sAuthorization Code\n", "web")
	fmt.Printf("  %-10sClient Credentials\n", "m2m")
	fmt.Printf("  %-10sGenerate JWT Credential for Oauth for Okta without DPoP\n", "jwt")

	fmt.Println("\nFlags:")
	fmt.Printf("  %-3s %-20s Okta Authorization Server\n", "-i,", "--issuer")
	fmt.Printf("  %-3s %-20s OIDC Client id of Okta App\n", "-c,", "--client-id")
	fmt.Printf("  %-3s %-20s OIDC Client Client Secret of Okta App (for web apps)\n", "-x,", "--client-secret")
	fmt.Printf("  %-3s %-20s OAuth Scopes Requested, comma seperated (ie okta.apps.read,okta.groups.manage)\n", "-s,", "--scopes")
	fmt.Printf("  %-3s %-20s OAuth Redirect URI\n", "-r,", "--redirect-uri")
	fmt.Printf("  %-3s %-20s PKCE code Verifier (for flows that use PKVE)\n", "-v,", "--code-verifier")
	fmt.Printf("  %-3s %-20s Authorization Code Value (needed for web flow if not redirecting to 'http://localhost:<port>/callback')\n", "-a,", "--auth-code")
	fmt.Printf("  %-3s %-20s For web flows if redirecting to this process port to run http server on (will start server on 'http://localhost:<port>/callback')\n", "-p,", "--port")
	fmt.Printf("  %-3s %-20s API endpoint the DPoP Access Token will be used for\n", "-e,", "--api-endpoint")
	fmt.Printf("  %-3s %-20s HTTP Method used with the DPoP Access Token (GET/POST/etc)\n", "-m,", "--api-method")
	fmt.Printf("  %-3s %-20s File location with PEM encoded private key to sign JWT (needed for o4o when using m2m)\n", "-j,", "--jwt-pem-file")
	fmt.Printf("  %-3s %-20s Key id of JWK registered in Okta\n", "-k,", "--jwt-key")
	fmt.Printf("  %-3s %-20s File location with PEM encoded private key to sign DPoP (if not specified a JWKS will dynamically be generated)\n", "-o,", "--dpop-pem-file")
	fmt.Printf("  %-3s %-20s Debug Network Requests and Responses\n", "-d,", "--debug")
	fmt.Printf("\n\n")
	os.Exit(0)
}
