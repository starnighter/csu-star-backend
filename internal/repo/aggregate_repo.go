package repo

import (
	"csu-star-backend/internal/model"
	"fmt"
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
}

type aggregateRepository struct {
	db *gorm.DB
}

func NewAggregateRepository(db *gorm.DB) AggregateRepository {
	return &aggregateRepository{db: db}
}

func (r *aggregateRepository) RefreshTeacherStats() error {
	return r.db.Exec(fmt.Sprintf(`
		UPDATE teachers
		SET
			avg_teaching_score = COALESCE(stats.avg_teaching_score, 0),
			avg_grading_score = COALESCE(stats.avg_grading_score, 0),
			avg_attendance_score = COALESCE(stats.avg_attendance_score, 0),
			approval_rate = COALESCE(stats.approval_rate, 0),
			eval_count = COALESCE(stats.eval_count, 0),
			favorite_count = COALESCE(favorite_stats.favorite_count, 0),
			updated_at = CURRENT_TIMESTAMP
		FROM teachers AS base
		LEFT JOIN (
			SELECT
				teacher_id,
				ROUND(AVG(teaching_score)::numeric, 2) AS avg_teaching_score,
				ROUND(AVG(grading_score)::numeric, 2) AS avg_grading_score,
				ROUND(AVG(attendance_score)::numeric, 2) AS avg_attendance_score,
				ROUND(AVG(CASE WHEN (teaching_score + grading_score + attendance_score) / 3.0 >= 4 THEN 100 ELSE 0 END)::numeric, 2) AS approval_rate,
				COUNT(*) AS eval_count
			FROM teacher_evaluations
			WHERE %s
			GROUP BY teacher_id
		) AS stats ON stats.teacher_id = base.id
		LEFT JOIN (
			SELECT
				target_id AS teacher_id,
				COUNT(*) AS favorite_count
			FROM favorites
			WHERE target_type = 'teacher'
			GROUP BY target_id
		) AS favorite_stats ON favorite_stats.teacher_id = base.id
		WHERE teachers.id = base.id
	`, visibleTeacherEvaluationCondition("teacher_evaluations"))).Error
}

func (r *aggregateRepository) RefreshCourseStats() error {
	return r.db.Exec(fmt.Sprintf(`
		UPDATE courses
		SET
			avg_workload_score = COALESCE(stats.avg_workload_score, 0),
			avg_gain_score = COALESCE(stats.avg_gain_score, 0),
			avg_difficulty_score = COALESCE(stats.avg_difficulty_score, 0),
			eval_count = COALESCE(stats.eval_count, 0),
			resource_count = COALESCE(resource_stats.resource_count, 0),
			download_total = COALESCE(resource_stats.download_total, 0),
			view_total = COALESCE(resource_stats.view_total, 0),
			like_total = COALESCE(resource_stats.like_total, 0),
			favorite_count = COALESCE(favorite_stats.favorite_count, 0),
			resource_favorite_count = COALESCE(resource_favorite_stats.favorite_count, 0),
			updated_at = CURRENT_TIMESTAMP
		FROM courses AS base
		LEFT JOIN (
			SELECT
				course_id,
				ROUND(AVG(workload_score)::numeric, 2) AS avg_workload_score,
				ROUND(AVG(gain_score)::numeric, 2) AS avg_gain_score,
				ROUND(AVG(difficulty_score)::numeric, 2) AS avg_difficulty_score,
				COUNT(*) AS eval_count
			FROM course_evaluations
			WHERE %s
			GROUP BY course_id
		) AS stats ON stats.course_id = base.id
		LEFT JOIN (
			SELECT
				course_id,
				COUNT(*) AS resource_count,
				COALESCE(SUM(download_count), 0) AS download_total,
				COALESCE(SUM(view_count), 0) AS view_total,
				COALESCE(SUM(like_count), 0) AS like_total
			FROM resources
			WHERE status = 'approved'
			GROUP BY course_id
		) AS resource_stats ON resource_stats.course_id = base.id
		LEFT JOIN (
			SELECT
				target_id AS course_id,
				COUNT(*) AS favorite_count
			FROM favorites
			WHERE target_type = 'course'
			GROUP BY target_id
		) AS favorite_stats ON favorite_stats.course_id = base.id
		LEFT JOIN (
			SELECT
				resources.course_id,
				COUNT(*) AS favorite_count
			FROM favorites
			JOIN resources ON resources.id = favorites.target_id
			WHERE favorites.target_type = 'resource'
				AND resources.status = 'approved'
			GROUP BY resources.course_id
		) AS resource_favorite_stats ON resource_favorite_stats.course_id = base.id
		WHERE courses.id = base.id
	`, visibleCourseEvaluationCondition("course_evaluations"))).Error
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
		FavoriteCnt  float64
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
				COALESCE(t.favorite_count, 0) AS favorite_cnt,
				COALESCE(evals.eval_count, t.eval_count, 0) AS eval_cnt`).
		Joins(`
				LEFT JOIN (
					SELECT
					teacher_id,
					ROUND(AVG(teaching_score)::numeric, 2) AS avg_teaching,
					ROUND(AVG(grading_score)::numeric, 2) AS avg_grading,
					ROUND(AVG(attendance_score)::numeric, 2) AS avg_attendance,
					COUNT(*) AS eval_count
				FROM teacher_evaluations
				WHERE ` + visibleTeacherEvaluationCondition("teacher_evaluations"))
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
		"favorite_count": func(v row) float64 { return v.FavoriteCnt },
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
		AvgScore      float64
		AvgHomework   float64
		AvgGain       float64
		AvgExamDiff   float64
		ResourceCount float64
		FavoriteCnt   float64
	}

	var rows []row
	query := r.db.Table("courses c").
		Select(`
				c.id AS course_id,
				COALESCE(ROUND((COALESCE(evals.avg_homework, c.avg_workload_score) + COALESCE(evals.avg_gain, c.avg_gain_score) + COALESCE(evals.avg_exam_diff, c.avg_difficulty_score)) / 3.0, 2), 0) AS avg_score,
				COALESCE(evals.avg_homework, c.avg_workload_score, 0) AS avg_homework,
				COALESCE(evals.avg_gain, c.avg_gain_score, 0) AS avg_gain,
				COALESCE(evals.avg_exam_diff, c.avg_difficulty_score, 0) AS avg_exam_diff,
				COALESCE(c.resource_count, 0) AS resource_count,
				COALESCE(c.favorite_count, 0) AS favorite_cnt`).
		Joins(`
				LEFT JOIN (
					SELECT
					course_id,
					ROUND(AVG(workload_score)::numeric, 2) AS avg_homework,
					ROUND(AVG(gain_score)::numeric, 2) AS avg_gain,
					ROUND(AVG(difficulty_score)::numeric, 2) AS avg_exam_diff
				FROM course_evaluations
				WHERE ` + visibleCourseEvaluationCondition("course_evaluations"))
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
		"favorite_count": func(v row) float64 { return v.FavoriteCnt },
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
				CourseID:  item.CourseID,
				Period:    period,
				Dimension: dimension,
				Rank:      i + 1,
				Score:     scoreFn(item),
			})
			keepCourseIDs = append(keepCourseIDs, item.CourseID)
		}

		if err := r.upsertCourseRankings(period, dimension, batch, keepCourseIDs); err != nil {
			return err
		}
	}
	return nil
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
				DoUpdates: clause.AssignmentColumns([]string{"rank", "score", "updated_at"}),
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
