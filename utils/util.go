package utils

import (
	"github.com/google/uuid"
	//"github.com/chromedp/cdproto/har"
)

func GenerateUUID() string {
	return uuid.NewString()
}

func ConvertHeaderMap(m interface{}) map[string][]string {
	r := map[string][]string{}
	for k, v := range m.(map[string]interface{}) {
		r[k] = []string{}
		for _, v2 := range v.([]interface{}) {
			r[k] = append(r[k], v2.(string))
		}
	}
	return r
}
