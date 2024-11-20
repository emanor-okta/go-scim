package web

import (
	"bytes"
	"compress/gzip"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io"

	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/andybalholm/brotli"
	"github.com/klauspost/compress/zstd"

	"github.com/emanor-okta/go-scim/apps"
	messageLogs "github.com/emanor-okta/go-scim/server/log"
	"github.com/emanor-okta/go-scim/types"
	"github.com/emanor-okta/go-scim/utils"
	// br "github.com/google/brotli/go/cbrotli"
)

// const default_proxy_port = 8084
const http_header_scim_id = "X-Go-Scim-Id"
const proxy_msg = "proxy.gohtml"

var server *http.Server
var proxy *httputil.ReverseProxy

var publicAddress string

// func init() {
// 	// with default Mux can only add a specific route once so do in init() instead of startProxy()
// 	// commonMiddlewares is defined/set in webHandlers.go. Should probably be moved to config.
// 	http.HandleFunc("/", utils.AddMiddleware(handleProxy, commonMiddlewares...))
// }

func InitProxy(commonMiddlewares []types.Middleware) {
	// init() called before commonMiddlewares had filterIP middleware, use custom Init
	http.HandleFunc("/", utils.AddMiddleware(handleProxy, commonMiddlewares...))
	publicAddress = utils.Config.Server.Public_address
	publicAddress = strings.Split((strings.Split(publicAddress, "//")[1]), ":")[0]
}

func startProxy(address string, originUrl *url.URL, sni string) {
	// hack to fix ngrok not reusing established connections (a guess)
	f := func(conn net.Conn, connState http.ConnState) {
		if connState == http.StateIdle {
			err := conn.Close()
			if err != nil {
				log.Printf("ConnState callback failed to close idle connection: %v\n", err)
			}
		}

	}
	server = &http.Server{
		Addr:      address,
		ConnState: f,
	}

	// http.HandleFunc("/", handleProxy)
	// proxy = httputil.NewSingleHostReverseProxy(originUrl)
	rewrite := func(pr *httputil.ProxyRequest) {
		pr.SetURL(originUrl)
		// test SNI add host
		pr.Out.Host = sni //"gw.oktamanor.net"
	}
	proxy = &httputil.ReverseProxy{Rewrite: rewrite}
	proxy.ModifyResponse = modifyResponseImpl
	proxy.Transport = http.DefaultTransport
	proxy.Transport.(*http.Transport).TLSClientConfig = &tls.Config{
		// Set InsecureSkipVerify to skip the default validation we are
		// replacing. This will not disable VerifyConnection.
		InsecureSkipVerify: true,
		VerifyConnection: func(cs tls.ConnectionState) error {
			opts := x509.VerifyOptions{
				DNSName:       cs.ServerName,
				Intermediates: x509.NewCertPool(),
			}
			for _, cert := range cs.PeerCertificates[1:] {
				opts.Intermediates.AddCert(cert)
			}
			_, err := cs.PeerCertificates[0].Verify(opts)
			//return err
			log.Printf("Certificate Verifiaction error: %s\n", err)
			return nil
		},
		// SNI Support
		//ServerName:/*"gw.oktamanor.net", //*/ originUrl.Host,
		ServerName: sni, //"gw.oktamanor.net",
	}
	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		log.Printf("proxy.ErrorHandler: %+v\n", err)
	}

	go func() {
		log.Printf("Starting Proxy on: %s, origin set to: %s, sni set to: %s\n", address, originUrl.String(), sni)
		if err := server.ListenAndServe(); err != nil {
			// if err := server.ListenAndServeTLS("/Users/erikmanor/Certs/erikdevelopernot.com/origin/cert+chain.pem", "/Users/erikmanor/Certs/erikdevelopernot.com/origin/pkey.pem"); err != nil {
			log.Printf("Proxy server down: %s\n", err)
		}
	}()
}

