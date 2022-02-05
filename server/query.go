package server

import (
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

const (
	filter     = "filter"
	startIndex = "startIndex"
	count      = "count"
)

type _filter struct {
	userName    string
	displayName string
}

type query struct {
	filter     _filter
	startIndex int
	count      int
}

func didRedirect(res *http.ResponseWriter, req *http.Request) bool {
	fmt.Println(req.URL)
	if strings.Contains(req.URL.Path, `/v1/`) {
		redir := "https://c0f2-2601-644-8f00-d4e0-75c2-52f6-159f-3085.ngrok.io" + strings.Replace(req.URL.Path, `/v1`, `/v2`, 1)
		fmt.Println(redir)
		(*res).Header().Add("Location", redir)
		(*res).WriteHeader(http.StatusPermanentRedirect)
		(*res).Write(nil)
		return true
	}
	return false
}

func getQuery(params url.Values) query {
	var q query
	for k, v := range params {
		// fmt.Printf("k:%v, v:%v\n", k, v[0])
		switch k {
		case filter:
			f := strings.Fields(v[0])
			if len(f) > 2 {
				switch f[0] {
				case "userName":
					q.filter.userName = strings.ReplaceAll(f[2], "\"", "")
				case "displayName":
					q.filter.displayName = strings.ReplaceAll(f[2], "\"", "")
				default:
					log.Printf("Unknown Query Filter: %v\n", v)
				}
				continue
			}

			log.Printf("Unknown Query Filter: %v\n", v)
		case startIndex:
			i, err := strconv.Atoi(v[0])
			if err != nil {
				log.Printf("Error converting startIndex: %v, err: %v\n", v[0], err)
			}
			q.startIndex = i
		case count:
			i, err := strconv.Atoi(v[0])
			if err != nil {
				log.Printf("Error converting count: %v, err: %v\n", v[0], err)
			}
			q.count = i
		default:
			log.Printf("getQuery() received unknown Query Param k: %v, v: %v\n", k, v[0])
		}
	}
	return q
}

func debugQueryParams(q *query) {
	fmt.Printf("count: %v\n", q.count)
	fmt.Printf("filter userName: %v\n", q.filter.userName)
	fmt.Printf("filter displayName: %v\n", q.filter.displayName)
	fmt.Printf("startIndex: %v\n", q.startIndex)
}
