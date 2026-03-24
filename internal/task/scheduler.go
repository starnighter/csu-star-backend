package task

import (
	"context"
	"csu-star-backend/internal/repo"
	"csu-star-backend/logger"
	"csu-star-backend/pkg/utils"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

const rankingCacheTTL = 2 * time.Hour

type Scheduler struct {
	db            *gorm.DB
	aggregateRepo repo.AggregateRepository
}

func NewScheduler(db *gorm.DB, ar repo.AggregateRepository) *Scheduler {
	return &Scheduler{
		db:            db,
		aggregateRepo: ar,
	}
}

func (s *Scheduler) Start(ctx context.Context) {
	s.runDailyMaintenance()
	s.runRefresh()

	go s.runTicker(ctx, time.Hour, s.runRefresh)
	go s.runTicker(ctx, 6*time.Hour, s.runDailyMaintenance)
}

func (s *Scheduler) runTicker(ctx context.Context, interval time.Duration, fn func()) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			fn()
		}
	}
}

func (s *Scheduler) runRefresh() {
	if err := s.aggregateRepo.RefreshTeacherStats(); err != nil {
		logger.Log.Error("刷新教师统计失败", zap.Error(err))
	}
	if err := s.aggregateRepo.RefreshCourseStats(); err != nil {
		logger.Log.Error("刷新课程统计失败", zap.Error(err))
	}
	if err := s.aggregateRepo.RefreshResourceStats(); err != nil {
		logger.Log.Error("刷新资源统计失败", zap.Error(err))
	}

	now := time.Now()
	weekAgo := now.Add(-7 * 24 * time.Hour)
	monthAgo := now.Add(-30 * 24 * time.Hour)
	for _, item := range []struct {
		period string
		since  *time.Time
	}{
		{period: "all", since: nil},
		{period: "week", since: &weekAgo},
		{period: "month", since: &monthAgo},
	} {
		if err := s.aggregateRepo.RefreshTeacherRankings(item.period, item.since); err != nil {
			logger.Log.Error("刷新教师排行榜失败", zap.String("period", item.period), zap.Error(err))
		}
		if err := s.aggregateRepo.RefreshCourseRankings(item.period, item.since); err != nil {
			logger.Log.Error("刷新课程排行榜失败", zap.String("period", item.period), zap.Error(err))
		}
	}

	if err := s.syncTeacherRankingRedis(); err != nil {
		logger.Log.Error("同步教师排行榜缓存失败", zap.Error(err))
	}
	if err := s.syncCourseRankingRedis(); err != nil {
		logger.Log.Error("同步课程排行榜缓存失败", zap.Error(err))
	}
	if err := s.syncResourceRankingRedis(); err != nil {
		logger.Log.Error("同步资源排行榜缓存失败", zap.Error(err))
	}

	s.syncHotKeywords()
}

func (s *Scheduler) runDailyMaintenance() {
	if err := s.aggregateRepo.TrimSearchHistories(20); err != nil {
		logger.Log.Error("清理搜索历史失败", zap.Error(err))
	}
	s.rotateHotKeywordWindows()
}

func (s *Scheduler) syncTeacherRankingRedis() error {
	type row struct {
		TeacherID int64
		Period    string
		Dimension string
		Score     float64
		DeptID    int16
	}

	var items []row
	if err := s.db.Table("teacher_rankings").
		Select("teacher_id, period, dimension, score, department_id AS dept_id").
		Scan(&items).Error; err != nil {
		return err
	}

	if err := deleteKeysByPattern("cache:ranking:teacher:*"); err != nil {
		return err
	}

	keys := make(map[string]struct{})
	pipe := utils.RDB.Pipeline()
	for _, item := range items {
		member := strconv.FormatInt(item.TeacherID, 10)
		keyAll := TeacherRankingCacheKey(item.Dimension, item.Period, 0)
		keyDept := TeacherRankingCacheKey(item.Dimension, item.Period, item.DeptID)
		keys[keyAll] = struct{}{}
		keys[keyDept] = struct{}{}
		pipe.ZAdd(utils.Ctx, keyAll, redis.Z{Score: item.Score, Member: member})
		pipe.ZAdd(utils.Ctx, keyDept, redis.Z{Score: item.Score, Member: member})
	}
	if _, err := pipe.Exec(utils.Ctx); err != nil {
		return err
	}
	return expireRedisKeys(keys)
}

func (s *Scheduler) syncCourseRankingRedis() error {
	type row struct {
		CourseID  int64
		Period    string
		Dimension string
		Score     float64
	}

	var items []row
	if err := s.db.Table("course_rankings").
		Select("course_id, period, dimension, score").
		Scan(&items).Error; err != nil {
		return err
	}

	if err := deleteKeysByPattern("cache:ranking:course:*"); err != nil {
		return err
	}

	keys := make(map[string]struct{})
	pipe := utils.RDB.Pipeline()
	for _, item := range items {
		key := CourseRankingCacheKey(item.Dimension, item.Period)
		keys[key] = struct{}{}
		pipe.ZAdd(utils.Ctx, key, redis.Z{Score: item.Score, Member: strconv.FormatInt(item.CourseID, 10)})
	}
	if _, err := pipe.Exec(utils.Ctx); err != nil {
		return err
	}
	return expireRedisKeys(keys)
}

