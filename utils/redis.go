package utils

import (
	"context"
	"errors"
	"fmt"
	"log"

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
)

type UserPatch struct {
	Active        bool
	ActiveValue   bool
	Password      bool
	PasswordValue string
}

type LuaScriptSHA struct {
	LuaGetByRange                  string
	LuaGetByFilter                 string
	LuaGetByUUID                   string
	LuaAddUser                     string
	LuaUpdateUuserActive           string
	LuaUpdateUuserInActive         string
	LuaDeleteUser                  string
	LuaPatchUser                   string
	LuaDeleteGroup                 string
	LuaUpdateGroupName             string
	LuaAddGroup                    string
	LuaUpdateGroup                 string
	LuaPatchGroupAddMembers        string
	LuaPatchGroupRemoveMember      string
	LuaPatchGroupReplaceAllMembers string
}

var rdb *redis.Client
var ctx context.Context = context.Background()
var luaScripts LuaScriptSHA

func InitializeRedis(config *Configuration) error {
	rdb = redis.NewClient(&redis.Options{
		// Addr:     "localhost:6379",
		Addr:     config.Redis.Address,
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
	luaScripts.LuaAddGroup = loadLuaScriptIntoCache(LUA_ADD_GROUP)
	luaScripts.LuaUpdateGroup = loadLuaScriptIntoCache(LUA_UPDATE_GROUP)
	luaScripts.LuaPatchGroupAddMembers = loadLuaScriptIntoCache(LUA_PATCH_GROUP_ADD_MEMBER)
	luaScripts.LuaPatchGroupRemoveMember = loadLuaScriptIntoCache(LUA_PATCH_GROUP_REMOVE_MEMBER)
	luaScripts.LuaPatchGroupReplaceAllMembers = loadLuaScriptIntoCache(LUA_PATCH_GROUP_REPLACE_ALL_MEMBERS)

	return nil
}

func loadLuaScriptIntoCache(script string) string {
	result := rdb.ScriptLoad(ctx, script)
	if result.Err() != nil {
		log.Fatalf("Error loading Lua Script into redis..\nscript: %v\nerror: %v\n\n", script, result.Err().Error())
	}

	return result.Val()
}

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

func getByRange(startIndex int, count int, key, embedded_key string) (interface{}, error) {
	keys := []string{key, embedded_key}
	args := []string{fmt.Sprintf("%v", startIndex-1), fmt.Sprintf("%v", startIndex-1+count-1)}
	result, err := rdb.EvalSha(ctx, luaScripts.LuaGetByRange, keys, args).Result()
	if err != nil {
		log.Printf("Redis Error Getting %v by Range, startIndex: %v, count: %v\nerr: %v\n\n", key, startIndex, count, err)
		return nil, err
	}

	return result, nil
}

func getByFilter(name, key, embedded_key string) (interface{}, error) {
	result, err := rdb.EvalSha(ctx, luaScripts.LuaGetByFilter, []string{key, embedded_key}, name).Result()
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

func FlushDB() error {
	status := rdb.FlushDB(ctx)
	if status.Err() != nil {
		return status.Err()
	}

	return nil
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
	keys := []string{uuid, USERS_LOOKUP_KEY, USERS_KEY, USERS_REVERSE_LOOKUP_KEY}
	args := []string{string(doc), userName, uuid}
	if err := rdb.EvalSha(ctx, luaScripts.LuaAddUser, keys, args).Err(); err != nil {
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
	keys := []string{uuid, USERS_LOOKUP_KEY, USERS_KEY, USERS_REVERSE_LOOKUP_KEY}
	if err := rdb.EvalSha(ctx, luaScripts.LuaDeleteUser, keys, uuid).Err(); err != nil {
		if err != redis.Nil {
			log.Printf("Redis Error Deleting User: %s\nerr: %v\n\n", uuid, err)
			return err
		}
		return errors.New("not_found")
	}
	return nil
}

func PatchUser(uuid string, userPatch UserPatch) error {
	keys := []string{uuid, fmt.Sprintf("%v", userPatch.Active), fmt.Sprintf("%v", userPatch.Password)}
	args := []string{fmt.Sprintf("%v", userPatch.ActiveValue), userPatch.PasswordValue}
	if err := rdb.EvalSha(ctx, luaScripts.LuaPatchUser, keys, args).Err(); err != nil {
		if err != redis.Nil {
			log.Printf("Redis Error Patching User: %s\nerr: %v\n\n", uuid, err)
			return err
		}
		return errors.New("not_found")
	}
	return nil
}

func GetUserCount() (int64, error) {
	intCmd := rdb.LLen(ctx, USERS_KEY)
	if intCmd.Err() != nil {
		return 0, intCmd.Err()
	}

	return intCmd.Val(), nil
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

func AddGroup(doc []byte, groupName, uuid, groupSnippet string, members, ids []string) error {
	keys := []string{uuid, GROUPS_LOOKUP_KEY, GROUPS_REVERSE_LOOKUP_KEY, GROUPS_KEY, fmt.Sprintf("%v_members", uuid)}
	args := []string{string(doc), groupName, uuid, groupSnippet, fmt.Sprintf("%v", len(ids))}
	i := 0
	for _, v := range ids {
		keys = append(keys, fmt.Sprintf("%v_groups", v))
		args = append(args, v)
		args = append(args, members[i])
		i = i + 1
	}

	if err := rdb.EvalSha(ctx, luaScripts.LuaAddGroup, keys, args).Err(); err != nil {
		if err != redis.Nil {
			log.Printf("Redis Error Adding Group with Members: %s\nerr: %v\n\n", doc, err)
			return err
		}
		return errors.New("group_already_exists")
	}
	return nil
}

func UpdateGroup(doc []byte, groupName, uuid, groupSnippet string, members, ids []string) error {
	keys := []string{uuid, GROUPS_LOOKUP_KEY, GROUPS_REVERSE_LOOKUP_KEY, GROUPS_KEY, fmt.Sprintf("%v_members", uuid)}
	args := []string{string(doc), groupName, uuid, groupSnippet, fmt.Sprintf("%v", len(ids))}
	i := 0
	for _, v := range ids {
		keys = append(keys, fmt.Sprintf("%v_groups", v))
		args = append(args, v)
		args = append(args, members[i])
		i = i + 1
	}

	if err := rdb.EvalSha(ctx, luaScripts.LuaUpdateGroup, keys, args).Err(); err != nil {
		log.Printf("Redis Error Updating Group with Members: %s\nerr: %v\n\n", doc, err)
		return err
	}
	return nil
}

func AddGroupMembers(uuid string, ids, values []string) error {
	keys := []string{GROUPS_REVERSE_LOOKUP_KEY, fmt.Sprintf("%v_members", uuid)}
	args := []string{uuid, fmt.Sprintf("%v", len(ids))}
	keys = append(keys, ids...)
	args = append(args, values...)
	if err := rdb.EvalSha(ctx, luaScripts.LuaPatchGroupAddMembers, keys, args).Err(); err != nil {
		log.Printf("Redis Error AddGroupMembers: %v\nerr: %v\n\n", uuid, err.Error())
		return err
	}

	return nil
}

func RemoveGroupMembers(uuid, member string) error {
	keys := []string{uuid + "_members", member + "_groups"}
	args := []string{member, uuid}
	if err := rdb.EvalSha(ctx, luaScripts.LuaPatchGroupRemoveMember, keys, args).Err(); err != nil {
		log.Printf("Redis Error RemoveGroupMembers: %v\nerr: %v\n\n", uuid, err.Error())
		return err
	}
	return nil
}

func ReplaceGroupMembers(uuid string, ids, members []string) error {
	keys := []string{uuid + "_members", GROUPS_REVERSE_LOOKUP_KEY}
	args := []string{uuid, fmt.Sprintf("%v", len(ids))}
	for i, v := range members {
		args = append(args, ids[i], v)
		keys = append(keys, ids[i]+"_groups")
	}

	err := rdb.EvalSha(ctx, luaScripts.LuaPatchGroupReplaceAllMembers, keys, args).Err()
	if err != nil {
		log.Printf("Redis Error ReplaceGroupMembers: %v\nerr: %v\n\n", uuid, err.Error())
		return err
	}
	return nil
}

func DelGroup(uuid string) error {
	keys := []string{GROUPS_REVERSE_LOOKUP_KEY, uuid + "_members", uuid, GROUPS_LOOKUP_KEY, GROUPS_KEY}
	err := rdb.EvalSha(ctx, luaScripts.LuaDeleteGroup, keys, uuid).Err()
	if err != nil {
		log.Printf("Redis Error Deleting Group: %s\nerr: %v\n\n", uuid, err)
		return err
	}
	return nil
}

func UpdateGroupName(uuid string, name string) error {
	keys := []string{GROUPS_REVERSE_LOOKUP_KEY, GROUPS_LOOKUP_KEY, uuid, uuid + "_members"}
	args := []string{uuid, name}
	err := rdb.EvalSha(ctx, luaScripts.LuaUpdateGroupName, keys, args).Err()
	if err != nil {
		log.Printf("Redis Error UpdateGroupName for UUID: %s\nerr: %v\n\n", uuid, err)
		return err
	}
	return nil
}

func GetGroupCount() (int64, error) {
	intCmd := rdb.LLen(ctx, GROUPS_KEY)
	if intCmd.Err() != nil {
		return 0, intCmd.Err()
	}

	return intCmd.Val(), nil
}

// func Test(user []byte) {

// 	// cmd := "redis.call('set', KEYS[1], ARGV[1]); redis.call('HSet', KEYS[2], ARGV[2], ARGV[3]); redis.call('LPush', KEYS[3], ARGV[3]); return 1;"
// 	// if err := rdb.Eval(ctx, cmd, []string{"8888", USERS_LOOKUP_KEY, USERS_KEY}, user, "test@mail.com", "8888").Err(); err != nil {
// 	// 	log.Fatalln(err)
// 	// }

// 	cmd := `redis.call('set', "k1", 'mm\\\"m'); return 1;`
// 	if err := rdb.Eval(ctx, cmd, []string{}).Err(); err != nil {
// 		fmt.Println(err.Error())
// 	}
// }

// func Test2() {
// 	// cmd := "local myTable={}; local l1=redis.call('hgetall',KEYS[1]); local l2=redis.call('hgetall',KEYS[2]); local l3=redis.call('hgetall',KEYS[3]); myTable[1]='__USER__L1'; myTable[2]=l1; myTable[3]='__USER__L2'; myTable[4]=l2; myTable[5]='__USER__L3'; myTable[6]=l3; return myTable"
// 	cmd := "local myTable={}; myTable[1]=redis.call('hgetall',KEYS[1]); myTable[2]=redis.call('hgetall',KEYS[2]); myTable[3]=redis.call('hgetall',KEYS[3]); return myTable"

// 	keys := []string{"k1", "k2", "non"}
// 	result, err := rdb.Eval(ctx, cmd, keys).Result()
// 	if err != nil {
// 		log.Fatal(err)
// 	}
// 	fmt.Printf("Result: %v\n", result)
// 	fmt.Printf("result: %T\n", result)

// 	fmt.Printf("result: %T\n", (result.([]interface{})[0]).([]interface{})[0])
// 	for i, v := range result.([]interface{}) {
// 		a := v.([]interface{})
// 		if len(a) == 0 {
// 			fmt.Printf("%v  -- EMPTY\n", keys[i])
// 			continue
// 		}
// 		for ii, vv := range a {
// 			fmt.Printf("%v  --  i: %v, v: %v\n", keys[i], ii, vv)
// 		}
// 	}
// }
