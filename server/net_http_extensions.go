package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"

	messageLogs "github.com/emanor-okta/go-scim/server/log"
)

const (
	scim_msg = "messages.gohtml"
)

type LoggerResponseWriter struct {
	RW http.ResponseWriter
	R  *http.Request
}

func (lrw LoggerResponseWriter) Header() http.Header {
	return lrw.RW.Header()
}

func (lrw LoggerResponseWriter) Write(b []byte) (int, error) {
	if config.Server.Log_messages || config.Server.Proxy_messages {
		//header
		h := make(map[string][]string)
		var header string
		if config.Server.Proxy_messages {
			var sb strings.Builder
			for k, v := range lrw.RW.Header() {
				sb.WriteString(fmt.Sprintf("%v : %v\n", k, v))
				h[k] = v
			}
			header = sb.String()
		}

		// body
		if len(b) > 1 {
			buf := bytes.Buffer{}
			if err := json.Indent(&buf, b, "", "   "); err != nil {
				log.Printf("server.net_http_extensions.Write: Error Formatting JSON: %s\n", err)
				fmt.Printf("%s\n", string(b))
			} else {
				// fmt.Printf("%s\n", buf.String())
				messageLogs.AddResponse(fmt.Sprintf("%p", lrw.R), buf.String(), scim_msg, &header, h)
			}
		} else {
			messageLogs.AddResponse(fmt.Sprintf("%p", lrw.R), "", scim_msg, &header, h)
		}
	}

	return lrw.RW.Write(b)
}

func (lrw LoggerResponseWriter) WriteHeader(statusCode int) {
	if config.Server.Log_messages {
		messageLogs.AddResponseStatus(fmt.Sprintf("%p", lrw.R), statusCode)
	}

	lrw.RW.WriteHeader(statusCode)
}
