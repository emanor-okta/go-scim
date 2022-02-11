package utils

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"

	"github.com/go-redis/redis/v8"
)

// https://redis.io/commands
// docker run --name go-scim-redis -p 6379:6379 -d redis redis-server --save 60 1 --loglevel warning
// docker exec -i go-scim-redis bash (cd /usr/local/bin)

const (
	USERS_KEY                 = "_users"
	GROUPS_KEY                = "_groups"
	USERS_LOOKUP_KEY          = "_users_lookup"
	USERS_REVERSE_LOOKUP_KEY  = "_users_reverse_lookup"
	GROUPS_LOOKUP_KEY         = "_groups_lookup"
	GROUPS_REVERSE_LOOKUP_KEY = "_groups_reverse_lookup"
	EMBEDDED_MEMBERS          = "_members"
	EMBEDDED_GROUPS           = "_groups"
	// ADD_USER_CMD              = "redis.call('set', KEYS[1], ARGV[1]); redis.call('HSet', KEYS[2], ARGV[2], ARGV[3]); redis.call('LPush', KEYS[3], ARGV[3]); return 1;"
	// DEL_USER_CMD = "redis.call('del', KEYS[1]); redis.call('HDel', KEYS[2], ARGV[1]); redis.call('LRem', KEYS[3], 0, ARGV[2]); return 1;"
	// DEL_GROUP_CMD             = `local n = redis.call('HGet',KEYS[1],ARGV[1]); local m = redis.call('HKeys',KEYS[2]); for k,v in ipairs(m) do redis.call('HDel',v.."_groups",ARGV[1]); end; redis.call('del',KEYS[3]);redis.call('HDel',KEYS[4],n);redis.call('LRem',KEYS[5],0,ARGV[1]);redis.call('Del',KEYS[2]);redis.call('HDel',KEYS[1],ARGV[1]);return 1;`
	// GROUP_MEMBERS = "t[%v]=redis.call('HVals',KEYS[%v]);"
	// CHANGE_GROUP_NAME = "local k=redis.call('HGet',KEYS[1],ARGV[1]);redis.call('HDel',KEYS[2],k);redis.call('HSet',KEYS[1],ARGV[1],ARGV[2]);redis.call('HSet',KEYS[2],ARGV[2],ARGV[1]);redis.call('Set',KEYS[3],ARGV[3]);return 1;"
)

type UserPatch struct {
	Active        bool
	ActiveValue   bool
	Password      bool
	PasswordValue string
}

type LuaScriptSHA struct {
	LuaGetByRange          string
	LuaGetByFilter         string
	LuaGetByUUID           string
	LuaAddUser             string
	LuaUpdateUuserActive   string
	LuaUpdateUuserInActive string
	LuaDeleteUser          string
	LuaPatchUser           string
	LuaDeleteGroup         string
	LuaUpdateGroupName     string
}

var rdb *redis.Client
var ctx context.Context = context.Background()
var luaScripts LuaScriptSHA

func InitializeRedis(config *Configuration) error {
	rdb = redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "", // no password set
		DB:       0,  // use default DB
	})

	//test connection
	intCmd := rdb.ClientID(ctx)
	if intCmd.Err() != nil {
		return intCmd.Err()
	}

	luaScripts.LuaGetByRange = loadLuaScriptIntoCache(LUA_GET_BY_RANGE)
	luaScripts.LuaGetByFilter = loadLuaScriptIntoCache(LUA_GET_BY_FILTER)
	luaScripts.LuaGetByUUID = loadLuaScriptIntoCache(LUA_GET_BY_UUID)
	luaScripts.LuaAddUser = loadLuaScriptIntoCache(LUA_ADD_USER)
	luaScripts.LuaUpdateUuserActive = loadLuaScriptIntoCache(LUA_UPDATE_USER_ACTIVE)
	luaScripts.LuaUpdateUuserInActive = loadLuaScriptIntoCache(LUA_UPDATE_USER_INACTIVE)
	luaScripts.LuaDeleteUser = loadLuaScriptIntoCache(LUA_DELETE_USER)
	luaScripts.LuaPatchUser = loadLuaScriptIntoCache(LUA_PATCH_USER)
	luaScripts.LuaDeleteGroup = loadLuaScriptIntoCache(LUA_DELETE_GROUP)
	luaScripts.LuaUpdateGroupName = loadLuaScriptIntoCache(LUA_UPDATE_GROUP_NAME)

	return nil
}

