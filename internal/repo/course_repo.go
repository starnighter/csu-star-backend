package repo

import (
	"csu-star-backend/internal/model"
	"fmt"
	"time"

	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type CourseListQuery struct {
	Q          string
	CourseType string
	Sort       string
	Page       int
	Size       int
}

type CourseRankingQuery struct {
	RankType    string
	Period      string
	Page        int
	Size        int
	IsIncreased bool
}

type CourseEvaluationQuery struct {
	CourseID int64
	Sort     string
	Page     int
	Size     int
}

type CourseListItem struct {
	ID                 int64   `json:"id"`
	Name               string  `json:"name"`
	DepartmentID       int16   `json:"department_id"`
	DepartmentName     string  `json:"department_name"`
	Credits            float64 `json:"credits"`
	CourseType         string  `json:"course_type"`
	Description        string  `json:"description"`
	AvgWorkloadScore   float64 `json:"avg_workload_score"`
	AvgGainScore       float64 `json:"avg_gain_score"`
	AvgDifficultyScore float64 `json:"avg_difficulty_score"`
	AvgScore           float64 `json:"avg_score"`
	ResourceCount      int     `json:"resource_count"`
	EvalCount          int     `json:"eval_count"`
	HotScore           int     `json:"hot_score"`
}

type CourseTeacherItem struct {
	ID           int64  `json:"id"`
	Name         string `json:"name"`
	Title        string `json:"title"`
	DepartmentID int16  `json:"department_id"`
	AvatarURL    string `json:"avatar_url"`
}

type CourseDetail struct {
	ID                 int64               `json:"id"`
	Name               string              `json:"name"`
	DepartmentID       int16               `json:"department_id"`
	DepartmentName     string              `json:"department_name"`
	Credits            float64             `json:"credits"`
	CourseType         string              `json:"course_type"`
	Description        string              `json:"description"`
	Metadata           datatypes.JSON      `json:"metadata"`
	AvgWorkloadScore   float64             `json:"avg_workload_score"`
	AvgGainScore       float64             `json:"avg_gain_score"`
	AvgDifficultyScore float64             `json:"avg_difficulty_score"`
	AvgScore           float64             `json:"avg_score"`
	ResourceCount      int                 `json:"resource_count"`
	EvalCount          int                 `json:"eval_count"`
	HotScore           int                 `json:"hot_score"`
	CreatedAt          time.Time           `json:"created_at"`
	UpdatedAt          time.Time           `json:"updated_at"`
	Teachers           []CourseTeacherItem `json:"teachers"`
}

type CourseEvaluationItem struct {
	ID              int64     `json:"id"`
	UserID          int64     `json:"user_id"`
	CourseID        int64     `json:"course_id"`
	WorkloadScore   int       `json:"workload_score"`
	GainScore       int       `json:"gain_score"`
	DifficultyScore int       `json:"difficulty_score"`
	AvgRating       float64   `json:"avg_rating"`
	Comment         string    `json:"comment"`
	IsAnonymous     bool      `json:"is_anonymous"`
	Status          string    `json:"status"`
	LikeCount       int64     `json:"like_count"`
	IsLiked         bool      `json:"is_liked"`
	AuthorID        int64     `json:"author_id"`
	AuthorName      string    `json:"author_name"`
	AuthorAvatarURL string    `json:"author_avatar_url"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

type MyCourseEvaluationItem struct {
	ID              int64     `json:"id"`
	CourseID        int64     `json:"course_id"`
	CourseName      string    `json:"course_name"`
	WorkloadScore   int       `json:"workload_score"`
	GainScore       int       `json:"gain_score"`
	DifficultyScore int       `json:"difficulty_score"`
	AvgRating       float64   `json:"avg_rating"`
	Comment         string    `json:"comment"`
	IsAnonymous     bool      `json:"is_anonymous"`
	Status          string    `json:"status"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

type CourseRankingItem struct {
	ID             int64   `json:"id"`
	Name           string  `json:"name"`
	DepartmentID   int16   `json:"department_id"`
	DepartmentName string  `json:"department_name"`
	Score          float64 `json:"score"`
	Rank           int64   `json:"rank"`
}

type CourseRepository interface {
	FindCourses(query CourseListQuery) ([]CourseListItem, int64, error)
	FindCourseDetail(id int64) (*CourseDetail, error)
	FindCourseRankings(query CourseRankingQuery) ([]CourseRankingItem, int64, error)
	FindCourseRankingItemsByIDs(ids []int64) ([]CourseRankingItem, error)
	ListCourseEvaluations(query CourseEvaluationQuery) ([]CourseEvaluationItem, int64, error)
	ListMyCourseEvaluations(userID int64, page, size int) ([]MyCourseEvaluationItem, int64, error)
	CreateCourseEvaluation(evaluation *model.CourseEvaluations) error
	UpdateCourseEvaluation(evaluation *model.CourseEvaluations) error
	DeleteCourseEvaluation(id int64) error
	GetCourseEvaluationByID(id int64) (*model.CourseEvaluations, error)
	CourseExists(id int64) (bool, error)
	RecalculateCourseStats(courseID int64) error
}

type courseRepository struct {
	db *gorm.DB
}

func NewCourseRepository(db *gorm.DB) CourseRepository {
	return &courseRepository{db: db}
}

func (r *courseRepository) FindCourses(query CourseListQuery) ([]CourseListItem, int64, error) {
	var items []CourseListItem
	var total int64

	base := r.db.Table("courses").
		Joins("LEFT JOIN departments ON departments.id = courses.department_id")

	if query.Q != "" {
		base = base.Where("courses.name ILIKE ?", "%"+query.Q+"%")
	}
	if query.CourseType != "" {
		switch query.CourseType {
		case string(model.CourseTypePublic), string(model.CourseTypeNonPublic):
			base = base.Where("courses.course_type = ?", query.CourseType)
		}
	}

	if err := base.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	err := base.Select(`
		courses.id,
		courses.name,
		courses.department_id,
		departments.name AS department_name,
		courses.credits,
		courses.course_type,
		courses.description,
		COALESCE(courses.avg_workload_score, 0) AS avg_workload_score,
		COALESCE(courses.avg_gain_score, 0) AS avg_gain_score,
		COALESCE(courses.avg_difficulty_score, 0) AS avg_difficulty_score,
		ROUND((COALESCE(courses.avg_workload_score, 0) + COALESCE(courses.avg_gain_score, 0) + COALESCE(courses.avg_difficulty_score, 0)) / 3.0, 2) AS avg_score,
		COALESCE(courses.resource_count, 0) AS resource_count,
		COALESCE(courses.eval_count, 0) AS eval_count,
		COALESCE(courses.hot_score, 0) AS hot_score`).
		Order(courseSortExpr(query.Sort) + " DESC").
		Order("courses.id ASC").
		Offset((query.Page - 1) * query.Size).
		Limit(query.Size).
		Scan(&items).Error
	if err != nil {
		return nil, 0, err
	}

	return items, total, nil
}

func (r *courseRepository) FindCourseDetail(id int64) (*CourseDetail, error) {
	var detail CourseDetail
	err := r.db.Table("courses").
		Joins("LEFT JOIN departments ON departments.id = courses.department_id").
		Select(`
			courses.id,
			courses.name,
			courses.department_id,
			departments.name AS department_name,
			courses.credits,
			courses.course_type,
			courses.description,
			courses.metadata,
			COALESCE(courses.avg_workload_score, 0) AS avg_workload_score,
			COALESCE(courses.avg_gain_score, 0) AS avg_gain_score,
			COALESCE(courses.avg_difficulty_score, 0) AS avg_difficulty_score,
			ROUND((COALESCE(courses.avg_workload_score, 0) + COALESCE(courses.avg_gain_score, 0) + COALESCE(courses.avg_difficulty_score, 0)) / 3.0, 2) AS avg_score,
			COALESCE(courses.resource_count, 0) AS resource_count,
			COALESCE(courses.eval_count, 0) AS eval_count,
			COALESCE(courses.hot_score, 0) AS hot_score,
			courses.created_at,
			courses.updated_at`).
		Where("courses.id = ?", id).
		Scan(&detail).Error
	if err != nil {
		return nil, err
	}
	if detail.ID == 0 {
		return nil, gorm.ErrRecordNotFound
	}

	var teachers []CourseTeacherItem
	err = r.db.Table("course_teachers").
		Joins("JOIN teachers ON teachers.id = course_teachers.teacher_id").
		Where("course_teachers.course_id = ?", id).
		Distinct("teachers.id, teachers.name, teachers.title, teachers.department_id, teachers.avatar_url").
		Order("teachers.id ASC").
		Scan(&teachers).Error
	if err != nil {
		return nil, err
	}
	detail.Teachers = teachers

	return &detail, nil
}

func (r *courseRepository) FindCourseRankings(query CourseRankingQuery) ([]CourseRankingItem, int64, error) {
	var total int64
	rankingBase := r.db.Table("course_rankings").
		Where("dimension = ? AND period = ?", query.RankType, query.Period)
	if err := rankingBase.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if total > 0 {
		var items []CourseRankingItem
		orderDirection := "DESC"
		if query.IsIncreased {
			orderDirection = "ASC"
		}
		err := rankingBase.
			Joins("JOIN courses ON courses.id = course_rankings.course_id").
			Joins("LEFT JOIN departments ON departments.id = courses.department_id").
			Select(`
				courses.id,
				courses.name,
				courses.department_id,
				departments.name AS department_name,
				COALESCE(course_rankings.score, 0) AS score`).
			Order("course_rankings.score " + orderDirection).
			Order("course_rankings.course_id ASC").
			Offset((query.Page - 1) * query.Size).
			Limit(query.Size).
			Scan(&items).Error
		if err != nil {
			return nil, 0, err
		}
		startRank := int64((query.Page-1)*query.Size + 1)
		for i := range items {
			items[i].Rank = startRank + int64(i)
		}
		return items, total, nil
	}

	base := r.db.Table("courses").
		Joins("LEFT JOIN departments ON departments.id = courses.department_id")
	if err := base.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var items []CourseRankingItem
	orderDirection := "DESC"
	if query.IsIncreased {
		orderDirection = "ASC"
	}

	err := base.Select(fmt.Sprintf(`
		courses.id,
		courses.name,
		courses.department_id,
		departments.name AS department_name,
		%s AS score`, courseRankingExpr(query.RankType))).
		Order(courseRankingExpr(query.RankType) + " " + orderDirection).
		Order("courses.id ASC").
		Offset((query.Page - 1) * query.Size).
		Limit(query.Size).
		Scan(&items).Error
	if err != nil {
		return nil, 0, err
	}

	startRank := int64((query.Page-1)*query.Size + 1)
	for i := range items {
		items[i].Rank = startRank + int64(i)
	}

	return items, total, nil
}

func (r *courseRepository) FindCourseRankingItemsByIDs(ids []int64) ([]CourseRankingItem, error) {
	if len(ids) == 0 {
		return []CourseRankingItem{}, nil
	}

	var items []CourseRankingItem
	err := r.db.Table("courses").
		Joins("LEFT JOIN departments ON departments.id = courses.department_id").
		Select(`
			courses.id,
			courses.name,
			courses.department_id,
			departments.name AS department_name`).
		Where("courses.id IN ?", ids).
		Scan(&items).Error
	if err != nil {
		return nil, err
	}
	return items, nil
}

func (r *courseRepository) ListCourseEvaluations(query CourseEvaluationQuery) ([]CourseEvaluationItem, int64, error) {
	var items []CourseEvaluationItem
	var total int64

	base := r.db.Table("course_evaluations").
		Where("course_evaluations.course_id = ? AND course_evaluations.status = ?", query.CourseID, model.ResourceStatusApproved)

	if err := base.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	err := base.
		Joins("JOIN users ON users.id = course_evaluations.user_id").
		Select(`
			course_evaluations.id,
			course_evaluations.user_id,
			course_evaluations.course_id,
			course_evaluations.workload_score,
			course_evaluations.gain_score,
			course_evaluations.difficulty_score,
			ROUND((course_evaluations.workload_score + course_evaluations.gain_score + course_evaluations.difficulty_score) / 3.0, 2) AS avg_rating,
			course_evaluations.comment,
			course_evaluations.is_anonymous,
			course_evaluations.status,
			COALESCE((
				SELECT COUNT(*)
				FROM likes
				WHERE likes.target_type = 'course_evaluation' AND likes.target_id = course_evaluations.id
			), 0) AS like_count,
			FALSE AS is_liked,
			users.id AS author_id,
			users.nickname AS author_name,
			users.avatar_url AS author_avatar_url,
			course_evaluations.created_at,
			course_evaluations.updated_at`).
		Order(courseEvaluationSortExpr(query.Sort)).
		Order("course_evaluations.id DESC").
		Offset((query.Page - 1) * query.Size).
		Limit(query.Size).
		Scan(&items).Error
	if err != nil {
		return nil, 0, err
	}

	return items, total, nil
}

func (r *courseRepository) ListMyCourseEvaluations(userID int64, page, size int) ([]MyCourseEvaluationItem, int64, error) {
	var items []MyCourseEvaluationItem
	var total int64

	base := r.db.Table("course_evaluations").Where("course_evaluations.user_id = ?", userID)
	if err := base.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	err := base.
		Joins("JOIN courses ON courses.id = course_evaluations.course_id").
		Select(`
			course_evaluations.id,
			course_evaluations.course_id,
			courses.name AS course_name,
			course_evaluations.workload_score,
			course_evaluations.gain_score,
			course_evaluations.difficulty_score,
			ROUND((course_evaluations.workload_score + course_evaluations.gain_score + course_evaluations.difficulty_score) / 3.0, 2) AS avg_rating,
			course_evaluations.comment,
			course_evaluations.is_anonymous,
			course_evaluations.status,
			course_evaluations.created_at,
			course_evaluations.updated_at`).
		Order("course_evaluations.created_at DESC").
		Order("course_evaluations.id DESC").
		Offset((page - 1) * size).
		Limit(size).
		Scan(&items).Error
	if err != nil {
		return nil, 0, err
	}

	return items, total, nil
}

func (r *courseRepository) CreateCourseEvaluation(evaluation *model.CourseEvaluations) error {
	return r.db.Create(evaluation).Error
}

func (r *courseRepository) UpdateCourseEvaluation(evaluation *model.CourseEvaluations) error {
	return r.db.Model(&model.CourseEvaluations{}).
		Where("id = ?", evaluation.ID).
		Updates(map[string]interface{}{
			"workload_score":   evaluation.WorkloadScore,
			"gain_score":       evaluation.GainScore,
			"difficulty_score": evaluation.DifficultyScore,
			"comment":          evaluation.Comment,
			"is_anonymous":     evaluation.IsAnonymous,
			"status":           evaluation.Status,
			"updated_at":       time.Now(),
		}).Error
}

func (r *courseRepository) DeleteCourseEvaluation(id int64) error {
	return r.db.Delete(&model.CourseEvaluations{}, id).Error
}

func (r *courseRepository) GetCourseEvaluationByID(id int64) (*model.CourseEvaluations, error) {
	var evaluation model.CourseEvaluations
	err := r.db.First(&evaluation, id).Error
	if err != nil {
		return nil, err
	}
	return &evaluation, nil
}

func (r *courseRepository) CourseExists(id int64) (bool, error) {
	var count int64
	if err := r.db.Table("courses").Where("id = ?", id).Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r *courseRepository) RecalculateCourseStats(courseID int64) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		return tx.Exec(`
			UPDATE courses
			SET
				avg_workload_score = stats.avg_workload_score,
				avg_gain_score = stats.avg_gain_score,
				avg_difficulty_score = stats.avg_difficulty_score,
				eval_count = stats.eval_count,
				resource_count = resource_stats.resource_count,
				hot_score = resource_stats.hot_score,
				updated_at = CURRENT_TIMESTAMP
			FROM (
				SELECT
					COALESCE(ROUND(AVG(workload_score)::numeric, 2), 0) AS avg_workload_score,
					COALESCE(ROUND(AVG(gain_score)::numeric, 2), 0) AS avg_gain_score,
					COALESCE(ROUND(AVG(difficulty_score)::numeric, 2), 0) AS avg_difficulty_score,
					COUNT(*) AS eval_count
				FROM course_evaluations
				WHERE course_id = ? AND status = 'approved'
			) AS stats,
			(
				SELECT
					COUNT(*) AS resource_count,
					COALESCE(SUM(download_count + like_count + comment_count), 0) AS hot_score
				FROM resources
				WHERE course_id = ? AND status = 'approved'
			) AS resource_stats
			WHERE courses.id = ?`, courseID, courseID, courseID).Error
	})
}

