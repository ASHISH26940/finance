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

	opts, err := redis.ParseURL(config.RedisUrl)
	if err != nil {
		return err
	}

	if strings.TrimSpace(config.RedisPass) != "" {
		opts.Password = config.RedisPass
	}

	opts.ReadTimeout = 0
	opts.WriteTimeout = 0
	opts.DialTimeout = 0
	opts.PoolSize = 200
	opts.MinIdleConns = 50

	rdb := redis.NewClient(opts)

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