func loadLuaScriptIntoCache(script string) string {
	result := rdb.ScriptLoad(ctx, script)
	if result.Err() != nil {
		log.Fatalf("Error loading Lua Script into redis..\nscript: %v\nerror: %v\n\n", script, result.Err().Error())
	}

	return result.Val()
}

// func SaveDoc(key string, doc interface{}) error {
// 	err := rdb.Set(ctx, key, doc, 0).Err()
// 	if err != nil {
// 		log.Printf("Redis Error Saving, Key: %v, error: %v\n\n", key, err)
// 		return err
// 	}
// 	return nil
// }

func UpdateDoc(key string, doc interface{}) error {
	args := redis.SetArgs{Mode: "XX"}
	_, err := rdb.SetArgs(ctx, key, doc, args).Result()
	if err != nil {
		if err != redis.Nil {
			log.Printf("Redis Error Saving, Key: %v, error: %v\n\n", key, err)
			return err
		}
		return errors.New("not_found")
	}
	return nil
}

// func GetDoc(key string) (string, error) {
// 	doc, err := rdb.Get(ctx, key).Result()
// 	if err != nil {
// 		if err != redis.Nil {
// 			log.Printf("Redis Error Getting, Key: %v, error: %v\n\n", key, err)
// 			return "", err
// 		}
// 		return "", errors.New("not_found")
// 	}
// 	return doc, nil
// }

// func AddLookup(user, uuid string) error {
// 	if err := rdb.HSet(ctx, USERS_LOOKUP_KEY, user, uuid).Err(); err != nil {
// 		log.Printf("Redis Error Adding Lookup, User: %v, UUID: %v, error: %v\n\n", user, uuid, err)
// 		return err
// 	}
// 	return nil
// }

// func ListPush(key, value string) error {
// 	if err := rdb.LPush(ctx, key, value).Err(); err != nil {
// 		log.Printf("Redis Error Pushing to List: %s for: %v, err: %v\n\n", key, value, err)
// 		return err
// 	}
// 	return nil
// }

// func lRange(key string, startIndex, count int) ([]string, error) {
// 	docs, err := rdb.LRange(ctx, key, int64(startIndex-1), int64(startIndex+count-2)).Result()
// 	if err != nil {
// 		log.Printf("Redis Error LRange: %s, err: %v\n\n", key, err)
// 		return nil, err
// 	}
// 	return docs, nil
// }

// func getByFilterORIG(search, lookupKey string) (string, error) {
// 	uuid, err := rdb.HGet(ctx, lookupKey, search).Result()
// 	if err != nil {
// 		if err != redis.Nil {
// 			log.Printf("Redis Error Getting Lookup: %v, error: %v\n\n", search, err)
// 			return "", err
// 		}
// 		return "", errors.New("not_found")
// 	}
// 	return GetDoc(uuid)
// }

func getByRange(startIndex int, count int, key, embedded_key string) (interface{}, error) {
	// cmd := `local ids=redis.call('LRange',KEYS[1],ARGV[1],ARGV[2]);local outer={};local oi=1;for k,id in ipairs(ids) do local inner={};inner[1]=redis.call('Get',id);local i=2;local l=redis.call('HVals',id..KEYS[2]);for k2,v in ipairs(l) do inner[i]=v;i=i+1;end;outer[oi]=inner;oi=oi+1;end;return outer;`
	// fmt.Println(LUA_GET_BY_RANGE)
	result, err := rdb.EvalSha(ctx, luaScripts.LuaGetByRange, []string{key, embedded_key}, []string{fmt.Sprintf("%v", startIndex-1), fmt.Sprintf("%v", startIndex-1+count-1)}).Result()
	if err != nil {
		log.Printf("Redis Error Getting %v by Range, startIndex: %v, count: %v\nerr: %v\n\n", key, startIndex, count, err)
		return nil, err
	}

	return result, nil
}

func getByFilter(name, key, embedded_key string) (interface{}, error) {
	// cmd := `local uuid=redis.call('HGet', KEYS[1],ARGV[1]);if not uuid then return nil; end;local t={};t[1]=redis.call('Get',uuid);local i=2;local l=redis.call('HVals',uuid..KEYS[2]);for k,v in ipairs(l) do t[i]=v;i=i+1;end;return t;`
	result, err := rdb.EvalSha(ctx, luaScripts.LuaGetByFilter, []string{key, embedded_key}, []string{name}).Result()
	if err != nil {
		if err != redis.Nil {
			log.Printf("Redis Error Getting by Filter: %s\nerr: %v\n\n", name, err)
			return nil, err
		}
		return nil, errors.New("not_found")
	}

	return result, nil
}

