package server

import (
	"encoding/json"
	// "io"
	"log"
	"net/http"
)

func handleSSFReq(res http.ResponseWriter, req *http.Request) {
	log.Printf("Received SSF Request:\n%+v\n", req)

	if req.Method == http.MethodPost {
		// POST
		b, err := getBody(req)
		if err != nil {
			log.Printf("Error getting POST body: %v\n", err)
			// res.WriteHeader(http.StatusBadRequest)
			// res.Write([]byte(err.Error()))
			// return
		}

		var m map[string]interface{}
		if err := json.Unmarshal(b, &m); err != nil {
			log.Printf("Error decoding Json Data: %v\n", err)
			log.Printf("%+v\n", b)
			log.Printf("%+v\n", string(b))
			// res.WriteHeader(http.StatusBadRequest)
			// res.Write([]byte(err.Error()))
			// return
		}
	}

	res.WriteHeader(http.StatusAccepted)
}
