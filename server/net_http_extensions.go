package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	messageLogs "github.com/emanor-okta/go-scim/server/log"
)

type LoggerResponseWriter struct {
	RW http.ResponseWriter
	R  *http.Request
}

func (lrw LoggerResponseWriter) Header() http.Header {
	return lrw.RW.Header()
}

func (lrw LoggerResponseWriter) Write(b []byte) (int, error) {
	if config.Server.Log_messages {
		if b != nil && len(b) > 1 {
			buf := bytes.Buffer{}
			if err := json.Indent(&buf, b, "", "   "); err != nil {
				fmt.Printf("Error Formatting JSON: %s\n", err)
			} else {
				// fmt.Printf("%s\n", buf.String())
				messageLogs.AddResponse(fmt.Sprintf("%p", lrw.R), buf.String())
			}
		} else {
			messageLogs.AddResponse(fmt.Sprintf("%p", lrw.R), "")
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
