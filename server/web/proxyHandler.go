package web

import (
	"bytes"
	"compress/gzip"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/emanor-okta/go-scim/apps"
	"github.com/emanor-okta/go-scim/filters"
	messageLogs "github.com/emanor-okta/go-scim/server/log"
	// br "github.com/google/brotli/go/cbrotli"
)

// br "github.com/google/brotli/go/cbrotli"
// const default_proxy_port = 8084
const http_header_scim_id = "X-Go-Scim-Id"
const proxy_msg = "proxy.gohtml"

var server *http.Server
var proxy *httputil.ReverseProxy

func init() {
	// with default Mux can only add a specific route once so do in init() instead of startProxy()
	http.HandleFunc("/", handleProxy)
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
	if res.Request.Method == http.MethodPost {
		manualFilter := filters.ManualFilter{}
		manualFilter.PostResponse(res.Header, res.Cookies(), nil, res.Request.URL.RequestURI())
		res.Header.Add("Set-Cookie", "MyCookie=4B89AC; Path=/; Secure; HttpOnly")
	}
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
	res.Body.Close()
	res.Body = ioutil.NopCloser(bytes.NewBuffer(b))
	if err != nil {
		fmt.Printf("Error reading Proxy Response Data: %v\n", err)
		return nil
	}
	if len(b) > 1 {
		// fmt.Printf("Body:\n%s\n", string(b))
		buf := bytes.Buffer{}
		if err := json.Indent(&buf, b, "", "   "); err != nil {
			// check content encoding
			var compressionReader io.Reader
			encoding := res.Header.Get("Content-Encoding")
			reader := bytes.NewReader(b)
			if encoding == "gzip" {
				compressionReader, err = gzip.NewReader(reader)
				if err != nil {
					fmt.Printf("Error Reading gzip content: %s\n", err)
				}
				// else {
				// 	b, _ = ioutil.ReadAll(compressionReader)
				// }
			} else if encoding == "br" {
				// compressionReader = br.NewReader(reader)
			}
			if compressionReader != nil {
				b, _ = ioutil.ReadAll(compressionReader)
				//compressionReader.Close()
			}
			messageLogs.AddResponse(id, string(b), proxy_msg, &header, h)
		} else {
			messageLogs.AddResponse(id, buf.String(), proxy_msg, &header, h)
		}
	} else {
		messageLogs.AddResponse(id, "", proxy_msg, &header, h)
	}

	return nil
}

// var i int = 0

// http handlers
func handleProxy(res http.ResponseWriter, req *http.Request) {
	// TODO - change based off of port binding - hack for now
	if !strings.Contains(req.Host, "gw.oktamanor.net") {
		apps.HandleApprouting(res, req, strings.Split(req.Host, ".")[0])
		return
	}

	log.Printf("proxy: RequestURI=%s\n", req.RequestURI)
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
		b, err := io.ReadAll(req.Body)
		req.Body.Close()
		req.Body = ioutil.NopCloser(bytes.NewBuffer(b))
		if err != nil {
			fmt.Printf("Error reading Proxy Request Data: %v\n", err)
			return
		}

		if len(b) > 1 {
			buf := bytes.Buffer{}
			if err := json.Indent(&buf, b, "", "   "); err != nil {
				log.Printf("handleProxy() - Error Formatting JSON: %s\n", err)
				m.RequestBody = string(b)
			} else {
				m.RequestBody = buf.String()
			}
		}
	}

	// TEST - REMOVE
	/*
		fmt.Printf("%+v\n", m)
		if strings.Contains(req.URL.RequestURI(), "/token") {
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
	// END TEST - REMOVE

	// add unique message id as http header too match response, use req memory address
	req.Header.Add(http_header_scim_id, fmt.Sprintf("%p", req))
	messageLogs.AddRequest(fmt.Sprintf("%p", req), m)

	// send to origin
	proxy.ServeHTTP(res, req)
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

func handleProxyMessages(res http.ResponseWriter, req *http.Request) {
	fmt.Println("Returning Proxy Messages")
	getMessages(res, req, "proxy.gohtml")
}