// use ReverseProxy hook to get access to the Origins response body
func modifyResponseImpl(res *http.Response) error {
	// get request id from http header and set status
	id := res.Request.Header.Get(http_header_scim_id)
	messageLogs.AddResponseStatus(id, res.StatusCode)

	//TEST
	// if res.Request.Method == http.MethodPost || res.Request.Method == http.MethodGet {
	// 	manualFilter := filters.ManualFilter{}
	// 	manualFilter.PostResponse(res.Header, res.Cookies(), nil, res.Request.URL.RequestURI())
	// 	res.Header.Add("Set-Cookie", "MyCookie=4B89AC; Path=/; Secure; HttpOnly")
	// }
	//END Test

	// process response from origin
	//header
	h := make(map[string][]string)
	sb := strings.Builder{}
	for k, v := range res.Header {
		sb.WriteString(fmt.Sprintf("%v : %v\n", k, v))
		h[k] = v
	}

	header := sb.String()
	// fmt.Printf("header:\n%s\n", header)
	// body
	b, err := io.ReadAll(res.Body)
	// fmt.Printf("\n>>%s<<\n", string(b))
	res.Body.Close()
	if err != nil {
		fmt.Printf("Error reading Proxy Response Data: %v\n", err)
		return nil
	}

	var bytesBody []byte
	encoding := res.Header.Get("Content-Encoding")
	if len(b) > 1 {
		buf := bytes.Buffer{}
		if err := json.Indent(&buf, b, "", "   "); err != nil {
			// check content encoding
			var compressionReader io.Reader
			reader := bytes.NewReader(b)
			if strings.ToLower(encoding) == "gzip" {
				compressionReader, err = gzip.NewReader(reader)
				if err != nil {
					fmt.Printf("Error Getting gzip Reader: %s\n", err)
				}
			} else if encoding == "br" {
				compressionReader = brotli.NewReader(reader)
			} else if encoding == "zstd" {
				compressionReader, err = zstd.NewReader(reader)
				if err != nil {
					fmt.Printf("Error Getting zstd Reader: %s\n", err)
				}
			}

			if compressionReader != nil {
				bytesBody, _ = io.ReadAll(compressionReader)
			}
		} else {
			bytesBody = buf.Bytes()
		}
	} else {
		bytesBody = nil
	}

	messageLogBody := bytesBody
	if filterResponseMessage(res.Request) {
		// h, b = (*config.ReqFilter).(*filters.ManualFilter).FilterRequest(h, bytesBody, fmt.Sprintf("%s Response For: %s", res.Request.Method, res.Request.RequestURI), "json")
		h, b = (*config.ReqFilter).FilterRequest(h, bytesBody, fmt.Sprintf("%s Response For: %s", res.Request.Method, res.Request.RequestURI), "json")
		res.Header = h
		sb.Reset()
		for k, v := range h {
			sb.WriteString(fmt.Sprintf("%v : %v\n", k, v))
			h[k] = v
		}
		header = sb.String()
		// log.Printf("Filtered Message: \n%s\n", string(b))
		messageLogBody = make([]byte, len(b))
		copy(messageLogBody, b)

		// check content encoding to see if we need to re-encode
		var compressionBuf bytes.Buffer
		if strings.ToLower(encoding) == "gzip" {
			gzipWriter := gzip.NewWriter(&compressionBuf)
			_, err = gzipWriter.Write(b)
			if err != nil {
				log.Printf("Error, gzipWriter gzip content: %s\n", err)
			} else {
				err = gzipWriter.Close()
				if err != nil {
					fmt.Printf("Error, gzipWriter error closing: %s\n", err)
				}
				b = compressionBuf.Bytes()
			}
		} else if encoding == "br" {
			brotliWriter := brotli.NewWriter(&compressionBuf)
			_, err = brotliWriter.Write(b)
			if err != nil {
				log.Printf("Error, brotliWriter gzip content: %s\n", err)
			} else {
				err = brotliWriter.Close()
				if err != nil {
					fmt.Printf("Error, brotliWriter error closing: %s\n", err)
				}
				b = compressionBuf.Bytes()
			}
		} else if encoding == "zstd" {
			writer, err := zstd.NewWriter(&compressionBuf)
			if err != nil {
				log.Printf("Error creating %s Writer: %s\n", encoding, err)
			} else {
				err = writer.Close()
				if err != nil {
					fmt.Printf("Error closing Writer %s: %s\n", encoding, err)
				}
				b = compressionBuf.Bytes()
			}
		}

		res.Body = io.NopCloser(bytes.NewBuffer(b))
		// fmt.Printf("%+v\n", res.Body)
		cl := res.Header.Get("Content-Length")
		if cl != "" && cl != "0" {
			// res.ContentLength = int64(len(b))
			// setting res.ContentLength is not used for response sent back to client, need to set it in the header Map. Seems like a bug ??
			res.Header.Set("Content-Length", strconv.FormatInt(int64(len(b)), 10))
		}

	} else {
		res.Body = io.NopCloser(bytes.NewBuffer(b))
	}

	messageLogs.AddResponse(id, string(messageLogBody), proxy_msg, &header, h)
	return nil
}

