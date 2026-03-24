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
	return "cache:ranking:resource:" + rankType
}

func HotKeywordCacheKey(period string) string {
	return "search:hot:" + period
}

func ReadRankingIDs(key string, page, size int, isIncreased bool) ([]int64, []float64, int64, error) {
	total, err := utils.RDB.ZCard(utils.Ctx, key).Result()
	if err != nil {
		return nil, nil, 0, err
	}
	if total == 0 {
		return nil, nil, 0, nil
	}

	start := int64((page - 1) * size)
	stop := start + int64(size) - 1

	var zs []redis.Z
	if isIncreased {
		zs, err = utils.RDB.ZRangeWithScores(utils.Ctx, key, start, stop).Result()
	} else {
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
