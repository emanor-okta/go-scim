package filters

import "fmt"

type TestReqFilter struct {
	// json string
}

func (trf TestReqFilter) Request(json string) string {
	fmt.Println("In Test Request Filter")
	return json
}