// var i int = 1

// http handlers
func handleProxy(res http.ResponseWriter, req *http.Request) {
	// TODO - change based off of port binding - hack for now
	if !strings.Contains(req.Host, "localhost") &&
		// !strings.Contains(req.Host, "gw.oktamanor.net") &&
		!strings.Contains(req.Host, publicAddress) &&
		!strings.Contains(req.Host, "okta.com") &&
		!strings.Contains(req.Host, "oktapreview.com") {
		apps.HandleApprouting(res, req, strings.Split(req.Host, ".")[0])
		return
	}

	log.Printf("proxy: RequestURI=%s\n", req.RequestURI)
	// log.Printf("%v\n", req)
	if !config.Server.Proxy_messages {
		res.WriteHeader(int(http.StatusServiceUnavailable))
		return
	}

	now := time.Now()
	//Headers
	h := make(map[string][]string)
	var sb strings.Builder
	for k, v := range req.Header {
		sb.WriteString(fmt.Sprintf("%v : %v\n", k, v))
		h[k] = v
	}

	m := messageLogs.Message{
		TimeStamp:     now,
		Method:        req.Method,
		Url:           fmt.Sprintf("https://%s%s", req.Host, req.URL.RequestURI()),
		Headers:       sb.String(),
		ReqHeadersMap: h,
	}
	//Body
	if req.Method == http.MethodPost || req.Method == http.MethodPatch || req.Method == http.MethodPut {
		contentType := ""
		b, err := io.ReadAll(req.Body)
		req.Body.Close()
		if err != nil {
			fmt.Printf("Error reading Proxy Request Data: %v\n", err)
			return
		}

		if len(b) > 1 {
			// fmt.Printf("byte length: %v, Content-Length: %v\n", len(b), req.ContentLength)
			buf := bytes.Buffer{}
			if err := json.Indent(&buf, b, "", "   "); err != nil {
				log.Printf("handleProxy() - Error Formatting JSON: %s\n%s\n", err, string(b))
				m.RequestBody = string(b)
			} else {
				// TODO - might base this off of http header content-type
				contentType = "json"
				m.RequestBody = buf.String()
			}

			// Should this message be filtered
			if filterRequestMessage(req) {
				var newBytes []byte
				// h, newBytes = (*config.ReqFilter).(*filters.ManualFilter).FilterRequest(h, []byte(m.RequestBody), fmt.Sprintf("%s Request For: %s", req.Method, req.RequestURI), contentType)
				h, newBytes = (*config.ReqFilter).FilterRequest(h, []byte(m.RequestBody), fmt.Sprintf("%s Request For: %s", req.Method, req.RequestURI), contentType)
				req.Body = io.NopCloser(bytes.NewBuffer(newBytes))
				if req.ContentLength > 0 {
					req.ContentLength = int64(len(newBytes))
				}
				// res.Header.Add("Set-Cookie", "MyCookie=4B89AC; Path=/; Secure; HttpOnly")
			} else {
				req.Body = io.NopCloser(bytes.NewBuffer(b))
			}
		}
	} else if req.Method == http.MethodGet || req.Method == http.MethodOptions || req.Method == http.MethodDelete {
		if filterRequestMessage(req) {
			// h, _ = (*config.ReqFilter).(*filters.ManualFilter).FilterRequest(h, []byte{}, fmt.Sprintf("%s Request to: %s", req.Method, req.RequestURI), "")
			h, _ = (*config.ReqFilter).FilterRequest(h, []byte{}, fmt.Sprintf("%s Request to: %s", req.Method, req.RequestURI), "")
		}
	}

	req.Header = h

	// TEST Closing a connection - Leave for future tests.
	/*
		fmt.Printf("req.RequestURI = %s, strings.Contains(req.RequestURI, \"/oauth2/v1/token\") = %v\n", req.RequestURI, strings.Contains(req.RequestURI, "/oauth2/v1/token"))
		//fmt.Printf("%+v\n", m)
		if strings.Contains(req.RequestURI, "/oauth2/v1/token") {
			i = i + 1
			// if i%2 == 0 {
			if i > 2 {
				hj, ok := res.(http.Hijacker)
				if !ok {
					http.Error(res, "webserver doesn't support hijacking", http.StatusInternalServerError)
					return
				}
				conn, _, err := hj.Hijack()
				if err != nil {
					http.Error(res, err.Error(), http.StatusInternalServerError)
					return
				}
				fmt.Println(">>>> Closing /token Connection")
				if err := conn.Close(); err != nil {
					fmt.Printf(">>>> Closing /token Connection ERROR: %s\n", err)
				}
				return
			}
		}
	*/
	// END TEST -

	// add unique message id as http header too match response, use req memory address
	req.Header.Add(http_header_scim_id, fmt.Sprintf("%p", req))
	messageLogs.AddRequest(fmt.Sprintf("%p", req), m)

	// send to origin
	proxy.ServeHTTP(res, req)
}

