package utils

import (
	"context"
	"csu-star-backend/config"
	"log"

	"github.com/redis/go-redis/v9"
)

var (
	RDB      *redis.Client
	Ctx      = context.Background()
	redisCfg = config.GlobalConfig.Redis
)

func InitRedis() {
	RDB = redis.NewClient(&redis.Options{
		Addr:     redisCfg.Addr,
		Password: redisCfg.Password,
		DB:       redisCfg.DB,
	})

	if err := RDB.Ping(Ctx).Err(); err != nil {
		log.Fatalf("Redis连接失败：%v\n", err)
	}
}