func getByUUID(uuid, embedded_key string) (interface{}, error) {
	// cmd := `local doc=redis.call('Get', KEYS[1]);if not doc then return nil; end;local t={};t[1]=doc;local i=2;local l=redis.call('HVals',KEYS[1]..KEYS[2]);for k,v in ipairs(l) do t[i]=v;i=i+1;end;return t;`
	result, err := rdb.EvalSha(ctx, luaScripts.LuaGetByUUID, []string{uuid, embedded_key}).Result()
	if err != nil {
		if err != redis.Nil {
			log.Printf("Redis Error Getting Document by UUID: %s\nerr: %v\n\n", uuid, err)
			return nil, err
		}
		return nil, errors.New("not_found")
	}

	return result, nil
}

/*
 * User specific functions
 */

func GetUsersByRange(startIndex int, count int) (interface{}, error) {
	return getByRange(startIndex, count, USERS_KEY, EMBEDDED_GROUPS)
}

func GetUserByFilter(user string) (interface{}, error) {
	return getByFilter(user, USERS_LOOKUP_KEY, EMBEDDED_GROUPS)
}

func GetUserByUUID(uuid string) (interface{}, error) {
	return getByUUID(uuid, EMBEDDED_GROUPS)
}

func AddUser(doc []byte, userName, uuid string) error {
	// cmd := `if redis.call('HExists',KEYS[2],ARGV[2]) == 1 then return nil;end;redis.call('set', KEYS[1], ARGV[1]);redis.call('HSet', KEYS[2], ARGV[2], ARGV[3]);redis.call('HSet', KEYS[4], ARGV[3], ARGV[2]);redis.call('LPush', KEYS[3], ARGV[3]);return 1;`
	if err := rdb.EvalSha(ctx, luaScripts.LuaAddUser, []string{uuid, USERS_LOOKUP_KEY, USERS_KEY, USERS_REVERSE_LOOKUP_KEY}, doc, userName, uuid).Err(); err != nil {
		if err != redis.Nil {
			log.Printf("Redis Error Adding New User: %s\nerr: %v\n\n", doc, err)
			return err
		}
		return errors.New("user_already_exists")
	}
	return nil
}

func UpdateUser(uuid string, doc []byte, active bool, userElement string, ids, groups []string) error {
	var luaScript string
	args := []interface{}{doc, userElement, len(ids)}
	if active {
		luaScript = luaScripts.LuaUpdateUuserActive
		for i, v := range ids {
			args = append(args, v)
			args = append(args, groups[i])
		}
	} else {
		luaScript = luaScripts.LuaUpdateUuserInActive
	}

	if err := rdb.EvalSha(ctx, luaScript, []string{uuid}, args...).Err(); err != nil {
		if err != redis.Nil {
			log.Printf("Redis Error Adding New User: %s\nerr: %v\n\n", doc, err)
			return err
		}
		return errors.New("user_already_exists")
	}
	return nil
}

func DelUser(uuid string) error {
	// cmd := `local n=redis.call('HGet',KEYS[4],ARGV[1]);if not n then return nil;end;redis.call('HDel',KEYS[2],n);redis.call('HDel',KEYS[4],ARGV[1]);redis.call('Del',KEYS[1]);redis.call('LRem',KEYS[3],0,ARGV[1]);local grps=redis.call('HKeys',KEYS[1].."_groups");redis.call('Del',KEYS[1].."_groups");for k,v in ipairs(grps) do redis.call('HDel',v.."_members",ARGV[1]);end;return 1;`
	if err := rdb.EvalSha(ctx, luaScripts.LuaDeleteUser, []string{uuid, USERS_LOOKUP_KEY, USERS_KEY, USERS_REVERSE_LOOKUP_KEY}, uuid /*, userName*/).Err(); err != nil {
		if err != redis.Nil {
			log.Printf("Redis Error Deleting User: %s\nerr: %v\n\n", uuid, err)
			return err
		}
		return errors.New("not_found")
	}
	return nil
}

