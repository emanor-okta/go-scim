package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	messageLogs "github.com/emanor-okta/go-scim/server/log"
	"github.com/emanor-okta/go-scim/utils"
)

//type Middleware func(http.HandlerFunc) http.HandlerFunc

const (
	_logPrefix = "server.middleware."
)

// func addMiddleware(h http.HandlerFunc, m ...types.Middleware) http.HandlerFunc {
// 	if len(m) < 1 {
// 		return h
// 	}

// 	middlewares := h
// 	for _, v := range m {
// 		middlewares = v(middlewares)
// 	}

// 	return middlewares
// }

func getHeadersMiddleware(h http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var sb strings.Builder
		sb.WriteString("HTTP Headers\n")
		for k, v := range r.Header {
			sb.WriteString(fmt.Sprintf("%v : %v\n", k, v))
		}

		fmt.Printf("\n%v\n", sb.String())
		h.ServeHTTP(w, r)
	})
}

func getBodyMiddleware(h http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost || r.Method == http.MethodPatch || r.Method == http.MethodPut {
			b, err := io.ReadAll(r.Body)
			r.Body.Close()
			r.Body = io.NopCloser(bytes.NewBuffer(b))
			if err != nil {
				fmt.Printf("Error reading Json Data: %v\n", err)
				defer h.ServeHTTP(w, r)
				return
			}

			var sb strings.Builder
			sb.WriteString("> Request Body\n")
			sb.WriteString(string(b))
			fmt.Printf("\n%v\n", sb.String())
		}
		h.ServeHTTP(w, r)
	})
}

func filterIpMiddleware(h http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// if config.Server.FilterIPs {
		// Need to find correct way to do this, for now check X-Forwarded-For, followed by ReoteAddr
		addr := utils.GetRemoteAddress(r)
		// addr := r.Header.Get("X-Forwarded-For")
		// if addr == "" {
		// 	addr = r.RemoteAddr
		// 	if addr != "" {
		// 		index := strings.LastIndex(addr, ":")
		// 		if index > -1 {
		// 			addr = addr[0:index]
		// 		}
		// 	}
		// }
		fmt.Printf("Address: %s\n", addr)
		_, ok := config.Server.Allowed_ips[addr]
		if ok {
			h.ServeHTTP(w, r)
		} else {
			log.Printf("%sfilterIpMiddleware: Denying Request from %s\n", _logPrefix, addr)
			w.WriteHeader(http.StatusForbidden)
		}
	})
}

func logMessagesMiddleware(h http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if config.Server.Log_messages || config.Server.Proxy_messages {
			now := time.Now()
			//Headers
			hMap := make(map[string][]string)
			var sb strings.Builder
			// sb.WriteString("HTTP Headers\n")
			for k, v := range r.Header {
				sb.WriteString(fmt.Sprintf("%v : %v\n", k, v))
				hMap[k] = v
			}
			m := messageLogs.Message{
				TimeStamp:     now,
				Method:        r.Method,
				Url:           r.URL.RequestURI(),
				Headers:       sb.String(),
				ReqHeadersMap: hMap,
			}

			//Body
			if r.Method == http.MethodPost || r.Method == http.MethodPatch || r.Method == http.MethodPut {
				b, err := io.ReadAll(r.Body)
				r.Body.Close()
				r.Body = io.NopCloser(bytes.NewBuffer(b))
				if err != nil {
					fmt.Printf("Error reading Json Data: %v\n", err)
					defer h.ServeHTTP(w, r)
					return
				}

				if b != nil && len(b) > 1 {
					buf := bytes.Buffer{}
					if err := json.Indent(&buf, b, "", "   "); err != nil {
						log.Printf("getHeadersMiddleware() - Error Formatting JSON: %s\n", err)
					} else {
						m.RequestBody = buf.String()
					}
				}
			}
			messageLogs.AddRequest(fmt.Sprintf("%p", r), m)
		}
		h.ServeHTTP(w, r)
	})
}

func logMessageResponseSudoMiddleware(h http.HandlerFunc) http.HandlerFunc {
	// not really middle-ware instead use a custom ResponseWriter to log return messages
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		lrw := LoggerResponseWriter{RW: w, R: r}
		h.ServeHTTP(lrw, r)
	})
}
