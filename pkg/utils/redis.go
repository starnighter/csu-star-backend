package utils

import (
	"context"
	"csu-star-backend/config"
	"log"

	"github.com/redis/go-redis/v9"
)

var (
	RDB *redis.Client
	Ctx = context.Background()
)

func InitRedis() {
	RDB = redis.NewClient(&redis.Options{
		Addr:     config.GlobalConfig.Redis.Addr,
		Password: config.GlobalConfig.Redis.Password,
		DB:       config.GlobalConfig.Redis.DB,
	})

	if err := RDB.Ping(Ctx).Err(); err != nil {
		log.Fatalf("Redis连接失败：%v\n", err)
	}
}
