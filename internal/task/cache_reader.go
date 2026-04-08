package task

import (
	"csu-star-backend/pkg/utils"
	"fmt"
	"strconv"

	"github.com/redis/go-redis/v9"
)

func TeacherRankingCacheKey(rankType, period string, departmentID int16) string {
	return fmt.Sprintf("cache:ranking:teacher:%s:%s:%d", rankType, period, departmentID)
}

func CourseRankingCacheKey(rankType, period string) string {
	return fmt.Sprintf("cache:ranking:course:%s:%s", rankType, period)
}

func ResourceRankingCacheKey(rankType string) string {
	return "cache:ranking:resource_collection:v2:" + rankType
}

func ReadRankingIDs(key string, page, size int, isIncreased bool) ([]int64, []float64, int64, error) {
	start := int64((page - 1) * size)

	var zs []redis.Z
	var total int64
	var err error
	if isIncreased {
		total, err = utils.RDB.ZCount(utils.Ctx, key, "(0", "+inf").Result()
		if err != nil {
			return nil, nil, 0, err
		}
		if total == 0 {
			return nil, nil, 0, nil
		}
		zs, err = utils.RDB.ZRangeByScoreWithScores(utils.Ctx, key, &redis.ZRangeBy{
			Min:    "(0",
			Max:    "+inf",
			Offset: start,
			Count:  int64(size),
		}).Result()
	} else {
		total, err = utils.RDB.ZCard(utils.Ctx, key).Result()
		if err != nil {
			return nil, nil, 0, err
		}
		if total == 0 {
			return nil, nil, 0, nil
		}
		stop := start + int64(size) - 1
		zs, err = utils.RDB.ZRevRangeWithScores(utils.Ctx, key, start, stop).Result()
	}
	if err != nil {
		return nil, nil, 0, err
	}

	ids := make([]int64, 0, len(zs))
	scores := make([]float64, 0, len(zs))
	for _, z := range zs {
		value, ok := z.Member.(string)
		if !ok {
			continue
		}
		id, convErr := strconv.ParseInt(value, 10, 64)
		if convErr != nil {
			continue
		}
		ids = append(ids, id)
		scores = append(scores, z.Score)
	}
	return ids, scores, total, nil
}