func filterRequestMessage(req *http.Request) bool {
	if config.ProxyMessageFilter.FilterMessages {
		if filterURL, ok := config.ProxyMessageFilter.FilterURLs[strings.Split(req.RequestURI, "?")[0]]; ok {
			if filterURL.REQUEST && filterMethod(req.Method, filterURL) {
				return true
			}
		}
	}

	return false
}

func filterResponseMessage(req *http.Request) bool {
	if config.ProxyMessageFilter.FilterMessages {
		if filterURL, ok := config.ProxyMessageFilter.FilterURLs[strings.Split(req.RequestURI, "?")[0]]; ok {
			if filterURL.RESPONSE && filterMethod(req.Method, filterURL) {
				return true
			}
		}
	}

	return false
}

func filterMethod(method string, url utils.ProxyFilterURL) bool {
	switch method {
	case "GET":
		return url.GET
	case "POST":
		return url.POST
	case "PATCH":
		return url.PATCH
	case "PUT":
		return url.PUT
	case "DELETE":
		return url.DELETE
	case "OPTIONS":
		return url.OPTIONS
	}
	return false
}

func handleToggleProxyLogging(res http.ResponseWriter, req *http.Request) {
	state, err := strconv.ParseBool(req.URL.Query().Get("enabled"))
	if err != nil {
		log.Printf("handleToggleProxyLogging.ParseBool() failed: %v\n", err)
		res.WriteHeader(400)
		res.Write(nil)
		return
	}

	log.Printf("Setting Proxy Logging to %v\n", state)
	if state {
		u, err := url.Parse(req.URL.Query().Get("url"))
		if err != nil || u.String() == "" {
			log.Printf("handleToggleProxyLogging.url.Parse() failed: %v\n", err)
			res.WriteHeader(400)
			return
		}
		if !u.IsAbs() {
			log.Printf("Invalid Origin URL Specified: %s\n", req.URL.Query().Get("url"))
			res.WriteHeader(http.StatusBadRequest)
			res.Write([]byte(`{"error":"invalid origin url"}`))
			return
		}

		port, err := strconv.ParseInt(req.URL.Query().Get("port"), 10, 64)
		if err != nil {
			log.Printf("Invalid Proxy Port Specified: %s\n", req.URL.Query().Get("port"))
			res.WriteHeader(http.StatusBadRequest)
			res.Write([]byte(`{"error":"invalid port number"}`))
			return
		}

		sni := req.URL.Query().Get("sni")
		if sni == "" {
			log.Printf("handleToggleProxyLogging.url.Parse(sni) failed: %v\n", err)
			log.Println("Setting SNI to Origin")
			sni = u.Host
		}

		address := fmt.Sprintf(":%v", port)
		startProxy(address, u, sni)
		config.Server.Proxy_address = address
		config.Server.Proxy_port = int(port)
		config.Server.Proxy_origin = u.String()
		config.Server.Proxy_sni = sni
	} else {
		log.Println("Shutting down proxy")
		err := server.Close()
		if err != nil {
			log.Printf("Error shutting down Proxy: %s\n", err)
		}
	}

	config.Server.Proxy_messages = state
	res.WriteHeader(200)
	res.Write(nil)
}

func handleToggleProxyFilterIps(res http.ResponseWriter, req *http.Request) {
	state, err := strconv.ParseBool(req.URL.Query().Get("enabled"))
	if err != nil {
		log.Printf("handleToggleProxyFilterIps.ParseBool() failed: %v\n", err)
		res.WriteHeader(400)
		res.Write(nil)
		return
	}

	log.Printf("Setting Proxy Filter IPs to %v\n", state)
	config.Server.ProxyFilterIps = state
	res.WriteHeader(200)
	res.Write(nil)
}

// func handleProxyFilterConfig(res http.ResponseWriter, req *http.Request) {

// }

func handleProxyMessages(res http.ResponseWriter, req *http.Request) {
	fmt.Println("Returning Proxy Messages")
	getMessages(res, req, "proxy.gohtml")
}