func PatchUser(uuid string, userPatch UserPatch) error {
	// cmd := `local u=redis.call('Get',KEYS[1]);if not u then return nil;end;if KEYS[2]=="true" then u=string.gsub(u,"\"active\":.-,","\"active\":"..ARGV[1]..",");  if ARGV[1]=="false" then local grps=redis.call('HKeys',KEYS[1].."_groups");redis.call('Del',KEYS[1].."_groups");for k,v in ipairs(grps) do redis.call('HDel',v.."_members",KEYS[1]);end;end;   end;if KEYS[3]=="true" then u=string.gsub(u,"\"password\":\".-\"","\"password\":\""..ARGV[2].."\"");end;redis.call('Set',KEYS[1],u); return 1; `
	// fmt.Println(LUA_PATCH_USER)
	if err := rdb.EvalSha(ctx, luaScripts.LuaPatchUser, []string{uuid, fmt.Sprintf("%v", userPatch.Active), fmt.Sprintf("%v", userPatch.Password)}, fmt.Sprintf("%v", userPatch.ActiveValue), userPatch.PasswordValue).Err(); err != nil {
		if err != redis.Nil {
			log.Printf("Redis Error Patching User: %s\nerr: %v\n\n", uuid, err)
			return err
		}
		return errors.New("not_found")
	}
	return nil
}

/*
 * Group specific functions
 */

func GetGroupByFilter(group string) (interface{}, error) {
	return getByFilter(group, GROUPS_LOOKUP_KEY, EMBEDDED_MEMBERS)
}

func GetGroupByUUID(uuid string) (interface{}, error) {
	return getByUUID(uuid, EMBEDDED_MEMBERS)
}

func GetGroupsByRange(startIndex int, count int) (interface{}, error) {
	return getByRange(startIndex, count, GROUPS_KEY, EMBEDDED_MEMBERS)
}

// func GetGroupMembers(ids []string) (interface{}, error) {
// 	var b strings.Builder
// 	b.WriteString("local t={};")
// 	for i, v := range ids {
// 		b.WriteString(fmt.Sprintf(GROUP_MEMBERS, i+1, i+1))
// 		ids[i] = fmt.Sprintf("%v_members", v)
// 	}
// 	b.WriteString("return t;")
// 	result, err := rdb.Eval(ctx, b.String(), ids).Result()
// 	if err != nil {
// 		fmt.Printf("\n\nError redis.GetGroupMembers rdb.Eval: %v\n\n", err)
// 		return nil, err
// 	}
// 	return result, nil
// }

func AddGroup(doc []byte, groupName, uuid string, members, ids []string) error {
	return saveGroupBits(doc, groupName, uuid, members, ids, true)
}

func UpdateGroup(doc []byte, groupName, uuid string, members, ids []string) error {
	return saveGroupBits(doc, groupName, uuid, members, ids, false)
}

func saveGroupBits(doc []byte, groupName, uuid string, members, ids []string, create bool) error {
	var args, keys []string
	var b strings.Builder
	b.WriteString("redis.call('Set',KEYS[1],ARGV[1]);")
	var argI int
	if create {
		b.WriteString("redis.call('HSet',KEYS[2],ARGV[2],ARGV[3]);redis.call('HSet',KEYS[3],ARGV[3],ARGV[2]);redis.call('LPush',KEYS[4],ARGV[4]);")
		args = append(args, string(doc), groupName, uuid, uuid)
		keys = append(keys, uuid, GROUPS_LOOKUP_KEY, GROUPS_REVERSE_LOOKUP_KEY, GROUPS_KEY)
		argI = 5
	} else {
		b.WriteString(`local j=redis.call('HKeys',KEYS[5]);for k,v in ipairs(j) do redis.call('HDel',v.."_groups",ARGV[2]);end;redis.call('Del',KEYS[5]);`)
		b.WriteString("local k=redis.call('HGet',KEYS[2],ARGV[2]);redis.call('HDel',KEYS[3],k);redis.call('HSet',KEYS[3],ARGV[3],ARGV[2]);redis.call('HSet',KEYS[2],ARGV[2],ARGV[3]);redis.call('Del',KEYS[4]);")
		args = append(args, string(doc), uuid, groupName)
		keys = append(keys, uuid, GROUPS_REVERSE_LOOKUP_KEY, GROUPS_LOOKUP_KEY, uuid+"_members")
		argI = 4
	}

	keys = append(keys, uuid+"_members")
	if len(ids) > 0 {
		// keys = append(keys, uuid+"_members")
		b.WriteString("redis.call('HSet',KEYS[5]")
		var i int
		var v string
		for i, v = range members {
			b.WriteString(fmt.Sprintf(",ARGV[%v],ARGV[%v]", argI, argI+1))
			args = append(args, ids[i], v)
			argI += 2
		}

		b.WriteString(fmt.Sprintf(`);local l=redis.call('HKeys',KEYS[5]);for k,v in ipairs(l) do redis.call('HSet',v.."_groups",ARGV[%v],ARGV[%v]); end;return 1`, argI+1, argI))
		args = append(args, fmt.Sprintf(`{"value":"%v","display":"%v"}`, uuid, groupName), uuid)

		if err := rdb.Eval(ctx, b.String(), keys, args).Err(); err != nil {
			log.Printf("Redis Error Adding Group with Members: %s\nerr: %v\n\n", doc, err)
			return err
		}
	} else {
		b.WriteString("return 1;")
		if err := rdb.Eval(ctx, b.String(), keys, args).Err(); err != nil {
			log.Printf("Redis Error Adding Group without Members: %s\nerr: %v\n\n", doc, err)
			return err
		}
	}
	return nil
}

