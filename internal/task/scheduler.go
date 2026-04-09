package task

import (
	"context"
	"csu-star-backend/internal/constant"
	"csu-star-backend/internal/repo"
	"csu-star-backend/logger"
	"csu-star-backend/pkg/utils"
	"encoding/json"
	"errors"
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
	courseRepo    repo.CourseRepository
	teacherRepo   repo.TeacherRepository
	miscRepo      repo.MiscRepository
	refreshing    atomic.Bool
	maintaining   atomic.Bool
	randomizing   atomic.Bool
}

func NewScheduler(db *gorm.DB, ar repo.AggregateRepository, cr repo.CourseRepository, tr repo.TeacherRepository, mr repo.MiscRepository) *Scheduler {
	return &Scheduler{
		db:            db,
		aggregateRepo: ar,
		courseRepo:    cr,
		teacherRepo:   tr,
		miscRepo:      mr,
	}
}

func (s *Scheduler) Start(ctx context.Context) {
	go s.runInitialTasks(ctx)

	go s.runTicker(ctx, time.Hour, s.runRefresh)
	go s.runTicker(ctx, 6*time.Hour, s.runDailyMaintenance)
	go s.runTicker(ctx, 1*time.Hour, s.runRandomCoursesAndTeachers)
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
	s.runRandomCoursesAndTeachers()
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
}

func (s *Scheduler) runDailyMaintenance() {
	if !s.maintaining.CompareAndSwap(false, true) {
		logger.Log.Info("跳过日常维护：上一次维护仍在执行")
		return
	}
	defer s.maintaining.Store(false)

	if err := s.miscRepo.PurgeExpiredNotifications(time.Now()); err != nil {
		logger.Log.Error("清理过期通知失败", zap.Error(err))
	}
}

func (s *Scheduler) runRandomCoursesAndTeachers() {
	if !s.randomizing.CompareAndSwap(false, true) {
		logger.Log.Info("跳过随机课程与教师：上一次随机仍在执行")
		return
	}
	defer s.randomizing.Store(false)

	err := s.SyncRandomCourses()
	if err != nil {
		logger.Log.Error("生成随机课程数据失败", zap.Error(err))
	}
	err = s.SyncRandomTeachers()
	if err != nil {
		logger.Log.Error("生成随机老师数据失败", zap.Error(err))
	}
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
		CourseID      int64
		ResourceCount float64
		Downloads     float64
		Views         float64
		Likes         float64
		FavoriteCount float64
		Comprehensive float64
	}

	var items []row
	if err := s.db.Table("courses").
		Select(`
			courses.id AS course_id,
			COALESCE(courses.resource_count, 0) AS resource_count,
			COALESCE(courses.download_total, 0) AS downloads,
			COALESCE(courses.view_total, 0) AS views,
			COALESCE(courses.like_total, 0) AS likes,
			COALESCE(courses.resource_favorite_count, 0) AS favorite_count,
			(
				COALESCE(courses.resource_count, 0) * 12 +
				COALESCE(courses.download_total, 0) * 8 +
				COALESCE(courses.resource_favorite_count, 0) * 5 +
				COALESCE(courses.like_total, 0)
			) AS comprehensive`).
		Scan(&items).Error; err != nil {
		return err
	}

	keys := map[string]struct{}{
		ResourceRankingCacheKey("comprehensive"):  {},
		ResourceRankingCacheKey("downloads"):      {},
		ResourceRankingCacheKey("views"):          {},
		ResourceRankingCacheKey("likes"):          {},
		ResourceRankingCacheKey("favorite_count"): {},
		ResourceRankingCacheKey("resource_count"): {},
	}
	snapshots := make(map[string]map[string]float64, len(keys))
	for _, item := range items {
		member := strconv.FormatInt(item.CourseID, 10)
		for key, score := range map[string]float64{
			ResourceRankingCacheKey("comprehensive"):  item.Comprehensive,
			ResourceRankingCacheKey("downloads"):      item.Downloads,
			ResourceRankingCacheKey("views"):          item.Views,
			ResourceRankingCacheKey("likes"):          item.Likes,
			ResourceRankingCacheKey("favorite_count"): item.FavoriteCount,
			ResourceRankingCacheKey("resource_count"): item.ResourceCount,
		} {
			addSnapshotMember(snapshots, key, member, score)
		}
	}
	if err := syncSortedSetSnapshots(snapshots); err != nil {
		return err
	}
	return expireRedisKeys(keys)
}

func (s *Scheduler) SyncRandomCourses() error {
	ids, err := s.courseRepo.ListRandomCourseIDs(constant.RandCoursesCount)
	if err != nil {
		return err
	}
	if len(ids) == 0 {
		if err := utils.RDB.Del(utils.Ctx, constant.CacheRandomCoursesPrefix+"666").Err(); err != nil {
			return err
		}
		return nil
	}
	courses, err := s.courseRepo.FindRandomCourses(ids)
	if err != nil {
		return err
	}

	pipe := utils.RDB.Pipeline()
	pipe.Del(utils.Ctx, constant.CacheRandomCoursesPrefix+"666")
	for _, item := range courses.Items {
		jsonItem, err := json.Marshal(item)
		if err != nil {
			return err
		}
		pipe.RPush(utils.Ctx, constant.CacheRandomCoursesPrefix+"666", string(jsonItem))
	}
	pipe.Expire(utils.Ctx, constant.CacheRandomCoursesPrefix+"666", 2*time.Hour)
	if _, err := pipe.Exec(utils.Ctx); err != nil {
		return err
	}
	return nil
}

func (s *Scheduler) SyncRandomTeachers() error {
	ids := utils.RandUniqueInts(constant.RandTeachersMin, constant.RandTeachersMax, constant.RandTeachersCount)
	if ids == nil || len(ids) == 0 {
		ids = []int64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20}
	}

	teachers, err := s.teacherRepo.FindRandomTeachers(ids)
	if err != nil {
		return err
	}

	pipe := utils.RDB.Pipeline()
	pipe.Del(utils.Ctx, constant.CacheRandomTeachersPrefix+"666")
	for _, teacher := range teachers.Items {
		jsonItem, err := json.Marshal(teacher)
		if err != nil {
			return err
		}
		pipe.RPush(utils.Ctx, constant.CacheRandomTeachersPrefix+"666", string(jsonItem))
	}
	pipe.Expire(utils.Ctx, constant.CacheRandomTeachersPrefix+"666", 2*time.Hour)
	if _, err := pipe.Exec(utils.Ctx); err != nil {
		return err
	}
	return nil
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
