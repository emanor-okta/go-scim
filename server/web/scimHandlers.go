package web

// Currently only entitlements

import (
	"fmt"
	"net/http"

	"github.com/emanor-okta/go-scim/utils"
)

func handleEntitlements(res http.ResponseWriter, req *http.Request) {
	// MyAddress := utils.GetRemoteAddress(req)
	// Payload := struct {
	// 	MyAddress  string
	// 	RestoreUrl string
	// }{MyAddress: MyAddress, RestoreUrl: req.URL.Query().Get("restore-url")}
	if config.Entitlements.ResourceTypes == nil {
		utils.LoadScimEntitlements()
		fmt.Printf("%+v\n", config.Entitlements)
	}

	names := []string{}
	for key := range config.Entitlements.Resources {
		names = append(names, key)
	}

	tpl.ExecuteTemplate(res, "entitlements.gohtml", struct {
		Services      utils.Services
		ResourceNames []string
	}{
		Services:      config.Services,
		ResourceNames: names,
	})
}
