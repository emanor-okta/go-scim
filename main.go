package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	v2 "github.com/emanor-okta/go-scim/types/v2"
	"github.com/go-redis/redis/v8"

	"github.com/emanor-okta/go-scim/filters"
	"github.com/emanor-okta/go-scim/server"
	"github.com/emanor-okta/go-scim/types/v2/extension"
	"github.com/emanor-okta/go-scim/utils"
)

func main() {
	fmt.Println("Starting")
	config := utils.LoadConfig("config.yaml")
	if err := utils.InitializeRedis(config); err != nil {
		log.Fatalf("Error initializing Redis: %v\n", err)
	}

	user := v2.EnterpriseUser{}
	user.Id = "sjhfe23u38hwfw"
	user.Schemas = []string{v2.USER_SCHEMA, extension.ENTERPRISE_USER_SCHEMA}
	user.Meta = v2.Meta{
		ResourceType: "User",
		Created:      "2010-01-23T04:56:22Z",
		LastModified: "2010-01-23T04:56:22Z",
		Version:      "sdfsifdnsjdnfk",
		Location:     "https:g.com/user/1",
	}
	user.ENTERPRISE_USER_SCHEMA.CostCenter = "9876"

	j, err := json.Marshal(user) //Indent(user, "", "  ")
	if err != nil {
		log.Fatalf("Marshal error: %v\n", err)
	}
	fmt.Println(string(j))
	// utils.Test(j)

	var u v2.User
	if err = json.Unmarshal(j, &u); err != nil {
		log.Fatalf("Unmarshal Error: %v\n", err)
	}

	// fmt.Printf("id: %s\nMeta: %v\nCostCenter: %v\n", u.Id, u.Meta, u.ENTERPRISE_USER_SCHEMA.CostCenter)
	fmt.Printf("id: %s\nMeta: %v\n", u.Id, u.Meta)

	var m map[string]interface{}
	if err = json.Unmarshal(j, &m); err != nil {
		log.Fatalf("Unmarshal to Map error: %v\n", err)
	}
	fmt.Printf("Map:\n%v\n", m)
	meta := m["meta"].(map[string]interface{})
	fmt.Println(meta)
	fmt.Printf("Created: %v, Modified: %v, Version: %v\n", meta["created"].(string), meta["lastModified"], meta["version"])

	// testRedis(string(j), j)

	/*
	 * Set Filters
	 */
	server.TestFilter = filters.TestReqFilter{}

	server.StartServer(config)
}

func testRedis(str string, bits []byte) {
	var ctx = context.Background()

	rdb := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "", // no password set
		DB:       0,  // use default DB
	})

	err := rdb.Set(ctx, "str", str, 0).Err()
	if err != nil {
		panic(err)
	}

	val, err := rdb.Get(ctx, "str").Result()
	if err == redis.Nil {
		fmt.Println("str does not exist")
	} else if err != nil {
		panic(err)
	}
	fmt.Println("str", val)

	err = rdb.Set(ctx, "bits", bits, 0).Err()
	if err != nil {
		panic(err)
	}

	val2, err := rdb.Get(ctx, "bits").Result()
	if err == redis.Nil {
		fmt.Println("bits does not exist")
	} else if err != nil {
		panic(err)
	}
	fmt.Println("bits", val2)
}
