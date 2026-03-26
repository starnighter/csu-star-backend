package repo

import (
	"csu-star-backend/internal/model"
	"sort"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const pgBulkInsertBatchSize = 1000
const rankingTopN = 100

type AggregateRepository interface {
	RefreshTeacherStats() error
	RefreshCourseStats() error
	RefreshResourceStats() error
	RefreshTeacherRankings(period string, since *time.Time) error
	RefreshCourseRankings(period string, since *time.Time) error
	SyncHotKeywords(period string, keywords map[string]float64) error
	TrimSearchHistories(limit int) error
}

type aggregateRepository struct {
	db *gorm.DB
}

func NewAggregateRepository(db *gorm.DB) AggregateRepository {
	return &aggregateRepository{db: db}
}

func (r *aggregateRepository) RefreshTeacherStats() error {
	return r.db.Exec(`
			UPDATE teachers
			SET
				avg_teaching_score = COALESCE(stats.avg_teaching_score, 0),
				avg_grading_score = COALESCE(stats.avg_grading_score, 0),
				avg_attendance_score = COALESCE(stats.avg_attendance_score, 0),
				approval_rate = COALESCE(stats.approval_rate, 0),
				eval_count = COALESCE(stats.eval_count, 0),
				updated_at = CURRENT_TIMESTAMP
			FROM (
				SELECT
					teacher_id,
					ROUND(AVG(teaching_score)::numeric, 2) AS avg_teaching_score,
					ROUND(AVG(grading_score)::numeric, 2) AS avg_grading_score,
					ROUND(AVG(attendance_score)::numeric, 2) AS avg_attendance_score,
					ROUND(AVG(CASE WHEN (teaching_score + grading_score + attendance_score) / 3.0 >= 4 THEN 100 ELSE 0 END)::numeric, 2) AS approval_rate,
					COUNT(*) AS eval_count
				FROM teacher_evaluations
				WHERE status = 'approved'
				GROUP BY teacher_id
			) AS stats
			WHERE teachers.id = stats.teacher_id
		`).Error
}

func (r *aggregateRepository) RefreshCourseStats() error {
	return r.db.Exec(`
		UPDATE courses
		SET
			avg_workload_score = COALESCE(stats.avg_workload_score, 0),
			avg_gain_score = COALESCE(stats.avg_gain_score, 0),
			avg_difficulty_score = COALESCE(stats.avg_difficulty_score, 0),
			eval_count = COALESCE(stats.eval_count, 0),
			resource_count = COALESCE(resource_stats.resource_count, 0),
			hot_score = COALESCE(resource_stats.hot_score, 0),
			updated_at = CURRENT_TIMESTAMP
		FROM (
			SELECT
				course_id,
				ROUND(AVG(workload_score)::numeric, 2) AS avg_workload_score,
				ROUND(AVG(gain_score)::numeric, 2) AS avg_gain_score,
				ROUND(AVG(difficulty_score)::numeric, 2) AS avg_difficulty_score,
				COUNT(*) AS eval_count
			FROM course_evaluations
			WHERE status = 'approved'
			GROUP BY course_id
		) AS stats
		FULL JOIN (
			SELECT
				course_id,
				COUNT(*) AS resource_count,
				COALESCE(SUM(download_count + like_count + comment_count), 0) AS hot_score
			FROM resources
			WHERE status = 'approved'
			GROUP BY course_id
		) AS resource_stats ON resource_stats.course_id = stats.course_id
		WHERE courses.id = COALESCE(stats.course_id, resource_stats.course_id)
	`).Error
}

func (r *aggregateRepository) RefreshResourceStats() error {
	return r.db.Exec(`
		UPDATE resources
		SET
			comment_count = COALESCE((
				SELECT COUNT(*)
				FROM comments
				WHERE comments.target_type = 'resource'
					AND comments.target_id = resources.id
					AND comments.status = 'active'
			), 0),
			updated_at = CURRENT_TIMESTAMP
	`).Error
}

func (r *aggregateRepository) RefreshTeacherRankings(period string, since *time.Time) error {
	type row struct {
		TeacherID    int64
		DepartmentID int16
		AvgScore     float64
		AvgQuality   float64
		AvgGrading   float64
		AvgAttend    float64
		GoodRate     float64
		ResourceCnt  float64
		EvalCnt      float64
	}

	var rows []row
	query := r.db.Table("teachers t").
		Select(`
			t.id AS teacher_id,
			t.department_id,
			COALESCE(ROUND((COALESCE(evals.avg_teaching, t.avg_teaching_score) + COALESCE(evals.avg_grading, t.avg_grading_score) + COALESCE(evals.avg_attendance, t.avg_attendance_score)) / 3.0, 2), 0) AS avg_score,
			COALESCE(evals.avg_teaching, t.avg_teaching_score, 0) AS avg_quality,
			COALESCE(evals.avg_grading, t.avg_grading_score, 0) AS avg_grading,
			COALESCE(evals.avg_attendance, t.avg_attendance_score, 0) AS avg_attend,
			COALESCE(evals.good_rate, t.approval_rate, 0) AS good_rate,
			COALESCE(t.resource_count, 0) AS resource_cnt,
			COALESCE(evals.eval_count, t.eval_count, 0) AS eval_cnt`).
		Joins(`
			LEFT JOIN (
				SELECT
					teacher_id,
					ROUND(AVG(teaching_score)::numeric, 2) AS avg_teaching,
					ROUND(AVG(grading_score)::numeric, 2) AS avg_grading,
					ROUND(AVG(attendance_score)::numeric, 2) AS avg_attendance,
					ROUND(AVG(CASE WHEN (teaching_score + grading_score + attendance_score) / 3.0 >= 4 THEN 100 ELSE 0 END)::numeric, 2) AS good_rate,
					COUNT(*) AS eval_count
				FROM teacher_evaluations
				WHERE status = 'approved'`)
	if since != nil {
		query = query.Joins(" AND created_at >= ? GROUP BY teacher_id ) evals ON evals.teacher_id = t.id", *since)
	} else {
		query = query.Joins(" GROUP BY teacher_id ) evals ON evals.teacher_id = t.id")
	}
	if err := query.Scan(&rows).Error; err != nil {
		return err
	}

	dimensions := map[string]func(row) float64{
		"avg_score":      func(v row) float64 { return v.AvgScore },
		"avg_quality":    func(v row) float64 { return v.AvgQuality },
		"avg_grading":    func(v row) float64 { return v.AvgGrading },
		"avg_attendance": func(v row) float64 { return v.AvgAttend },
		"good_rate":      func(v row) float64 { return v.GoodRate },
		"resource_count": func(v row) float64 { return v.ResourceCnt },
		"eval_count":     func(v row) float64 { return v.EvalCnt },
	}

	for dimension, scoreFn := range dimensions {
		sortedRows := append([]row(nil), rows...)
		sort.SliceStable(sortedRows, func(i, j int) bool {
			if scoreFn(sortedRows[i]) == scoreFn(sortedRows[j]) {
				return sortedRows[i].TeacherID < sortedRows[j].TeacherID
			}
			return scoreFn(sortedRows[i]) > scoreFn(sortedRows[j])
		})

		limit := minInt(len(sortedRows), rankingTopN)
		batch := make([]model.TeacherRankings, 0, limit)
		keepTeacherIDs := make([]int64, 0, limit)
		for i := 0; i < limit; i++ {
			item := sortedRows[i]
			batch = append(batch, model.TeacherRankings{
				TeacherID:    item.TeacherID,
				DepartmentID: item.DepartmentID,
				Period:       period,
				Dimension:    dimension,
				Rank:         i + 1,
				Score:        scoreFn(item),
			})
			keepTeacherIDs = append(keepTeacherIDs, item.TeacherID)
		}

		if err := r.upsertTeacherRankings(period, dimension, batch, keepTeacherIDs); err != nil {
			return err
		}
	}
	return nil
}

func (r *aggregateRepository) RefreshCourseRankings(period string, since *time.Time) error {
	type row struct {
		CourseID      int64
		DepartmentID  *int16
		AvgScore      float64
		AvgHomework   float64
		AvgGain       float64
		AvgExamDiff   float64
		ResourceCount float64
		HotScore      float64
	}

	var rows []row
	query := r.db.Table("courses c").
		Select(`
			c.id AS course_id,
			c.department_id,
			COALESCE(ROUND((COALESCE(evals.avg_homework, c.avg_workload_score) + COALESCE(evals.avg_gain, c.avg_gain_score) + COALESCE(evals.avg_exam_diff, c.avg_difficulty_score)) / 3.0, 2), 0) AS avg_score,
			COALESCE(evals.avg_homework, c.avg_workload_score, 0) AS avg_homework,
			COALESCE(evals.avg_gain, c.avg_gain_score, 0) AS avg_gain,
			COALESCE(evals.avg_exam_diff, c.avg_difficulty_score, 0) AS avg_exam_diff,
			COALESCE(c.resource_count, 0) AS resource_count,
			COALESCE(c.hot_score, 0) AS hot_score`).
		Joins(`
			LEFT JOIN (
				SELECT
					course_id,
					ROUND(AVG(workload_score)::numeric, 2) AS avg_homework,
					ROUND(AVG(gain_score)::numeric, 2) AS avg_gain,
					ROUND(AVG(difficulty_score)::numeric, 2) AS avg_exam_diff
				FROM course_evaluations
				WHERE status = 'approved'`)
	if since != nil {
		query = query.Joins(" AND created_at >= ? GROUP BY course_id ) evals ON evals.course_id = c.id", *since)
	} else {
		query = query.Joins(" GROUP BY course_id ) evals ON evals.course_id = c.id")
	}
	if err := query.Scan(&rows).Error; err != nil {
		return err
	}

	dimensions := map[string]func(row) float64{
		"avg_score":      func(v row) float64 { return v.AvgScore },
		"avg_homework":   func(v row) float64 { return v.AvgHomework },
		"avg_gain":       func(v row) float64 { return v.AvgGain },
		"avg_exam_diff":  func(v row) float64 { return v.AvgExamDiff },
		"resource_count": func(v row) float64 { return v.ResourceCount },
		"hot":            func(v row) float64 { return v.HotScore },
	}

	for dimension, scoreFn := range dimensions {
		sortedRows := append([]row(nil), rows...)
		sort.SliceStable(sortedRows, func(i, j int) bool {
			if scoreFn(sortedRows[i]) == scoreFn(sortedRows[j]) {
				return sortedRows[i].CourseID < sortedRows[j].CourseID
			}
			return scoreFn(sortedRows[i]) > scoreFn(sortedRows[j])
		})

		limit := minInt(len(sortedRows), rankingTopN)
		batch := make([]model.CourseRankings, 0, limit)
		keepCourseIDs := make([]int64, 0, limit)
		for i := 0; i < limit; i++ {
			item := sortedRows[i]
			batch = append(batch, model.CourseRankings{
				CourseID:     item.CourseID,
				DepartmentID: item.DepartmentID,
				Period:       period,
				Dimension:    dimension,
				Rank:         i + 1,
				Score:        scoreFn(item),
			})
			keepCourseIDs = append(keepCourseIDs, item.CourseID)
		}

		if err := r.upsertCourseRankings(period, dimension, batch, keepCourseIDs); err != nil {
			return err
		}
	}
	return nil
}

func (r *aggregateRepository) SyncHotKeywords(period string, keywords map[string]float64) error {
	if err := r.db.Where("period = ?", period).Delete(&model.HotKeywords{}).Error; err != nil {
		return err
	}
	if len(keywords) == 0 {
		return nil
	}
	items := make([]model.HotKeywords, 0, len(keywords))
	for keyword, count := range keywords {
		items = append(items, model.HotKeywords{
			Keyword: keyword,
			Period:  period,
			Count:   int(count),
		})
	}
	return r.db.CreateInBatches(items, pgBulkInsertBatchSize).Error
}

func (r *aggregateRepository) TrimSearchHistories(limit int) error {
	if limit <= 0 {
		limit = 20
	}
	return r.db.Exec(`
		DELETE FROM search_histories
		WHERE id IN (
			SELECT id
			FROM (
				SELECT
					id,
					ROW_NUMBER() OVER (PARTITION BY user_id ORDER BY created_at DESC, id DESC) AS rn
				FROM search_histories
			) ranked
			WHERE rn > ?
		)
	`, limit).Error
}

func (r *aggregateRepository) upsertTeacherRankings(period, dimension string, batch []model.TeacherRankings, keepTeacherIDs []int64) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if len(batch) > 0 {
			if err := tx.Clauses(clause.OnConflict{
				Columns: []clause.Column{
					{Name: "teacher_id"},
					{Name: "period"},
					{Name: "dimension"},
				},
				DoUpdates: clause.AssignmentColumns([]string{"department_id", "rank", "score", "updated_at"}),
			}).CreateInBatches(batch, pgBulkInsertBatchSize).Error; err != nil {
				return err
			}
		}

		cleanup := tx.Where("period = ? AND dimension = ?", period, dimension)
		if len(keepTeacherIDs) > 0 {
			cleanup = cleanup.Where("teacher_id NOT IN ?", keepTeacherIDs)
		}
		return cleanup.Delete(&model.TeacherRankings{}).Error
	})
}

func (r *aggregateRepository) upsertCourseRankings(period, dimension string, batch []model.CourseRankings, keepCourseIDs []int64) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if len(batch) > 0 {
			if err := tx.Clauses(clause.OnConflict{
				Columns: []clause.Column{
					{Name: "course_id"},
					{Name: "period"},
					{Name: "dimension"},
				},
				DoUpdates: clause.AssignmentColumns([]string{"department_id", "rank", "score", "updated_at"}),
			}).CreateInBatches(batch, pgBulkInsertBatchSize).Error; err != nil {
				return err
			}
		}

		cleanup := tx.Where("period = ? AND dimension = ?", period, dimension)
		if len(keepCourseIDs) > 0 {
			cleanup = cleanup.Where("course_id NOT IN ?", keepCourseIDs)
		}
		return cleanup.Delete(&model.CourseRankings{}).Error
	})
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
