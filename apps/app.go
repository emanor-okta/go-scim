package apps

import (
	"log"
	"net/http"
	"path"
)

func HandleApprouting(res http.ResponseWriter, req *http.Request, app string) {
	var fp string
	res.Header().Set("Content-type", "text/html; charset=utf-8")
	res.Header().Set("Access-Control-Allow-Origin", "*")
	res.Header().Set("Access-Control-Expose-Headers", "Location")
	if app == "app1" {
		//fp = path.Join("apps", "app1.html")
		http.Redirect(res, req, "https://gw.oktamanor.net/oauth2/default/v1/authorize?client_id=0oa2cpl777xczKzL21d7&response_type=code&response_mode=query&scope=openid profile email&redirect_uri=http://localhost:8080/login/callback&state=foreverInTheSameState&nonce=85", http.StatusTemporaryRedirect)
		return
	} else if app == "app2" {
		fp = path.Join("apps", "app2.html")
	} else {
		log.Printf("Error: app.HandleApprouting host not known: %s\n", app)
		return
	}

	http.ServeFile(res, req, fp)
}
