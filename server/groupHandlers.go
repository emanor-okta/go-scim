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

// {GET/POST} /scim/v2/Groups
func handleGroups(res http.ResponseWriter, req *http.Request) {
	// if didRedirect(&res, req) {
	// 	return
	// }
	res.Header().Add("content-type", content_type)

	if req.Method == http.MethodGet {
		// GET
		var grps interface{}
		q := getQuery(req.URL.Query())
		debugQueryParams(&q)
		if q.filter.displayName != "" {
			// ?filter=displayName eq <group name>
			g, err := utils.GetGroupByFilter(q.filter.displayName)
			if err != nil {
				handleEmptyListReturn(&res, err)
				return
			}
			grps = []interface{}{g}
		} else {
			// ?startIndex=<?>&count=<?>
			var err error
			grps, err = utils.GetGroupsByRange(q.startIndex, q.count)
			if err != nil {
				log.Printf("\n%v\n\n", err)
				handleEmptyListReturn(&res, err)
				return
			}
		}

		groups := embedGroupsMembers(grps)
		lr := buildListResponse(groups)
		j, err := json.Marshal(&lr)
		if err != nil {
			log.Fatalf("Error Marshalling ListResponse: %v\n", err)
		}
		fmt.Println(string(j))
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

		// TODO - SHOULD REMOVE THIS CHECK and make the AddGroup call fail if GROUP Already Exists
		if _, err = utils.GetGroupByFilter(m["displayName"].(string)); err == nil {
			// group already exist return 409
			handleErrorResponse(&res, GROUPALREADY_EXISTS, http.StatusConflict)
			return
		}

		uuid := utils.GenerateUUID()
		meta := v2.Meta{ResourceType: v2.TYPE_GROUP, Location: v2.LOCATION_GROUP + uuid}
		m["meta"] = meta
		m["id"] = uuid

		// if UsersPostReqFilter != nil {
		// 	doc = UsersPostReqFilter.UserPostRequest(doc)
		// }

		ids, mems := buildGroupsMembersList(m["members"].([]interface{}))

		b, _ = json.Marshal(m)
		m["members"] = []interface{}{}
		doc, _ := json.Marshal(m)

		if err = utils.AddGroup(doc, m["displayName"].(string), uuid, mems, ids); err != nil {
			res.WriteHeader(http.StatusOK)
			res.Write(nil)
			return
		}

		// if UsersPostResFilter != nil {
		// 	doc = UsersPostResFilter.UserPostResponse(doc)
		// }

		res.WriteHeader(http.StatusCreated)
		res.Write(b)
	} else {
		// NOT-SUPPORTED
		handleNotSupported(req, &res)
	}
}

// {GET/PUT/PATCH/DELETE} /scim/v2/Groups/<id>
func handleGroup(res http.ResponseWriter, req *http.Request) {
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
		// DELETE
		if err := utils.DelGroup(uuid); err != nil {
			log.Printf("Error for DELETE /scim/v2/Groups/%v, err: %v\n\n", uuid, err)
			handleErrorResponse(&res, err.Error(), http.StatusInternalServerError)
			return
		}

		res.WriteHeader(http.StatusNoContent)
		res.Write(nil)
	} else if req.Method == http.MethodGet {
		// GET
		grp, err := utils.GetGroupByUUID(uuid)
		if err != nil {
			handleErrorForKeyLookup(&res, err)
			return
		}

		groups := embedGroupsMembers([]interface{}{grp})
		res.WriteHeader(http.StatusOK)
		res.Write([]byte(groups[0].(string)))
	} else {
		b, err := getBody(req)
		if err != nil {
			res.WriteHeader(http.StatusInternalServerError)
			res.Write(nil)
			return
		}

		if req.Method == http.MethodPut {
			// PUT
			// if UsersPutReqFilter != nil {
			// 	b = UsersPutReqFilter.UserIdPutRequest(b)
			// }
			var m map[string]interface{}
			if err := json.Unmarshal(b, &m); err != nil {
				log.Printf("Error decoding Json Data: %v\n", err)
				res.WriteHeader(http.StatusOK)
				res.Write(nil)
				return
			}

			// Okta should send a PUT with prior meta data?
			meta := v2.Meta{ResourceType: v2.TYPE_GROUP, Location: v2.LOCATION_GROUP + uuid}
			m["meta"] = meta
			m["id"] = uuid

			ids, mems := buildGroupsMembersList(m["members"].([]interface{}))

			b, _ = json.Marshal(m)
			m["members"] = []interface{}{}
			doc, _ := json.Marshal(m)

			if err = utils.UpdateGroup(doc, m["displayName"].(string), uuid, mems, ids); err != nil {
				handleErrorResponse(&res, err.Error(), http.StatusInternalServerError)
				return
			}

			// if UsersPutResFilter != nil {
			// 	b = UsersPutResFilter.UserIdPutResponse(b)
			// }

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

			// if UsersPatchReqFilter != nil {
			// 	UsersPatchReqFilter.UserIdPatchRequest(&ops)
			// }

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

// CAN DELETE
// func getGroupMembers(group []map[string]interface{}) {
// 	var ids []string
// 	for _, v := range group {
// 		ids = append(ids, v["id"].(string))
// 	}

// 	result, err := utils.GetGroupMembers(ids)
// 	if err != nil {
// 		return
// 	}

// 	for i, v := range result.([]interface{}) {
// 		if len(v.([]interface{})) == 0 {
// 			continue
// 		}

// 		var b strings.Builder
// 		b.WriteString("[")
// 		for i_, v_ := range v.([]interface{}) {
// 			if i_ > 0 {
// 				b.WriteString(",")
// 			}
// 			b.WriteString(v_.(string))
// 		}

// 		b.WriteString("]")
// 		// fmt.Printf("groups: %v\n\n", b.String())
// 		var j interface{}
// 		if err = json.Unmarshal([]byte(b.String()), &j); err != nil {
// 			fmt.Printf("Error groupHandlers.getGroupMembers Unmarshall error: %v\n", err)
// 			continue
// 		}
// 		group[i]["members"] = j
// 	}
// }

/*
	Used by both Group/User requests to prepare either a list of members or groups to send to Redis
*/
func buildGroupsMembersList(a []interface{}) ([]string, []string) {
	mems := []string{}
	ids := []string{}
	for _, g := range a {
		// g := v.(map[string]interface{})
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