func AddGroupMembers(uuid string, ids, values []string) error {
	var groups, members strings.Builder
	var keys, args []string
	members.WriteString("redis.call('HSet',KEYS[1]")
	groups.WriteString(`local n=redis.call('HGet',KEYS[2],ARGV[1]);local g='{"value":"'..ARGV[1]..'","display":"'..n..'"}';`)
	keys = append(keys, uuid+"_members", GROUPS_REVERSE_LOOKUP_KEY)
	args = append(args, uuid)
	ki := 3
	ai := 2
	for i, v := range ids {
		members.WriteString(fmt.Sprintf(",ARGV[%v],ARGV[%v]", ai, ai+1))
		args = append(args, v, values[i])
		groups.WriteString(fmt.Sprintf("redis.call('HSet',KEYS[%v],ARGV[1],g);", ki))
		keys = append(keys, v+"_groups")
		ai += 2
		ki += 1
	}

	members.WriteString(");")
	groups.WriteString("return 1;")
	cmd := fmt.Sprintf("%v%v", members.String(), groups.String())
	if err := rdb.Eval(ctx, cmd, keys, args).Err(); err != nil {
		log.Printf("Redis Error AddGroupMembers: %v\nerr: %v\n\n", uuid, err.Error())
		return err
	}
	return nil
}

func RemoveGroupMembers(uuid string, member string) error {
	cmd := "redis.call('HDel',KEYS[1],ARGV[1]);redis.call('HDel',KEYS[2],ARGV[2]);return 1;"
	if err := rdb.Eval(ctx, cmd, []string{uuid + "_members", member + "_groups"}, []string{member, uuid}).Err(); err != nil {
		log.Printf("Redis Error RemoveGroupMembers: %v\nerr: %v\n\n", uuid, err.Error())
		return err
	}
	return nil
}

func ReplaceGroupMembers(uuid string, ids, members []string) error {
	var keys, args []string
	var ai, ki int
	var mems, grps strings.Builder
	mems.WriteString(`local j=redis.call('HKeys',KEYS[1]);for k,v in ipairs(j) do redis.call('HDel',v.."_groups",ARGV[1]);end;redis.call('Del',KEYS[1]);`)
	grps.WriteString(`local n=redis.call('HGet',KEYS[2],ARGV[1]);local g='{"value":"'..ARGV[1]..'","display":"'..n..'"}';`)
	keys = append(keys, uuid+"_members", GROUPS_REVERSE_LOOKUP_KEY)
	args = append(args, uuid)
	ki = 3
	ai = 2

	if len(ids) > 0 {
		mems.WriteString("redis.call('HSet',KEYS[1]")
		for i, v := range members {
			mems.WriteString(fmt.Sprintf(",ARGV[%v],ARGV[%v]", ai, ai+1))
			grps.WriteString(fmt.Sprintf(`redis.call('HSet',KEYS[%v],ARGV[1],g);`, ki))
			args = append(args, ids[i], v)
			keys = append(keys, ids[i]+"_groups")
			ai += 2
			ki += 1
		}
		mems.WriteString(");")
	}
	grps.WriteString("return 1;")
	err := rdb.Eval(ctx, fmt.Sprintf("%v%v", mems.String(), grps.String()), keys, args).Err()
	if err != nil {
		log.Printf("Redis Error ReplaceGroupMembers: %v\nerr: %v\n\n", uuid, err.Error())
		return err
	}
	return nil
}

func DelGroup(uuid string) error {
	err := rdb.EvalSha(ctx, luaScripts.LuaDeleteGroup, []string{GROUPS_REVERSE_LOOKUP_KEY, uuid + "_members", uuid, GROUPS_LOOKUP_KEY, GROUPS_KEY}, uuid).Err()
	if err != nil {
		log.Printf("Redis Error Deleting Group: %s\nerr: %v\n\n", uuid, err)
		return err
	}
	return nil
}

