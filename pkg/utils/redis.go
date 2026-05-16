package utils

import (
	"context"
	"csu-star-backend/config"
	"fmt"
	"log"

	"github.com/redis/go-redis/v9"
)

var (
	RDB *redis.Client
	Ctx = context.Background()
)

func InitRedis() {
	poolSize := config.GetConfig().Redis.PoolSize
	if poolSize <= 0 {
		poolSize = 50
	}
	minIdle := config.GetConfig().Redis.MinIdleConns
	if minIdle <= 0 {
		minIdle = 10
	}

	RDB = redis.NewClient(&redis.Options{
		Username:     config.GetConfig().Redis.Username,
		Addr:         config.GetConfig().Redis.Addr,
		Password:     config.GetConfig().Redis.Password,
		DB:           config.GetConfig().Redis.DB,
		PoolSize:     poolSize,
		MinIdleConns: minIdle,
	})

	if err := RDB.Ping(Ctx).Err(); err != nil {
		log.Fatalf("Redis连接失败：%v\n", err)
	}
}

// DeleteKeysByPattern 使用 SCAN 批量删除匹配 pattern 的 key
func DeleteKeysByPattern(pattern string) error {
	var cursor uint64
	for {
		keys, nextCursor, err := RDB.Scan(Ctx, cursor, pattern, 100).Result()
		if err != nil {
			return err
		}
		if len(keys) > 0 {
			if err := RDB.Del(Ctx, keys...).Err(); err != nil {
				return err
			}
		}
		cursor = nextCursor
		if cursor == 0 {
			return nil
		}
	}
}

func InvalidateCourseCache(courseID int64) {
	RDB.Del(Ctx, fmt.Sprintf("cache:course:detail:%d", courseID))
	DeleteKeysByPattern("cache:course:list:*")
}

func InvalidateTeacherCache(teacherID int64) {
	RDB.Del(Ctx, fmt.Sprintf("cache:teacher:detail:%d", teacherID))
	DeleteKeysByPattern("cache:teacher:list:*")
}