func (s *Scheduler) syncResourceRankingRedis() error {
	type row struct {
		ID        int64
		HotScore  float64
		Downloads float64
		Likes     float64
		Comments  float64
		Views     float64
	}

	var items []row
	if err := s.db.Table("resources").
		Where("status = ?", "approved").
		Select(`
			id,
			(download_count + like_count + comment_count) AS hot_score,
			download_count AS downloads,
			like_count AS likes,
			comment_count AS comments,
			view_count AS views`).
		Scan(&items).Error; err != nil {
		return err
	}

	if err := deleteKeysByPattern("cache:ranking:resource:*"); err != nil {
		return err
	}

	keys := map[string]struct{}{
		ResourceRankingCacheKey("hot_score"): {},
		ResourceRankingCacheKey("downloads"): {},
		ResourceRankingCacheKey("likes"):     {},
		ResourceRankingCacheKey("comments"):  {},
		ResourceRankingCacheKey("views"):     {},
	}
	pipe := utils.RDB.Pipeline()
	for _, item := range items {
		member := strconv.FormatInt(item.ID, 10)
		for key, score := range map[string]float64{
			ResourceRankingCacheKey("hot_score"): item.HotScore,
			ResourceRankingCacheKey("downloads"): item.Downloads,
			ResourceRankingCacheKey("likes"):     item.Likes,
			ResourceRankingCacheKey("comments"):  item.Comments,
			ResourceRankingCacheKey("views"):     item.Views,
		} {
			pipe.ZAdd(utils.Ctx, key, redis.Z{Score: score, Member: member})
		}
	}
	if _, err := pipe.Exec(utils.Ctx); err != nil {
		return err
	}
	return expireRedisKeys(keys)
}

func (s *Scheduler) syncHotKeywords() {
	for _, period := range []string{"day", "week", "month"} {
		zs, err := utils.RDB.ZRevRangeWithScores(utils.Ctx, HotKeywordCacheKey(period), 0, 99).Result()
		if err != nil {
			logger.Log.Error("读取热词缓存失败", zap.String("period", period), zap.Error(err))
			continue
		}

		keywords := make(map[string]float64, len(zs))
		for _, item := range zs {
			keyword, ok := item.Member.(string)
			if !ok || keyword == "" {
				continue
			}
			keywords[keyword] = item.Score
		}

		if err := s.aggregateRepo.SyncHotKeywords(period, keywords); err != nil {
			logger.Log.Error("持久化热词失败", zap.String("period", period), zap.Error(err))
		}
	}
}

func (s *Scheduler) rotateHotKeywordWindows() {
	now := time.Now()
	dayMarker := now.Format("2006-01-02")
	year, week := now.ISOWeek()
	weekMarker := fmt.Sprintf("%d-%02d", year, week)
	monthMarker := now.Format("2006-01")

	s.resetHotKeywordWindow("day", dayMarker)
	s.resetHotKeywordWindow("week", weekMarker)
	s.resetHotKeywordWindow("month", monthMarker)
}

func (s *Scheduler) resetHotKeywordWindow(period, marker string) {
	markerKey := "cache:search:hot:marker:" + period
	lastMarker, err := utils.RDB.Get(utils.Ctx, markerKey).Result()
	if err != nil && !errors.Is(err, redis.Nil) {
		logger.Log.Error("读取热词窗口标记失败", zap.String("period", period), zap.Error(err))
		return
	}
	if errors.Is(err, redis.Nil) {
		if setErr := utils.RDB.Set(utils.Ctx, markerKey, marker, 45*24*time.Hour).Err(); setErr != nil {
			logger.Log.Error("初始化热词窗口标记失败", zap.String("period", period), zap.Error(setErr))
		}
		return
	}
	if lastMarker == marker {
		return
	}

	pipe := utils.RDB.TxPipeline()
	pipe.Del(utils.Ctx, HotKeywordCacheKey(period))
	pipe.Set(utils.Ctx, markerKey, marker, 45*24*time.Hour)
	if _, err := pipe.Exec(utils.Ctx); err != nil {
		logger.Log.Error("重置热词窗口失败", zap.String("period", period), zap.Error(err))
	}
}

func expireRedisKeys(keys map[string]struct{}) error {
	if len(keys) == 0 {
		return nil
	}
	pipe := utils.RDB.Pipeline()
	for key := range keys {
		pipe.Expire(utils.Ctx, key, rankingCacheTTL)
	}
	_, err := pipe.Exec(utils.Ctx)
	return err
}

func deleteKeysByPattern(pattern string) error {
	var cursor uint64
	for {
		keys, nextCursor, err := utils.RDB.Scan(utils.Ctx, cursor, pattern, 100).Result()
		if err != nil {
			return err
		}
		if len(keys) > 0 {
			if err := utils.RDB.Del(utils.Ctx, keys...).Err(); err != nil {
				return err
			}
		}
		cursor = nextCursor
		if cursor == 0 {
			return nil
		}
	}
}
