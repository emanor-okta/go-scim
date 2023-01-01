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
	// fmt.Printf(">>>>>>>Using LoggerResponseWriter Header<<<<<<< %p\n", &lrw)
	// fmt.Printf("%+v\n", lrw.RW.Header())
	return lrw.RW.Header()
}

func (lrw LoggerResponseWriter) Write(b []byte) (int, error) {
	// fmt.Printf(">>>>>>>Using LoggerResponseWriter<<<<<<< %p\n", lrw.R)
	// test := fmt.Sprintf("%v", &lrw.R)
	// fmt.Printf("test=%v, type=%T\n", test, test)
	if b != nil && len(b) > 1 {
		buf := bytes.Buffer{}
		if err := json.Indent(&buf, b, "", "   "); err != nil {
			fmt.Printf("Error Formatting JSON: %s\n", err)
		} else {
			// fmt.Printf("%s\n", buf.String())
			messageLogs.AddResponse(fmt.Sprintf("%p", lrw.R), buf.String())
		}
	}
	return lrw.RW.Write(b)
}

func (lrw LoggerResponseWriter) WriteHeader(statusCode int) {
	// fmt.Println(">>>>>>>Using LoggerResponseWriter WriteHeader<<<<<<<")
	// fmt.Printf("%v\n", statusCode)
	lrw.RW.WriteHeader(statusCode)
}
