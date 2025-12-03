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

// {GET/POST} /goscim/scim/v2/Groups
func handleGroups(res http.ResponseWriter, req *http.Request) {
	// if didRedirect(&res, req) {
	// 	return
	// }
	res.Header().Add("content-type", content_type)
	path := "/goscim/scim/v2/Groups"

	if req.Method == http.MethodGet {
		// GET
		var grps interface{}
		q := getQuery(req.URL.Query())
		if debugQuery {
			debugQueryParams(&q)
		}

		if q.filter.displayName != "" {
			// ?filter=displayName eq <group name>
			path = fmt.Sprintf("%s?filter=displayName eq %s&startIndex=%v&count=%v", path, q.filter.displayName, q.startIndex, q.count)
			g, err := utils.GetGroupByFilter(q.filter.displayName)
			if err != nil {
				handleEmptyListReturn(&res, err, &reqFilter, fmt.Sprintf("GET %s  -  Response", path))
				return
			}
			grps = []interface{}{g}
		} else {
			// ?startIndex=<?>&count=<?>
			path = fmt.Sprintf("%s?startIndex=%v&count=%v", path, q.startIndex, q.count)
			var err error
			grps, err = utils.GetGroupsByRange(q.startIndex, q.count)
			if err != nil {
				log.Printf("\n%v\n\n", err)
				handleEmptyListReturn(&res, err, &reqFilter, fmt.Sprintf("GET %s  -  Response", path))
				return
			}
		}

		groups := embedGroupsMembers(grps)
		// reqFilter.GroupsGetResponse(groups) <-- change to pass v2.ListResponse

		lr := buildListResponse(groups)
		reqFilter.GroupsGetResponse(&lr, fmt.Sprintf("GET %s  -  Response", path))

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
		meta := v2.Meta{ResourceType: v2.TYPE_GROUP, Location: v2.LOCATION_GROUP + uuid}
		m["meta"] = meta
		m["id"] = uuid

		reqFilter.GroupsPostRequest(m, fmt.Sprintf("POST %s  -  Request", path))
		ids, mems := buildGroupsMembersList(m["members"].([]interface{}))

		b, _ = json.Marshal(m)
		m["members"] = []interface{}{}
		doc, _ := json.Marshal(m)
		groupSnippet := fmt.Sprintf(`{"display":"%v","value":"%v"}`, m["displayName"], uuid)

		if err = utils.AddGroup(doc, m["displayName"].(string), uuid, groupSnippet, mems, ids); err != nil {
			if err.Error() == "group_already_exists" {
				handleErrorResponse(&res, GROUPALREADY_EXISTS, http.StatusConflict)
			} else {
				handleErrorResponse(&res, fmt.Sprintf("Error adding group: %v", err), http.StatusInternalServerError)
			}
			return
		}

		b = reqFilter.GroupsPostResponse(b, fmt.Sprintf("POST %s  -  Response", path))
		res.WriteHeader(http.StatusCreated)
		res.Write(b)
	} else {
		// NOT-SUPPORTED
		handleNotSupported(req, &res)
	}
}

// test map
// var putMap = make(map[string]string, 0)
// end test map

