package apps

import (
	"log"
	"net/http"
	"path"
)

func HandleApprouting(res http.ResponseWriter, req *http.Request, app string) {
	var fp string
	res.Header().Set("Content-type", "text/html; charset=utf-8")
	if app == "app1" {
		fp = path.Join("apps", "app1.html")
	} else {
		log.Printf("Error: app.HandleApprouting host not known: %s\n", app)
		return
	}

	http.ServeFile(res, req, fp)
}
