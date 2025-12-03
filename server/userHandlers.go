package server

import (
	"compress/gzip"
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

/*
 * SCIM Handlers
 */

// {GET/POST} /goscim/scim/v2/Users
func handleUsers(res http.ResponseWriter, req *http.Request) {
	// if didRedirect(&res, req) {
	// 	return
	// }
	res.Header().Add("content-type", content_type)
	path := "/goscim/scim/v2/Users"

	if req.Method == http.MethodGet {
		// GET
		var docs interface{}
		q := getQuery(req.URL.Query())
		if debugQuery {
			debugQueryParams(&q)
		}

		if q.filter.userName != "" {
			// TEST
			// if strings.Contains(q.filter.userName, "igor") {
			// 	// return unauthorized
			// 	res.WriteHeader(http.StatusUnauthorized)
			// 	return
			// }
			// END TEST
			// ?filter=username eq <username>
			path = fmt.Sprintf("%s?filter=username eq %s&startIndex=%v&count=%v", path, q.filter.userName, q.startIndex, q.count)
			user, err := utils.GetUserByFilter(q.filter.userName)
			if err != nil {
				handleEmptyListReturn(&res, err, &reqFilter, fmt.Sprintf("GET %s  -  Response", path))
				return
			}
			docs = []interface{}{}
			docs = append(docs.([]interface{}), user)
		} else {
			// ?startIndex=<?>&count=<?>
			path = fmt.Sprintf("%s?startIndex=%v&count=%v", path, q.startIndex, q.count)
			var err error
			docs, err = utils.GetUsersByRange(q.startIndex, q.count)
			if err != nil {
				handleEmptyListReturn(&res, err, &reqFilter, fmt.Sprintf("GET %s  -  Response", path))
				return
			}
		}

		users := embedUsersGroups(docs)

		// users = reqFilter.UserGetResponse(users) <-- change to use v2.ListResponse below

		lr := buildListResponse(users)
		reqFilter.UserGetResponse(&lr, fmt.Sprintf("GET %s  -  Response", path))
		j, err := json.Marshal(&lr)
		if err != nil {
			log.Fatalf("Error Marshalling ListResponse: %v\n", err)
		}

		res.WriteHeader(http.StatusOK)
		res.Write(j)

	} else if req.Method == http.MethodPost {
		// POST
		b, err := getBody(req)
		if err != nil {
			log.Printf("Error getting POST body: %v\n", err)
			res.WriteHeader(http.StatusBadRequest)
			res.Write([]byte(err.Error()))
			return
		}

		var m map[string]interface{}
		if err := json.Unmarshal(b, &m); err != nil {
			log.Printf("Error decoding Json Data: %v\n", err)
			res.WriteHeader(http.StatusBadRequest)
			res.Write([]byte(err.Error()))
			return
		}

		uuid := utils.GenerateUUID()
		meta := v2.Meta{ResourceType: v2.TYPE_USER, Location: v2.LOCATION_USER + uuid}
		m["meta"] = meta
		m["id"] = uuid
		doc, _ := json.Marshal(m)

		doc = reqFilter.UserPostRequest(doc, fmt.Sprintf("POST %s  -  Request", path))

		if err = utils.AddUser(doc, m["userName"].(string), uuid); err != nil {
			if err.Error() == "user_already_exists" {
				handleErrorResponse(&res, USER_ALREADY_EXISTS, http.StatusConflict)
			} else {
				handleErrorResponse(&res, fmt.Sprintf("Error adding user: %v", err), http.StatusInternalServerError)
			}
			return
		}

		doc = reqFilter.UserPostResponse(doc, fmt.Sprintf("POST %s  -  Response", path))

		res.WriteHeader(http.StatusCreated)
		if _, err = res.Write(doc); err != nil {
			// if _, err = res.Write(nil); err != nil { // test not returning user record
			log.Printf("Error returning AddUser call: %v\b", err)
		}
	} else {
		// NOT-SUPPORTED
		handleNotSupported(req, &res)
	}
}

// {GET/PUT/PATCH/DELETE} /goscim/scim/v2/Users/<id>
func handleUser(res http.ResponseWriter, req *http.Request) {
	// if didRedirect(&res, req) {
	// 	return
	// }
	res.Header().Add("content-type", content_type)

	parts := strings.Split(req.URL.Path[1:], "/")
	if len(parts) != 5 || parts[4] == "" {
		res.WriteHeader(http.StatusNotFound)
		res.Write(nil)
		fmt.Printf("Not Found: %v, %v\n", len(parts), parts)
		return
	}
	uuid := parts[4]
	path := fmt.Sprintf("/goscim/scim/v2/Users/%s", uuid)

	if req.Method == http.MethodDelete {
		// DELETE (not used by Okta)
		if err := utils.DelUser(uuid); err != nil {
			if err.Error() == NOT_FOUND {
				handleErrorForKeyLookup(&res, err, uuid)
				return
			} else {
				log.Printf("Error for DELETE /goscim/scim/v2/User/%v, err: %v\n\n", uuid, err)
			}
		}

		res.WriteHeader(http.StatusOK)
		res.Write(nil)
	} else if req.Method == http.MethodGet {
		// GET
		doc, err := utils.GetUserByUUID(uuid)
		if err != nil {
			handleErrorForKeyLookup(&res, err, uuid)
			return
		}

		user := embedUsersGroups([]interface{}{doc})

		user[0] = reqFilter.UserIdGetResponse(user[0].(string), fmt.Sprintf("GET %s  -  Response", path))

		res.WriteHeader(http.StatusOK)
		res.Write([]byte(user[0].(string)))

		// TEST - encoding as gzip
		// var m map[string]interface{}
		// fmt.Println(user[0].(string))
		// err = json.Unmarshal([]byte(user[0].(string)), &m)
		// fmt.Println(err)
		// fmt.Printf("%+v\n", m)
		// writeCompressedResponse(res, user[0].(string))
	} else {
		b, err := getBody(req)
		if err != nil {
			res.WriteHeader(http.StatusBadRequest)
			res.Write([]byte(err.Error()))
			return
		}

		if req.Method == http.MethodPut {
			// PUT
			b = reqFilter.UserIdPutRequest(b, fmt.Sprintf("PUT %s  -  Request", path))

			var m map[string]interface{}
			err := json.Unmarshal(b, &m)
			if err != nil {
				res.WriteHeader(http.StatusBadRequest)
				res.Write([]byte(err.Error()))
				return
			}

			ids, groups := buildGroupsMembersList(m["groups"].([]interface{}))
			m["groups"] = []interface{}{}
			u, _ := json.Marshal(m)
			userElement := fmt.Sprintf(`{"display":"%v","value":"%v"}`, m["userName"], uuid)

			if err := utils.UpdateUser(uuid, u, m["active"].(bool), userElement, ids, groups); err != nil {
				handleErrorForKeyLookup(&res, err, uuid)
				return
			}

			b = reqFilter.UserIdPutResponse(b, fmt.Sprintf("PUT %s  -  Response", path))

			res.WriteHeader(http.StatusOK)
			if _, err = res.Write(b); err != nil {
				log.Printf("Error replying for PUT /goscim/scim/v2/User/%v, err: %v\n\n", uuid, err)
			}

			//res.Header().Add("Retry-After", "1")
			//res.WriteHeader(http.StatusTooManyRequests)
		} else if req.Method == http.MethodPatch {
			// PATCH
			var ops v2.PatchOp
			if err := json.Unmarshal(b, &ops); err != nil {
				log.Printf("Error Unmarshalling JSON for PATCH /goscim/scim/v2/User/%v, err: %v\n\n", uuid, err)
				handleErrorResponse(&res, err.Error(), http.StatusInternalServerError)
				return
			}

			reqFilter.UserIdPatchRequest(&ops, fmt.Sprintf("PATCH %s  -  Request", path))

			patchUser := utils.UserPatch{}
			for _, v := range ops.Operations {
				val := v.Value.(map[string]interface{})
				if val["password"] != nil {
					patchUser.Password = true
					patchUser.PasswordValue = val["password"].(string)
				} else {
					patchUser.Active = true
					patchUser.ActiveValue = val["active"].(bool)
				}
			}

			if err := utils.PatchUser(uuid, patchUser); err != nil {
				if err.Error() == NOT_FOUND {
					handleErrorForKeyLookup(&res, err, uuid)
				} else {
					log.Printf("Error for PATCH /goscim/scim/v2/User/%v, err: %v\n\n", uuid, err)
					handleErrorResponse(&res, err.Error(), http.StatusInternalServerError)
				}
				return
			}

			// res.Header().Add("Retry-After", "300")
			// res.WriteHeader(http.StatusTooManyRequests)
			res.WriteHeader(http.StatusNoContent)
			res.Write(nil)
		} else {
			// NOT-SUPPORTED
			handleNotSupported(req, &res)
		}
	}
}

func embedUsersGroups(docs interface{}) []interface{} {
	var users []interface{}
	for _, v := range docs.([]interface{}) {
		user := v.([]interface{})[0].(string)
		var b strings.Builder
		b.WriteString(`"groups":[`)
		for i := 1; i < len(v.([]interface{})); i++ {
			if i > 1 {
				b.WriteString(",")
			}
			b.WriteString(v.([]interface{})[i].(string))
		}

		b.WriteString("]")
		user = strings.Replace(user, `"groups":[]`, b.String(), 1)
		users = append(users, user)
	}
	return users
}

func writeCompressedResponse(w http.ResponseWriter, body any) {
	w.Header().Set("Content-Encoding", "gzip")
	// res.Header().Add("content-type", content_type)
	// w.Header().Del("content-type")

	gw := gzip.NewWriter(w)
	defer gw.Close()
	json.NewEncoder(gw).Encode(body)
}

func handleServiceProviderConfig(res http.ResponseWriter, req *http.Request) {
	test := `
	{
  "schemas":[
    "urn:ietf:params:scim:schemas:core:2.0:User",
    "urn:okta:schemas:scim:providerconfig:2.0"
  ],
  "documentationUrl":"https://support.okta.com/scim-fake-page.html",
  "patch":{
    "supported":true
  },
  "bulk":{
    "supported":false
  },
  "filter":{
    "supported":true,
    "maxResults":100
  },
  "changePassword":{
    "supported":true
  },
  "sort":{
    "supported":false
  },
  "etag":{
    "supported":false
  },
  "authenticationSchemes":[
  ],
  "urn:okta:schemas:scim:providerconfig:2.0":{
    "userManagementCapabilities":[
      "GROUP_PUSH",
      "IMPORT_NEW_USERS",
      "IMPORT_PROFILE_UPDATES",
      "PUSH_NEW_USERS",
      "PUSH_PASSWORD_UPDATES",
      "PUSH_PENDING_USERS",
      "PUSH_PROFILE_UPDATES",
      "PUSH_USER_DEACTIVATION",
      "REACTIVATE_USERS"
    ]
  }
}
  `
	res.Header().Add("content-type", content_type)
	_, err := res.Write([]byte(test))
	if err != nil {
		log.Printf("handleServiceProviderConfig error: %s\n", err)
	}
}
