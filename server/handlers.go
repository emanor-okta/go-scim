package server

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"

	v2 "github.com/emanor-okta/go-scim/types/v2"
	"github.com/emanor-okta/go-scim/utils"
)

const (
	content_type = "application/scim+json;charset=UTF-8"
)

var TestFilter ScimRequest

/*
 * SCIM Handlers
 */

// {GET/POST} /scim/v2/Users
func handleUsers(res http.ResponseWriter, req *http.Request) {
	res.Header().Add("content-type", content_type)

	if req.Method == http.MethodGet {
		// GET
		q := getQuery(req.URL.Query())
		debugQueryParams(&q)
		if q.filter.userName != "" {
			// ?filter=username eq <username>
			u, err := utils.GetUserByFilter(q.filter.userName)
			if err != nil {
				handleEmptyListReturn(&res, err)
				return
			}
			res.WriteHeader(http.StatusOK)
			res.Write([]byte(u))
		} else {
			// ?startIndex=<?>&count=<?>
			docs, err := utils.GetUsers(q.startIndex, q.count)
			if err != nil {
				// res.WriteHeader(http.StatusInternalServerError)
				// res.Write(nil)
				fmt.Printf("\n\n%v\n\n", err)
				handleEmptyListReturn(&res, err)
				return
			}

			lr := buildListResponse(docs)
			j, err := json.Marshal(&lr)
			if err != nil {
				log.Fatalf("Error Marshalling ListResponse: %v\n", err)
			}
			fmt.Println(string(j))
			res.WriteHeader(http.StatusOK)
			res.Write(j)
		}
	} else if req.Method == http.MethodPost {
		// POST
		b, err := getBody(req)
		if err != nil {
			res.WriteHeader(http.StatusOK)
			res.Write(nil)
			return
		}

		var m map[string]interface{}
		if err := json.Unmarshal(b, &m); err != nil {
			log.Printf("Error decoding Json Data: %v\n", err)
			res.WriteHeader(http.StatusOK)
			res.Write(nil)
			return
		}

		uuid := utils.GenerateUUID()
		meta := v2.Meta{ResourceType: "User", Location: "/scim/v2/Users/" + uuid}
		m["meta"] = meta
		m["id"] = uuid
		doc, _ := json.Marshal(m)

		// printBody(doc)
		// fmt.Println(m)
		if TestFilter != nil {
			TestFilter.Request(string(doc))
		}

		if _, err = utils.GetUserByFilter(m["userName"].(string)); err == nil {
			// user already exist return 409
			handleErrorResponse(&res, USER_ALREADY_EXISTS, http.StatusConflict)
			return
		}

		if err = utils.AddUser(doc, m["userName"].(string), uuid); err != nil {
			res.WriteHeader(http.StatusOK)
			res.Write(nil)
			return
		}
		res.WriteHeader(http.StatusCreated)
		if _, err = res.Write(doc); err != nil {
			log.Printf("Error returning AddUser call: %v\b", err)
		}
	} else {
		// NOT-SUPPORTED
		handleNotSupported(req, &res)
	}
}

// {GET/PUT/PATCH/DELETE} /scim/v2/Users/<id>
func handleUser(res http.ResponseWriter, req *http.Request) {
	res.Header().Add("content-type", content_type)

	parts := strings.Split(req.URL.Path[1:], "/")
	if len(parts) != 4 || parts[3] == "" {
		res.WriteHeader(http.StatusNotFound)
		res.Write(nil)
		fmt.Printf("Not Found: %v, %v\n", len(parts), parts)
		return
	}
	uuid := parts[3]

	if req.Method == http.MethodDelete {
		// DELETE (not used by Okta)
		doc, err := utils.GetDoc(uuid)
		if err != nil {
			handleErrorForKeyLookup(&res, err)
			return
		}
		var m map[string]interface{}
		json.Unmarshal([]byte(doc), &m)
		if err := utils.DelUser(uuid, m["userName"].(string)); err != nil {
			log.Fatalf("Error for DELETE /scim/v2/User/%v, err: %v\n\n", uuid, err)
		}

		res.WriteHeader(http.StatusOK)
		res.Write(nil)
	} else if req.Method == http.MethodGet {
		// GET
		doc, err := utils.GetDoc(uuid)
		if err != nil {
			handleErrorForKeyLookup(&res, err)
			return
		}
		res.WriteHeader(http.StatusOK)
		if _, err = res.Write([]byte(doc)); err != nil {
			log.Printf("Error replying for GET /scim/v2/User/%v, err: %v\n\n", uuid, err)
		}
	} else {
		b, err := getBody(req)
		if err != nil {
			res.WriteHeader(http.StatusInternalServerError)
			res.Write(nil)
			return
		}

		if req.Method == http.MethodPut {
			// PUT
			if err := utils.UpdateDoc(uuid, b); err != nil {
				handleErrorForKeyLookup(&res, err)
				return
			}
			res.WriteHeader(http.StatusOK)
			if _, err = res.Write(b); err != nil {
				log.Printf("Error replying for PUT /scim/v2/User/%v, err: %v\n\n", uuid, err)
			}
		} else if req.Method == http.MethodPatch {
			// PATCH
			var ops v2.PatchOp
			if err := json.Unmarshal(b, &ops); err != nil {
				log.Printf("Error Unmarshalling JSON for PATCH /scim/v2/User/%v, err: %v\n\n", uuid, err)
				res.WriteHeader(http.StatusInternalServerError)
				res.Write(nil)
				return
			}

			doc, err := utils.GetDoc(uuid)
			if err != nil {
				handleErrorForKeyLookup(&res, err)
				return
			}
			var m map[string]interface{}
			if err = json.Unmarshal([]byte(doc), &m); err != nil {
				log.Fatalf("Error Unmarshalling User for PATCH /scim/v2/User/%v, err: %v\n\n", uuid, err)
			}

			for _, v := range ops.Operations {
				if v.Value.Password != "" {
					m["password"] = v.Value.Password
				} else {
					m["active"] = v.Value.Active
				}
			}

			b, _ := json.Marshal(m)
			if err := utils.UpdateDoc(uuid, b); err != nil {
				handleErrorForKeyLookup(&res, err)
				return
			}
			res.WriteHeader(http.StatusOK)
			if _, err = res.Write(b); err != nil {
				log.Printf("Error replying for PATCH /scim/v2/User/%v, err: %v\n\n", uuid, err)
			}
		} else {
			// NOT-SUPPORTED
			handleNotSupported(req, &res)
		}
	}
}

/*
 * ADMIN Handlers
 */