func courseSortExpr(sort string) string {
	switch sort {
	case "avg_homework", "workload_score":
		return "COALESCE(courses.avg_workload_score, 0)"
	case "avg_gain", "gain_score":
		return "COALESCE(courses.avg_gain_score, 0)"
	case "avg_exam_diff", "difficulty_score":
		return "COALESCE(courses.avg_difficulty_score, 0)"
	case "resource_count":
		return "COALESCE(courses.resource_count, 0)"
	case "hot", "hot_score":
		return "COALESCE(courses.hot_score, 0)"
	default:
		return "ROUND((COALESCE(courses.avg_workload_score, 0) + COALESCE(courses.avg_gain_score, 0) + COALESCE(courses.avg_difficulty_score, 0)) / 3.0, 2)"
	}
}

func courseRankingExpr(rankType string) string {
	return courseSortExpr(rankType)
}

func courseEvaluationSortExpr(sort string) string {
	switch sort {
	case "avg_rating":
		return "ROUND((course_evaluations.workload_score + course_evaluations.gain_score + course_evaluations.difficulty_score) / 3.0, 2) DESC"
	case "likes":
		return `(
			SELECT COUNT(*)
			FROM likes
			WHERE likes.target_type = 'course_evaluation' AND likes.target_id = course_evaluations.id
		) DESC`
	default:
		return "course_evaluations.created_at DESC"
	}
}
