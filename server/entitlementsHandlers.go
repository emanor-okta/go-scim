package server

import (
	"log"
	"net/http"
	"strings"

	"github.com/emanor-okta/go-scim/utils"
)

func handleResourceTypes(res http.ResponseWriter, req *http.Request) {
	res.Header().Add("content-type", content_type)
	loadEntitlements()
	if req.Method == http.MethodPut {
		body, err := getBody(req)
		if err != nil {
			log.Printf("handleResourceTypes error: %s\n", err)
			res.WriteHeader(http.StatusInternalServerError)
			return
		}
		config.Entitlements.ResourceTypes = body
		err = utils.SetResourceTypes(string(body))
		if err != nil {
			log.Printf("handleResourceTypes error: %s\n", err)
			res.WriteHeader(http.StatusInternalServerError)
			return
		}
	}

	_, err := res.Write(config.Entitlements.ResourceTypes)
	if err != nil {
		log.Printf("handleResourceTypes error: %s\n", err)
	}
	// utils.SetResourceTypes(string(config.Entitlements.ResourceTypes))
}

func handleResourceType(res http.ResponseWriter, req *http.Request) {
	// log.Printf("ResourceType Request %s\n", req.RequestURI)
	res.Header().Add("content-type", content_type)
	loadEntitlements()
	paths := strings.Split(strings.Split(req.RequestURI, "?")[0], "/")
	resource := strings.ToLower(paths[len(paths)-1])
	if req.Method == http.MethodPut {
		// UPDATE existing resource or add it
		body, err := getBody(req)
		if err != nil {
			log.Printf("handleResourceType error: %s\n", err)
			res.WriteHeader(http.StatusInternalServerError)
			return
		}
		config.Entitlements.Resources[resource] = body
		err = utils.SetResource(resource, string(body))
		if err != nil {
			log.Printf("handleResourceType error: %s\n", err)
			res.WriteHeader(http.StatusInternalServerError)
			return
		}
	}

	data, ok := config.Entitlements.Resources[resource]
	var err error
	if ok {
		if req.Method == http.MethodDelete {
			// DELETE existing resource
			delete(config.Entitlements.Resources, resource)
			err = utils.DeleteResource(resource)
		} else {
			_, err = res.Write(data)
		}
	} else {
		_, err = res.Write([]byte("{}"))
	}
	if err != nil {
		log.Printf("handleResourceType for %s, error: %s\n", resource, err)
	}
}

func handleSchemas(res http.ResponseWriter, req *http.Request) {
	res.Header().Add("content-type", content_type)
	loadEntitlements()
	if req.Method == http.MethodPut {
		body, err := getBody(req)
		if err != nil {
			log.Printf("handleSchemas error: %s\n", err)
			res.WriteHeader(http.StatusInternalServerError)
			return
		}
		config.Entitlements.Schemas = body
		err = utils.SetSchema(string(body))
		if err != nil {
			log.Printf("handleSchemas error: %s\n", err)
			res.WriteHeader(http.StatusInternalServerError)
			return
		}
	}

	_, err := res.Write(config.Entitlements.Schemas)
	if err != nil {
		log.Printf("handleSchemas error: %s\n", err)
	}
}

func loadEntitlements() {
	if config.Entitlements.Schemas == nil || len(config.Entitlements.Schemas) < 1 {
		utils.LoadScimEntitlements()
	}
}
