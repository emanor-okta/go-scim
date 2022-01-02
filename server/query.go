package server

import (
	"fmt"
	"log"
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
	userName string
}

type query struct {
	filter     _filter
	startIndex int
	count      int
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
	fmt.Printf("filter: %v\n", q.filter.userName)
	fmt.Printf("startIndex: %v\n", q.startIndex)
}
