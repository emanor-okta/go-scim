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

/*
 * SCIM Handlers
 */

// {GET/POST} /scim/v2/Users
func handleUsers(res http.ResponseWriter, req *http.Request) {
	// if didRedirect(&res, req) {
	// 	return
	// }
	res.Header().Add("content-type", content_type)

	if req.Method == http.MethodGet {
		// GET
		var docs interface{}
		q := getQuery(req.URL.Query())
		debugQueryParams(&q)

		if q.filter.userName != "" {
			// ?filter=username eq <username>
			user, err := utils.GetUserByFilter(q.filter.userName)
			if err != nil {
				handleEmptyListReturn(&res, err)
				return
			}
			docs = []interface{}{}
			docs = append(docs.([]interface{}), user)
		} else {
			// ?startIndex=<?>&count=<?>
			var err error
			docs, err = utils.GetUsersByRange(q.startIndex, q.count)
			if err != nil {
				handleEmptyListReturn(&res, err)
				return
			}
		}

		users := embedUsersGroups(docs)
		lr := buildListResponse(users)

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
		meta := v2.Meta{ResourceType: v2.TYPE_USER, Location: v2.LOCATION_USER + uuid}
		m["meta"] = meta
		m["id"] = uuid
		doc, _ := json.Marshal(m)

		if UsersPostReqFilter != nil {
			doc = UsersPostReqFilter.UserPostRequest(doc)
		}

		if err = utils.AddUser(doc, m["userName"].(string), uuid); err != nil {
			if err.Error() == "user_already_exists" {
				handleErrorResponse(&res, USER_ALREADY_EXISTS, http.StatusConflict)
			} else {
				handleErrorResponse(&res, fmt.Sprintf("Error adding user: %v", err), http.StatusInternalServerError)
			}
			return
		}

		if UsersPostResFilter != nil {
			doc = UsersPostResFilter.UserPostResponse(doc)
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
	// if didRedirect(&res, req) {
	// 	return
	// }
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
		if err := utils.DelUser(uuid); err != nil {
			if err.Error() == NOT_FOUND {
				handleErrorForKeyLookup(&res, err)
				return
			} else {
				log.Printf("Error for DELETE /scim/v2/User/%v, err: %v\n\n", uuid, err)
			}
		}

		res.WriteHeader(http.StatusOK)
		res.Write(nil)
	} else if req.Method == http.MethodGet {
		// GET
		doc, err := utils.GetUserByUUID(uuid)
		if err != nil {
			handleErrorForKeyLookup(&res, err)
			return
		}

		user := embedUsersGroups([]interface{}{doc})

		//testing till add a GET filter
		// var m map[string]interface{}
		// json.Unmarshal([]byte(doc), &m)
		// m["urn:ietf:params:scim:schemas:extension:enterprise:2.0:User"] = nil
		// fmt.Printf("type: %T\n", m["name"])
		// m["name"].(map[string]interface{})["formatted"] = "First Last"
		// m["emails"].([]interface{})[0].(map[string]interface{})["value"] = "aaa@aaa.com<mailto:aaa@aaa.com>"
		// delete(m["meta"].(map[string]interface{}), "location")
		// delete(m, "displayName")
		// delete(m, "groups")
		// delete(m, "locale")
		// delete(m, "externalId")
		// delete(m, "password")
		// delete(m, "phoneNumbers")
		// m["userName"] = "aaa@aaa.com<mailto:aaa@aaa.com>"
		// b, _ := json.Marshal(m)

		res.WriteHeader(http.StatusOK)
		res.Write([]byte(user[0].(string)))
	} else {
		b, err := getBody(req)
		if err != nil {
			res.WriteHeader(http.StatusInternalServerError)
			res.Write(nil)
			return
		}

		if req.Method == http.MethodPut {
			if UsersPutReqFilter != nil {
				b = UsersPutReqFilter.UserIdPutRequest(b)
			}

			var m map[string]interface{}
			json.Unmarshal(b, &m)
			ids, groups := buildGroupsMembersList(m["groups"].([]interface{}))
			m["groups"] = []interface{}{}
			u, _ := json.Marshal(m)
			userElement := fmt.Sprintf(`{"display":"%v","value":"%v"}`, m["userName"], uuid)

			if err := utils.UpdateUser(uuid, u, m["active"].(bool), userElement, ids, groups); err != nil {
				handleErrorForKeyLookup(&res, err)
				return
			}

			if UsersPutResFilter != nil {
				b = UsersPutResFilter.UserIdPutResponse(b)
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
				handleErrorResponse(&res, err.Error(), http.StatusInternalServerError)
				return
			}

			if UsersPatchReqFilter != nil {
				UsersPatchReqFilter.UserIdPatchRequest(&ops)
			}

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
					handleErrorForKeyLookup(&res, err)
				} else {
					log.Printf("Error for PATCH /scim/v2/User/%v, err: %v\n\n", uuid, err)
					handleErrorResponse(&res, err.Error(), http.StatusInternalServerError)
				}
				return
			}

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
