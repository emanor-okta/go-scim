package utils

import (
	"io/ioutil"
	"log"

	"gopkg.in/yaml.v3"
)

type Configuration struct {
	Redis struct {
		Address  string
		Password string
		Db       int
	}
	Server struct {
		Address       string
		Debug_headers bool
		Debug_body    bool
	}
	Scim struct {
		Enable_groups bool
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
