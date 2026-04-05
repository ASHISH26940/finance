package database

import (
	"finance/config"
	"log"
	"strings"

	"github.com/go-redis/redis"
)

type RedisDbInstance struct {
	Db *redis.Client
}

var RedisDb RedisDbInstance

func RedisConnectDb(config *config.Configuration) error {
	if config == nil || strings.TrimSpace(config.RedisUrl) == "" {
		return nil
	}

	rdb := redis.NewClient(&redis.Options{
		Addr:         config.RedisUrl,
		Password:     config.RedisPass, // no password set
		DB:           0,                // use default DB
		ReadTimeout:  0,
		WriteTimeout: 0,
		DialTimeout:  0,
		PoolSize:     200,
		MinIdleConns: 50,
	})

	pong, err := rdb.Ping().Result()
	if err != nil {
		return err
	}

	log.Printf("connection to Redis established '%s'", pong)
	RedisDb = RedisDbInstance{Db: rdb}
	return nil
}

func RedisAvailable() bool {
	return RedisDb.Db != nil
}
