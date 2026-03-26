package task

import (
	"context"
	"csu-star-backend/internal/repo"
	"csu-star-backend/logger"
	"csu-star-backend/pkg/utils"
	"errors"
	"fmt"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

const rankingCacheTTL = 2 * time.Hour
const initialRefreshDelay = 15 * time.Second

type Scheduler struct {
	db            *gorm.DB
	aggregateRepo repo.AggregateRepository
	refreshing    atomic.Bool
	maintaining   atomic.Bool
}

func NewScheduler(db *gorm.DB, ar repo.AggregateRepository) *Scheduler {
	return &Scheduler{
		db:            db,
		aggregateRepo: ar,
	}
}

func (s *Scheduler) Start(ctx context.Context) {
	go s.runInitialTasks(ctx)

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

func (s *Scheduler) runInitialTasks(ctx context.Context) {
	// Delay heavy aggregate refreshes so API startup is not blocked by bulk writes.
	timer := time.NewTimer(initialRefreshDelay)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return
	case <-timer.C:
	}

	s.runDailyMaintenance()
	s.runRefresh()
}

func (s *Scheduler) runRefresh() {
	if !s.refreshing.CompareAndSwap(false, true) {
		logger.Log.Info("跳过聚合刷新：上一次刷新仍在执行")
		return
	}
	defer s.refreshing.Store(false)

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
	if !s.maintaining.CompareAndSwap(false, true) {
		logger.Log.Info("跳过日常维护：上一次维护仍在执行")
		return
	}
	defer s.maintaining.Store(false)

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

	keys := make(map[string]struct{})
	snapshots := make(map[string]map[string]float64)
	for _, item := range items {
		member := strconv.FormatInt(item.TeacherID, 10)
		keyAll := TeacherRankingCacheKey(item.Dimension, item.Period, 0)
		keyDept := TeacherRankingCacheKey(item.Dimension, item.Period, item.DeptID)
		keys[keyAll] = struct{}{}
		keys[keyDept] = struct{}{}
		addSnapshotMember(snapshots, keyAll, member, item.Score)
		addSnapshotMember(snapshots, keyDept, member, item.Score)
	}
	if err := syncSortedSetSnapshots(snapshots); err != nil {
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

	keys := make(map[string]struct{})
	snapshots := make(map[string]map[string]float64)
	for _, item := range items {
		key := CourseRankingCacheKey(item.Dimension, item.Period)
		keys[key] = struct{}{}
		addSnapshotMember(snapshots, key, strconv.FormatInt(item.CourseID, 10), item.Score)
	}
	if err := syncSortedSetSnapshots(snapshots); err != nil {
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

	keys := map[string]struct{}{
		ResourceRankingCacheKey("hot_score"): {},
		ResourceRankingCacheKey("downloads"): {},
		ResourceRankingCacheKey("likes"):     {},
		ResourceRankingCacheKey("comments"):  {},
		ResourceRankingCacheKey("views"):     {},
	}
	snapshots := make(map[string]map[string]float64, len(keys))
	for _, item := range items {
		member := strconv.FormatInt(item.ID, 10)
		for key, score := range map[string]float64{
			ResourceRankingCacheKey("hot_score"): item.HotScore,
			ResourceRankingCacheKey("downloads"): item.Downloads,
			ResourceRankingCacheKey("likes"):     item.Likes,
			ResourceRankingCacheKey("comments"):  item.Comments,
			ResourceRankingCacheKey("views"):     item.Views,
		} {
			addSnapshotMember(snapshots, key, member, score)
		}
	}
	if err := syncSortedSetSnapshots(snapshots); err != nil {
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

func addSnapshotMember(snapshots map[string]map[string]float64, key, member string, score float64) {
	if snapshots[key] == nil {
		snapshots[key] = make(map[string]float64)
	}
	snapshots[key][member] = score
}

func syncSortedSetSnapshots(snapshots map[string]map[string]float64) error {
	for key, members := range snapshots {
		if err := syncSortedSetSnapshot(key, members); err != nil {
			return err
		}
	}
	return nil
}

func syncSortedSetSnapshot(key string, members map[string]float64) error {
	existingMembers, err := utils.RDB.ZRange(utils.Ctx, key, 0, -1).Result()
	if err != nil && !errors.Is(err, redis.Nil) {
		return err
	}

	existingSet := make(map[string]struct{}, len(existingMembers))
	for _, member := range existingMembers {
		existingSet[member] = struct{}{}
	}

	pipe := utils.RDB.Pipeline()
	if len(members) > 0 {
		zs := make([]redis.Z, 0, len(members))
		for member, score := range members {
			zs = append(zs, redis.Z{Score: score, Member: member})
			delete(existingSet, member)
		}
		pipe.ZAdd(utils.Ctx, key, zs...)
	}

	if len(existingSet) > 0 {
		staleMembers := make([]interface{}, 0, len(existingSet))
		for member := range existingSet {
			staleMembers = append(staleMembers, member)
		}
		pipe.ZRem(utils.Ctx, key, staleMembers...)
	}

	_, err = pipe.Exec(utils.Ctx)
	return err
}
