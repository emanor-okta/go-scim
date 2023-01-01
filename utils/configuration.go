package utils

import (
	"io/ioutil"
	"log"

	"gopkg.in/yaml.v3"
)

type Configuration struct {
	ReqFilter *ReqFilter
	Redis     struct {
		Address  string
		Password string
		Db       int
	}
	Server struct {
		Address       string
		Web_address   string
		Web_console   bool
		Debug_headers bool
		Debug_body    bool
		Debug_query   bool
		Log_messages  bool
	}
	Scim struct {
		Enable_groups bool
	}
	WebMessageFilter struct {
		UserPostRequest      bool
		UserPostResponse     bool
		UserGetResponse      bool
		UserIdPutRequest     bool
		UserIdPutResponse    bool
		UserIdPatchRequest   bool
		UserIdPatchResponse  bool
		UserIdGetResponse    bool
		GroupsGetResponse    bool
		GroupsPostRequest    bool
		GroupsPostResponse   bool
		GroupsIdGetResponse  bool
		GroupsIdPutRequest   bool
		GroupsIdPutResponse  bool
		GroupsIdPatchRequest bool
	}
}

func LoadConfig(c string) *Configuration {
	var config Configuration
	buf, err := ioutil.ReadFile(c)
	if err != nil {
		log.Fatalf("No Configuration file exists: %v\n", err)
	}

	err = yaml.Unmarshal(buf, &config)
	if err != nil {
		log.Fatal(err)
	}
	return &config
}
