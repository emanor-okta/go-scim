package utils

import (
	"log"
	"os"

	"gopkg.in/yaml.v3"

	"github.com/emanor-okta/go-scim/types"
)

// Configuration Instance
var Config *Configuration

type ProxyFilterURL struct {
	URL      string `json:"url"`
	REQUEST  bool   `json:"request"`
	RESPONSE bool   `json:"response"`
	POST     bool   `json:"post"`
	PUT      bool   `json:"put"`
	GET      bool   `json:"get"`
	PATCH    bool   `json:"patch"`
	DELETE   bool   `json:"del"`
	OPTIONS  bool   `json:"options"`
}

type Services struct {
	Scim,
	Proxy,
	Hooks,
	Ssf,
	Dpop bool
}

type Hooks struct {
	FilterHooks bool
	AutoRespond bool
	Token,
	Saml,
	Password,
	Registration,
	Telephony,
	UserImport string
}

type Dpop struct {
	FlowType,
	Issuer,
	Code,
	CodeVerifier,
	RedirectURI,
	ClientId,
	ClientSecret,
	AssertPem,
	AssertKid,
	DpopPem,
	Port,
	ApiEndpoint,
	ApiMethod,
	Scopes string
	DebugNet bool
}

type Entitlements struct {
	ResourceTypes,
	Schemas []byte
	Resources map[string][]byte
	// Schemas map[string][]byte
}

type Configuration struct {
	Build     string
	ReqFilter *ReqFilter
	Services
	Hooks
	Dpop
	Entitlements
	CommonScimMiddlewares []types.Middleware
	Redis                 struct {
		Address  string
		Password string
		Db       int
	}
	Server struct {
		Address                       string
		Public_address                string
		Web_address                   string
		Web_console                   bool
		Debug_headers                 bool
		Debug_body                    bool
		Debug_query                   bool
		Log_messages                  bool
		Proxy_messages                bool
		Proxy_address                 string
		Proxy_port                    int
		Proxy_origin                  string
		Proxy_sni                     string
		ProxyDisabled                 bool
		ProxyFilterIps                bool
		Filter_ips                    bool
		Okta_public_ips_url           string
		Allowed_ips                   map[string]string
		Unauthorized_ips_oauth_config struct {
			Issuer,
			Client_id,
			Client_secret,
			Scopes,
			Redirect_uri string
		}
	}
	Scim struct {
		Enable_groups     bool
		ServerBaseAddress string
		UsersEndpoint     string
		GroupsEndpoint    string
	}
	WebMessageFilter struct {
		UserPostRequest      bool
		UserPostResponse     bool
		UserGetResponse      bool
		UserIdPutRequest     bool
		UserIdPutResponse    bool
		UserIdPatchRequest   bool
		UserIdPatchResponse  bool
		UserIdGetResponse    bool
		GroupsGetResponse    bool
		GroupsPostRequest    bool
		GroupsPostResponse   bool
		GroupsIdGetResponse  bool
		GroupsIdPutRequest   bool
		GroupsIdPutResponse  bool
		GroupsIdPatchRequest bool
	}

	ProxyMessageFilter struct {
		// RequestMessages  map[string]map[string]bool
		// ResponseMessages map[string]map[string]bool

		FilterURLs map[string]ProxyFilterURL
		// FilterURLs map[string]map[string]bool
		// RequestMessages  map[string]ProxyMessageFilterMethods
		// ResponseMessages map[string]ProxyMessageFilterMethods
		FilterMessages bool
	}
}

func LoadConfig(c string) *Configuration {
	var config Configuration
	buf, err := os.ReadFile(c)
	if err != nil {
		log.Fatalf("No Configuration file exists: %v\n", err)
	}

	err = yaml.Unmarshal(buf, &config)
	if err != nil {
		log.Fatal(err)
	}
	// config.ProxyMessageFilter.RequestMessages = make(map[string]map[string]bool)
	// config.ProxyMessageFilter.ResponseMessages = make(map[string]map[string]bool)
	// config.ProxyMessageFilter.RequestMessages = make(map[string]ProxyMessageFilterMethods)
	// config.ProxyMessageFilter.ResponseMessages = make(map[string]ProxyMessageFilterMethods)
	// config.ProxyMessageFilter.FilterURLs = make(map[string]map[string]bool)
	config.ProxyMessageFilter.FilterURLs = make(map[string]ProxyFilterURL)
	config.ProxyMessageFilter.FilterMessages = false

	config.Hooks = Hooks{}

	// config.Server.ProxyFilterIps = true

	//TEST
	// config.ProxyMessageFilter.FilterMessages = true
	// config.ProxyMessageFilter.FilterURLs["/get"] = ProxyFilterURL{URL: "/get", GET: true, REQUEST: true, RESPONSE: false}
	// config.ProxyMessageFilter.FilterURLs["/put"] = ProxyFilterURL{URL: "/put", PUT: true, REQUEST: true, RESPONSE: false}
	// config.ProxyMessageFilter.FilterURLs["https://httpbin.org/get"]["POST"] = true
	// config.ProxyMessageFilter.FilterURLs["https://httpbin.org/get"]["REQUEST"] = true
	// config.ProxyMessageFilter.FilterURLs["https://httpbin.org/get"]["RESPONSE"] = false
	// v1, ok := config.ProxyMessageFilter.FilterURLs["https://httpbin.org/get"]
	// if ok {
	// 	yes, ok := v1["POST"]
	// 	if ok {
	// 		fmt.Printf("DIDNT THROW IN CHECK: %v\n", yes)
	// 	}
	// }

	//END TEST

	Config = &config
	return &config
}