// {GET/PUT/PATCH/DELETE} /goscim/scim/v2/Groups/<id>
func handleGroup(res http.ResponseWriter, req *http.Request) {
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
	path := fmt.Sprintf("/goscim/scim/v2/Groups/%s", uuid)

	// if req.Method == http.MethodGet {
	// 	// if _, ok := putMap[uuid]; ok {
	// 	// 	//if uuid != "b415418d-5a3f-48c4-88f9-5803839cfbd0" {
	// 	// 	delete(putMap, uuid)
	// 	// 	res.Header().Add("Retry-After", "1")
	// 	// 	res.WriteHeader(http.StatusTooManyRequests)
	// 	// 	return
	// 	// }
	// 	// putMap[uuid] = uuid
	// } else if req.Method == http.MethodPut {
	// 	//putMap[uuid] = uuid
	// 	res.Header().Add("Retry-After", "1")
	// 	res.WriteHeader(http.StatusTooManyRequests)
	// 	return
	// }

	if req.Method == http.MethodDelete {
		// DELETE
		if err := utils.DelGroup(uuid); err != nil {
			log.Printf("Error for DELETE /goscim/scim/v2/Groups/%v, err: %v\n\n", uuid, err)
			handleErrorResponse(&res, err.Error(), http.StatusInternalServerError)
			return
		}

		res.WriteHeader(http.StatusNoContent)
		res.Write(nil)
	} else if req.Method == http.MethodGet {
		// GET
		grp, err := utils.GetGroupByUUID(uuid)
		if err != nil {
			handleErrorForKeyLookup(&res, err, uuid)
			return
		}

		groups := embedGroupsMembers([]interface{}{grp})
		groups[0] = reqFilter.GroupsIdGetResponse(groups[0], fmt.Sprintf("GET %s  -  Response", path))
		res.WriteHeader(http.StatusOK)
		// res.Write([]byte(groups[0].(string)))
		res.Write(groups[0].([]byte))
	} else {
		b, err := getBody(req)
		if err != nil {
			res.WriteHeader(http.StatusInternalServerError)
			res.Write(nil)
			return
		}

		if req.Method == http.MethodPut {
			// PUT
			var m map[string]interface{}
			if err := json.Unmarshal(b, &m); err != nil {
				log.Printf("Error decoding Json Data: %v\n", err)
				res.WriteHeader(http.StatusBadRequest)
				res.Write([]byte(err.Error()))
				return
			}

			// Okta should send a PUT with prior meta data?
			meta := v2.Meta{ResourceType: v2.TYPE_GROUP, Location: v2.LOCATION_GROUP + uuid}
			m["meta"] = meta
			m["id"] = uuid

			reqFilter.GroupsIdPutRequest(m, fmt.Sprintf("PUT %s  -  Request", path))
			ids, mems := buildGroupsMembersList(m["members"].([]interface{}))

			b, _ = json.Marshal(m)
			m["members"] = []interface{}{}
			doc, _ := json.Marshal(m)
			groupSnippet := fmt.Sprintf(`{"display":"%v","value":"%v"}`, m["displayName"], uuid)
			displayName := ""
			if value, exists := m["displayName"]; exists && value != nil {
				displayName = m["displayName"].(string)
			}

			if err = utils.UpdateGroup(doc, displayName, uuid, groupSnippet, mems, ids); err != nil {
				handleErrorResponse(&res, err.Error(), http.StatusInternalServerError)
				return
			}

			b = reqFilter.GroupsIdPutResponse(b, fmt.Sprintf("PUT %s  -  Response", path))

			res.WriteHeader(http.StatusOK)
			res.Write(b)
		} else if req.Method == http.MethodPatch {
			// PATCH
			var ops v2.PatchOp
			if err := json.Unmarshal(b, &ops); err != nil {
				log.Printf("Error Unmarshalling JSON for PATCH /scim/v2/Groups/%v, err: %v\n\n", uuid, err)
				handleErrorResponse(&res, err.Error(), http.StatusInternalServerError)
				return
			}

			reqFilter.GroupsIdPatchRequest(&ops, fmt.Sprintf("PATCH %s  -  Request", path))

			// process list of operations  ** TODO - Should this be run in a single transaction ?? **
			for _, o := range ops.Operations {
				if o.Op == v2.GROUP_ADD {
					var a []string
					var ids []string
					for _, v := range o.Value.([]interface{}) {
						m := v.(map[string]interface{})
						a = append(a, fmt.Sprintf(`{"value":"%v","display":"%v"}`, m["value"].(string), m["display"].(string)))
						ids = append(ids, m["value"].(string))
					}
					if err := utils.AddGroupMembers(uuid, ids, a); err != nil {
						handleErrorResponse(&res, err.Error(), http.StatusInternalServerError)
						return
					}
				} else if o.Op == v2.GROUP_REMOVE {
					if err := utils.RemoveGroupMembers(uuid, strings.ReplaceAll(strings.Split(o.Path, `"`)[1], `\`, "")); err != nil {
						handleErrorResponse(&res, err.Error(), http.StatusInternalServerError)
						return
					}
				} else if o.Op == v2.GROUP_REPLACE {
					if o.Path == v2.GROUP_PATH_MEMBERS {
						// replace all group members
						ids, members := buildGroupsMembersList(o.Value.([]interface{}))
						if err := utils.ReplaceGroupMembers(uuid, ids, members); err != nil {
							handleErrorResponse(&res, err.Error(), http.StatusInternalServerError)
							return
						}
					} else {
						// update group name
						name := o.Value.(map[string]interface{})["displayName"].(string)
						if err = utils.UpdateGroupName(uuid, name); err != nil {
							handleErrorResponse(&res, err.Error(), http.StatusInternalServerError)
							return
						}
					}
				} else {
					log.Printf("Unkown Group Patch Op: %v\n", o.Op)
					continue
				}
			}

			res.WriteHeader(http.StatusNoContent)
			res.Write(nil)
		} else {
			// NOT-SUPPORTED
			handleNotSupported(req, &res)
		}
	}
}

/*
Used by both Group/User requests to prepare either a list of members or groups to send to Redis
*/
func buildGroupsMembersList(a []interface{}) ([]string, []string) {
	mems := []string{}
	ids := []string{}
	for _, g := range a {
		g := g.(map[string]interface{})
		mems = append(mems, fmt.Sprintf(`{"display":"%v","value":"%v"}`, g["display"].(string), g["value"].(string)))
		ids = append(ids, g["value"].(string))
	}
	return ids, mems
}

func embedGroupsMembers(docs interface{}) []interface{} {
	var groups []interface{}
	for _, v := range docs.([]interface{}) {
		group := v.([]interface{})[0].(string)
		var b strings.Builder
		b.WriteString(`"members":[`)
		for i := 1; i < len(v.([]interface{})); i++ {
			if i > 1 {
				b.WriteString(",")
			}
			b.WriteString(v.([]interface{})[i].(string))
		}

		b.WriteString("]")
		group = strings.Replace(group, `"members":[]`, b.String(), 1)
		groups = append(groups, group)
	}
	return groups
}
