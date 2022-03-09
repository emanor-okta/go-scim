package server

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"strings"
)

type Middleware func(http.HandlerFunc) http.HandlerFunc

func addMiddleware(h http.HandlerFunc, m ...Middleware) http.HandlerFunc {
	if len(m) < 1 {
		return h
	}

	middlewares := h
	for _, v := range m {
		middlewares = v(middlewares)
	}

	return middlewares
}

func getHeadersMiddleware(h http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var sb strings.Builder
		sb.WriteString("> Request Headers\n")
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
			r.Body = ioutil.NopCloser(bytes.NewBuffer(b))
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
