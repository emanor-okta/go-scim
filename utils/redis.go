package utils

import (
	"context"
	"errors"
	"log"

	"github.com/go-redis/redis/v8"
)

// https://redis.io/commands
// docker run --name go-scim-redis -p 6379:6379 -d redis redis-server --save 60 1 --loglevel warning
// docker exec -i go-scim-redis bash (cd /usr/local/bin)

const (
	USERS_KEY    = "_users"
	GROUPS_KEY   = "_groups"
	LOOKUP_KEY   = "_lookup"
	ADD_USER_CMD = "redis.call('set', KEYS[1], ARGV[1]); redis.call('HSet', KEYS[2], ARGV[2], ARGV[3]); redis.call('LPush', KEYS[3], ARGV[3]); return 1;"
	DEL_USER_CMD = "redis.call('del', KEYS[1]); redis.call('HDel', KEYS[2], ARGV[1]); redis.call('LRem', KEYS[3], 0, ARGV[2]); return 1;"
)

var rdb *redis.Client
var ctx context.Context = context.Background()

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

	return nil
}

func SaveDoc(key string, doc interface{}) error {
	err := rdb.Set(ctx, key, doc, 0).Err()
	if err != nil {
		log.Printf("Redis Error Saving, Key: %v, error: %v\n\n", key, err)
		return err
	}
	return nil
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

func GetDoc(key string) (string, error) {
	doc, err := rdb.Get(ctx, key).Result()
	if err != nil {
		if err != redis.Nil {
			log.Printf("Redis Error Getting, Key: %v, error: %v\n\n", key, err)
			return "", err
		}
		return "", errors.New("not_found")
	}
	return doc, nil
}

func AddLookup(user, uuid string) error {
	if err := rdb.HSet(ctx, LOOKUP_KEY, user, uuid).Err(); err != nil {
		log.Printf("Redis Error Adding Lookup, User: %v, UUID: %v, error: %v\n\n", user, uuid, err)
		return err
	}
	return nil
}

func ListPush(key, value string) error {

	if err := rdb.LPush(ctx, key, value).Err(); err != nil {
		log.Printf("Redis Error Pushing to List: %s for: %v, err: %v\n\n", key, value, err)
		return err
	}
	return nil
}

func lRange(key string, startIndex, count int) ([]string, error) {
	docs, err := rdb.LRange(ctx, key, int64(startIndex-1), int64(startIndex+count-2)).Result()
	if err != nil {
		log.Printf("Redis Error LRange: %s, err: %v\n\n", key, err)
		return nil, err
	}
	return docs, nil
}

func GetUsers(startIndex int, count int) ([]interface{}, error) {
	ids, err := lRange(USERS_KEY, startIndex, count)
	if err != nil {
		return nil, err
	}
	if len(ids) == 0 {
		return nil, errors.New("not_found")
	}
	doc, err := rdb.MGet(ctx, ids...).Result()
	if err != nil {
		log.Printf("Redis Error Getting, MKey: %v, error: %v\n\n", ids, err)
		return nil, err
	}
	return doc, nil
}

func GetUserByFilter(user string) (string, error) {
	uuid, err := rdb.HGet(ctx, LOOKUP_KEY, user).Result()
	if err != nil {
		if err != redis.Nil {
			log.Printf("Redis Error Getting Lookup: %v, error: %v\n\n", user, err)
			return "", err
		}
		return "", errors.New("not_found")
	}
	return GetDoc(uuid)
}

func AddUser(doc []byte, userName, uuid string) error {
	if err := rdb.Eval(ctx, ADD_USER_CMD, []string{uuid, LOOKUP_KEY, USERS_KEY}, doc, userName, uuid).Err(); err != nil {
		log.Printf("Redis Error Adding New User: %s\nerr: %v\n\n", doc, err)
		return err
	}
	return nil
}

func DelUser(userName, uuid string) error {
	if err := rdb.Eval(ctx, DEL_USER_CMD, []string{uuid, LOOKUP_KEY, USERS_KEY}, uuid, userName).Err(); err != nil {
		log.Printf("Redis Error Deleting User: %s\nerr: %v\n\n", userName, err)
		return err
	}
	return nil
}

func Test(user []byte) {

	cmd := "redis.call('set', KEYS[1], ARGV[1]); redis.call('HSet', KEYS[2], ARGV[2], ARGV[3]); redis.call('LPush', KEYS[3], ARGV[3]); return 1;"
	if err := rdb.Eval(ctx, cmd, []string{"8888", LOOKUP_KEY, USERS_KEY}, user, "test@mail.com", "8888").Err(); err != nil {
		log.Fatalln(err)
	}
}