func UpdateGroupName(uuid string, name string) error {
	// cmd := `local k=redis.call('HGet',KEYS[1],ARGV[1]);redis.call('HDel',KEYS[2],k);redis.call('HSet',KEYS[1],ARGV[1],ARGV[2]);redis.call('HSet',KEYS[2],ARGV[2],ARGV[1]);  local function code (s) return (string.gsub(s, "\\(.)", function (x) return string.format("\\%03d", string.byte(x)) end)); end local function decode (s) return (string.gsub(s, "\\(%d%d%d)", function (d) return "\\" .. string.char(d) end)); end local j=redis.call("Get",KEYS[3]); j=code(j); j = string.gsub(j,"\"displayName\":\".-\"","\"displayName\":\"` + name + `\""); j = decode(j);  redis.call('Set',KEYS[3],j); local g='{"value":"'..ARGV[1]..'","display":"'..ARGV[2]..'"}'; local m=redis.call('HKeys',KEYS[4]);for k,v in ipairs(m) do redis.call('HSet',v.."_groups",ARGV[1],g);end; return 1;`
	// cmd := `local k=redis.call('HGet',KEYS[1],ARGV[1]);redis.call('HDel',KEYS[2],k);redis.call('HSet',KEYS[1],ARGV[1],ARGV[2]);redis.call('HSet',KEYS[2],ARGV[2],ARGV[1]);  local j=redis.call("Get",KEYS[3]); j = string.gsub(j,"\"displayName\":\".-\"","\"displayName\":\"` + name + `\"");   redis.call('Set',KEYS[3],j); local g='{"value":"'..ARGV[1]..'","display":"'..ARGV[2]..'"}'; local m=redis.call('HKeys',KEYS[4]);for k,v in ipairs(m) do redis.call('HSet',v.."_groups",ARGV[1],g);end; return 1;`

	err := rdb.EvalSha(ctx, luaScripts.LuaUpdateGroupName, []string{GROUPS_REVERSE_LOOKUP_KEY, GROUPS_LOOKUP_KEY, uuid, uuid + "_members"}, uuid, name).Err()
	if err != nil {
		log.Printf("Redis Error UpdateGroupName for UUID: %s\nerr: %v\n\n", uuid, err)
		return err
	}
	return nil
}

func Test(user []byte) {

	// cmd := "redis.call('set', KEYS[1], ARGV[1]); redis.call('HSet', KEYS[2], ARGV[2], ARGV[3]); redis.call('LPush', KEYS[3], ARGV[3]); return 1;"
	// if err := rdb.Eval(ctx, cmd, []string{"8888", USERS_LOOKUP_KEY, USERS_KEY}, user, "test@mail.com", "8888").Err(); err != nil {
	// 	log.Fatalln(err)
	// }

	cmd := `redis.call('set', "k1", 'mm\\\"m'); return 1;`
	if err := rdb.Eval(ctx, cmd, []string{}).Err(); err != nil {
		fmt.Println(err.Error())
	}
}

func Test2() {
	// cmd := "local myTable={}; local l1=redis.call('hgetall',KEYS[1]); local l2=redis.call('hgetall',KEYS[2]); local l3=redis.call('hgetall',KEYS[3]); myTable[1]='__USER__L1'; myTable[2]=l1; myTable[3]='__USER__L2'; myTable[4]=l2; myTable[5]='__USER__L3'; myTable[6]=l3; return myTable"
	cmd := "local myTable={}; myTable[1]=redis.call('hgetall',KEYS[1]); myTable[2]=redis.call('hgetall',KEYS[2]); myTable[3]=redis.call('hgetall',KEYS[3]); return myTable"

	keys := []string{"k1", "k2", "non"}
	result, err := rdb.Eval(ctx, cmd, keys).Result()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Result: %v\n", result)
	fmt.Printf("result: %T\n", result)

	fmt.Printf("result: %T\n", (result.([]interface{})[0]).([]interface{})[0])
	for i, v := range result.([]interface{}) {
		a := v.([]interface{})
		if len(a) == 0 {
			fmt.Printf("%v  -- EMPTY\n", keys[i])
			continue
		}
		for ii, vv := range a {
			fmt.Printf("%v  --  i: %v, v: %v\n", keys[i], ii, vv)
		}
	}
}
