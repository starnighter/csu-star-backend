package repo

import (
	"csu-star-backend/internal/model"
	"fmt"
	"time"

	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type TeacherListQuery struct {
	Q            string
	DepartmentID int16
	Sort         string
	Page         int
	Size         int
}

type TeacherRankingQuery struct {
	RankType     string
	Period       string
	DepartmentID int16
	Page         int
	Size         int
	IsIncreased  bool
}

type TeacherEvaluationQuery struct {
	TeacherID int64
	Sort      string
	Page      int
	Size      int
}

type TeacherListItem struct {
	ID                 int64   `json:"id"`
	Name               string  `json:"name"`
	Title              string  `json:"title"`
	DepartmentID       int16   `json:"department_id"`
	DepartmentName     string  `json:"department_name"`
	AvatarURL          string  `json:"avatar_url"`
	AvgTeachingScore   float64 `json:"avg_teaching_score"`
	AvgGradingScore    float64 `json:"avg_grading_score"`
	AvgAttendanceScore float64 `json:"avg_attendance_score"`
	AvgScore           float64 `json:"avg_score"`
	ApprovalRate       float64 `json:"approval_rate"`
	ResourceCount      int     `json:"resource_count"`
	EvalCount          int64   `json:"eval_count"`
}

type TeacherCourseItem struct {
	ID           int64   `json:"id"`
	Name         string  `json:"name"`
	DepartmentID int16   `json:"department_id"`
	Credits      float64 `json:"credits"`
	CourseType   string  `json:"course_type"`
	Description  string  `json:"description"`
}

type TeacherDetail struct {
	ID                 int64               `json:"id"`
	Name               string              `json:"name"`
	Title              string              `json:"title"`
	DepartmentID       int16               `json:"department_id"`
	DepartmentName     string              `json:"department_name"`
	AvatarURL          string              `json:"avatar_url"`
	Metadata           datatypes.JSON      `json:"metadata"`
	AvgTeachingScore   float64             `json:"avg_teaching_score"`
	AvgGradingScore    float64             `json:"avg_grading_score"`
	AvgAttendanceScore float64             `json:"avg_attendance_score"`
	AvgScore           float64             `json:"avg_score"`
	ApprovalRate       float64             `json:"approval_rate"`
	ResourceCount      int                 `json:"resource_count"`
	EvalCount          int64               `json:"eval_count"`
	CreatedAt          time.Time           `json:"created_at"`
	UpdatedAt          time.Time           `json:"updated_at"`
	Courses            []TeacherCourseItem `json:"courses"`
}

type TeacherEvaluationItem struct {
	ID              int64     `json:"id"`
	UserID          int64     `json:"user_id"`
	TeacherID       int64     `json:"teacher_id"`
	CourseID        int64     `json:"course_id"`
	CourseName      string    `json:"course_name"`
	TeachingScore   int       `json:"teaching_score"`
	GradingScore    int       `json:"grading_score"`
	AttendanceScore int       `json:"attendance_score"`
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

type MyTeacherEvaluationItem struct {
	ID              int64     `json:"id"`
	TeacherID       int64     `json:"teacher_id"`
	TeacherName     string    `json:"teacher_name"`
	CourseID        int64     `json:"course_id"`
	CourseName      string    `json:"course_name"`
	TeachingScore   int       `json:"teaching_score"`
	GradingScore    int       `json:"grading_score"`
	AttendanceScore int       `json:"attendance_score"`
	AvgRating       float64   `json:"avg_rating"`
	Comment         string    `json:"comment"`
	IsAnonymous     bool      `json:"is_anonymous"`
	Status          string    `json:"status"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

type TeacherRankingItem struct {
	ID             int64   `json:"id"`
	Name           string  `json:"name"`
	Title          string  `json:"title"`
	DepartmentID   int16   `json:"department_id"`
	DepartmentName string  `json:"department_name"`
	Score          float64 `json:"score"`
	Rank           int64   `json:"rank"`
}

type TeacherRepository interface {
	FindTeachers(query TeacherListQuery) ([]TeacherListItem, int64, error)
	FindTeacherDetail(id int64) (*TeacherDetail, error)
	FindTeacherRankings(query TeacherRankingQuery) ([]TeacherRankingItem, int64, error)
	FindTeacherRankingItemsByIDs(ids []int64) ([]TeacherRankingItem, error)
	ListTeacherEvaluations(query TeacherEvaluationQuery) ([]TeacherEvaluationItem, int64, error)
	ListMyTeacherEvaluations(userID int64, page, size int) ([]MyTeacherEvaluationItem, int64, error)
	CreateTeacherEvaluation(evaluation *model.TeacherEvaluations) error
	UpdateTeacherEvaluation(evaluation *model.TeacherEvaluations) error
	DeleteTeacherEvaluation(id int64) error
	GetTeacherEvaluationByID(id int64) (*model.TeacherEvaluations, error)
	TeacherExists(id int64) (bool, error)
	CourseExists(id int64) (bool, error)
	TeacherCourseRelationExists(teacherID, courseID int64) (bool, error)
	RecalculateTeacherStats(teacherID int64) error
}

type teacherRepository struct {
	db *gorm.DB
}

func NewTeacherRepository(db *gorm.DB) TeacherRepository {
	return &teacherRepository{db: db}
}

func (r *teacherRepository) FindTeachers(query TeacherListQuery) ([]TeacherListItem, int64, error) {
	var items []TeacherListItem
	var total int64

	base := r.db.Table("teachers").
		Joins("LEFT JOIN departments ON departments.id = teachers.department_id")

	if query.Q != "" {
		base = base.Where("teachers.name ILIKE ?", "%"+query.Q+"%")
	}
	if query.DepartmentID > 0 {
		base = base.Where("teachers.department_id = ?", query.DepartmentID)
	}

	if err := base.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	orderBy := teacherSortExpr(query.Sort) + " DESC"
	err := base.Select(`
		teachers.id,
		teachers.name,
		teachers.title,
		teachers.department_id,
		departments.name AS department_name,
		teachers.avatar_url,
		COALESCE(teachers.avg_teaching_score, 0) AS avg_teaching_score,
		COALESCE(teachers.avg_grading_score, 0) AS avg_grading_score,
		COALESCE(teachers.avg_attendance_score, 0) AS avg_attendance_score,
		ROUND((COALESCE(teachers.avg_teaching_score, 0) + COALESCE(teachers.avg_grading_score, 0) + COALESCE(teachers.avg_attendance_score, 0)) / 3.0, 2) AS avg_score,
		COALESCE(teachers.approval_rate, 0) AS approval_rate,
		COALESCE(teachers.resource_count, 0) AS resource_count,
		COALESCE(teachers.eval_count, 0) AS eval_count`).
		Order(orderBy).
		Order("teachers.id ASC").
		Offset((query.Page - 1) * query.Size).
		Limit(query.Size).
		Scan(&items).Error
	if err != nil {
		return nil, 0, err
	}

	return items, total, nil
}

func (r *teacherRepository) FindTeacherDetail(id int64) (*TeacherDetail, error) {
	var detail TeacherDetail
	err := r.db.Table("teachers").
		Joins("LEFT JOIN departments ON departments.id = teachers.department_id").
		Select(`
			teachers.id,
			teachers.name,
			teachers.title,
			teachers.department_id,
			departments.name AS department_name,
			teachers.avatar_url,
			teachers.metadata,
			COALESCE(teachers.avg_teaching_score, 0) AS avg_teaching_score,
			COALESCE(teachers.avg_grading_score, 0) AS avg_grading_score,
			COALESCE(teachers.avg_attendance_score, 0) AS avg_attendance_score,
			ROUND((COALESCE(teachers.avg_teaching_score, 0) + COALESCE(teachers.avg_grading_score, 0) + COALESCE(teachers.avg_attendance_score, 0)) / 3.0, 2) AS avg_score,
			COALESCE(teachers.approval_rate, 0) AS approval_rate,
			COALESCE(teachers.resource_count, 0) AS resource_count,
			COALESCE(teachers.eval_count, 0) AS eval_count,
			teachers.created_at,
			teachers.updated_at`).
		Where("teachers.id = ?", id).
		Scan(&detail).Error
	if err != nil {
		return nil, err
	}
	if detail.ID == 0 {
		return nil, gorm.ErrRecordNotFound
	}

	var courses []TeacherCourseItem
	err = r.db.Table("course_teachers").
		Joins("JOIN courses ON courses.id = course_teachers.course_id").
		Where("course_teachers.teacher_id = ?", id).
		Distinct("courses.id, courses.name, courses.department_id, courses.credits, courses.course_type, courses.description").
		Order("courses.id ASC").
		Scan(&courses).Error
	if err != nil {
		return nil, err
	}
	detail.Courses = courses

	return &detail, nil
}

func (r *teacherRepository) FindTeacherRankings(query TeacherRankingQuery) ([]TeacherRankingItem, int64, error) {
	var total int64
	rankingBase := r.db.Table("teacher_rankings").
		Where("dimension = ? AND period = ?", query.RankType, query.Period)
	if query.DepartmentID > 0 {
		rankingBase = rankingBase.Where("department_id = ?", query.DepartmentID)
	}

	if err := rankingBase.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if total > 0 {
		var items []TeacherRankingItem
		orderDirection := "DESC"
		if query.IsIncreased {
			orderDirection = "ASC"
		}
		err := rankingBase.
			Joins("JOIN teachers ON teachers.id = teacher_rankings.teacher_id").
			Joins("LEFT JOIN departments ON departments.id = teachers.department_id").
			Select(`
				teachers.id,
				teachers.name,
				teachers.title,
				teachers.department_id,
				departments.name AS department_name,
				COALESCE(teacher_rankings.score, 0) AS score`).
			Order("teacher_rankings.score " + orderDirection).
			Order("teacher_rankings.teacher_id ASC").
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

	var items []TeacherRankingItem
	base := r.db.Table("teachers").
		Joins("LEFT JOIN departments ON departments.id = teachers.department_id")
	if query.DepartmentID > 0 {
		base = base.Where("teachers.department_id = ?", query.DepartmentID)
	}

	if err := base.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	orderDirection := "DESC"
	if query.IsIncreased {
		orderDirection = "ASC"
	}

	err := base.Select(fmt.Sprintf(`
		teachers.id,
		teachers.name,
		teachers.title,
		teachers.department_id,
		departments.name AS department_name,
		%s AS score`, teacherSortExpr(query.RankType))).
		Order(teacherSortExpr(query.RankType) + " " + orderDirection).
		Order("teachers.id ASC").
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

func (r *teacherRepository) FindTeacherRankingItemsByIDs(ids []int64) ([]TeacherRankingItem, error) {
	if len(ids) == 0 {
		return []TeacherRankingItem{}, nil
	}

	var items []TeacherRankingItem
	err := r.db.Table("teachers").
		Joins("LEFT JOIN departments ON departments.id = teachers.department_id").
		Select(`
			teachers.id,
			teachers.name,
			teachers.title,
			teachers.department_id,
			departments.name AS department_name`).
		Where("teachers.id IN ?", ids).
		Scan(&items).Error
	if err != nil {
		return nil, err
	}
	return items, nil
}

func (r *teacherRepository) ListTeacherEvaluations(query TeacherEvaluationQuery) ([]TeacherEvaluationItem, int64, error) {
	var items []TeacherEvaluationItem
	var total int64

	base := r.db.Table("teacher_evaluations").
		Where("teacher_evaluations.teacher_id = ? AND teacher_evaluations.status = ?", query.TeacherID, model.ResourceStatusApproved)

	if err := base.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	err := base.
		Joins("JOIN users ON users.id = teacher_evaluations.user_id").
		Joins("LEFT JOIN courses ON courses.id = teacher_evaluations.course_id").
		Select(`
			teacher_evaluations.id,
			teacher_evaluations.user_id,
			teacher_evaluations.teacher_id,
			teacher_evaluations.course_id,
			courses.name AS course_name,
			teacher_evaluations.teaching_score,
			teacher_evaluations.grading_score,
			teacher_evaluations.attendance_score,
			ROUND((teacher_evaluations.teaching_score + teacher_evaluations.grading_score + teacher_evaluations.attendance_score) / 3.0, 2) AS avg_rating,
			teacher_evaluations.comment,
			teacher_evaluations.is_anonymous,
			teacher_evaluations.status,
			COALESCE((
				SELECT COUNT(*)
				FROM likes
				WHERE likes.target_type = 'teacher_evaluation' AND likes.target_id = teacher_evaluations.id
			), 0) AS like_count,
			FALSE AS is_liked,
			users.id AS author_id,
			users.nickname AS author_name,
			users.avatar_url AS author_avatar_url,
			teacher_evaluations.created_at,
			teacher_evaluations.updated_at`).
		Order(teacherEvaluationSortExpr(query.Sort)).
		Order("teacher_evaluations.id DESC").
		Offset((query.Page - 1) * query.Size).
		Limit(query.Size).
		Scan(&items).Error
	if err != nil {
		return nil, 0, err
	}

	return items, total, nil
}

func (r *teacherRepository) ListMyTeacherEvaluations(userID int64, page, size int) ([]MyTeacherEvaluationItem, int64, error) {
	var items []MyTeacherEvaluationItem
	var total int64

	base := r.db.Table("teacher_evaluations").Where("teacher_evaluations.user_id = ?", userID)
	if err := base.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	err := base.
		Joins("JOIN teachers ON teachers.id = teacher_evaluations.teacher_id").
		Joins("LEFT JOIN courses ON courses.id = teacher_evaluations.course_id").
		Select(`
			teacher_evaluations.id,
			teacher_evaluations.teacher_id,
			teachers.name AS teacher_name,
			teacher_evaluations.course_id,
			courses.name AS course_name,
			teacher_evaluations.teaching_score,
			teacher_evaluations.grading_score,
			teacher_evaluations.attendance_score,
			ROUND((teacher_evaluations.teaching_score + teacher_evaluations.grading_score + teacher_evaluations.attendance_score) / 3.0, 2) AS avg_rating,
			teacher_evaluations.comment,
			teacher_evaluations.is_anonymous,
			teacher_evaluations.status,
			teacher_evaluations.created_at,
			teacher_evaluations.updated_at`).
		Order("teacher_evaluations.created_at DESC").
		Order("teacher_evaluations.id DESC").
		Offset((page - 1) * size).
		Limit(size).
		Scan(&items).Error
	if err != nil {
		return nil, 0, err
	}

	return items, total, nil
}

func (r *teacherRepository) CreateTeacherEvaluation(evaluation *model.TeacherEvaluations) error {
	return r.db.Create(evaluation).Error
}

func (r *teacherRepository) UpdateTeacherEvaluation(evaluation *model.TeacherEvaluations) error {
	return r.db.Model(&model.TeacherEvaluations{}).
		Where("id = ?", evaluation.ID).
		Updates(map[string]interface{}{
			"course_id":        evaluation.CourseID,
			"teaching_score":   evaluation.TeachingScore,
			"grading_score":    evaluation.GradingScore,
			"attendance_score": evaluation.AttendanceScore,
			"comment":          evaluation.Comment,
			"is_anonymous":     evaluation.IsAnonymous,
			"status":           evaluation.Status,
			"updated_at":       time.Now(),
		}).Error
}

func (r *teacherRepository) DeleteTeacherEvaluation(id int64) error {
	return r.db.Delete(&model.TeacherEvaluations{}, id).Error
}

func (r *teacherRepository) GetTeacherEvaluationByID(id int64) (*model.TeacherEvaluations, error) {
	var evaluation model.TeacherEvaluations
	err := r.db.First(&evaluation, id).Error
	if err != nil {
		return nil, err
	}
	return &evaluation, nil
}

func (r *teacherRepository) TeacherExists(id int64) (bool, error) {
	var count int64
	if err := r.db.Table("teachers").Where("id = ?", id).Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r *teacherRepository) CourseExists(id int64) (bool, error) {
	var count int64
	if err := r.db.Table("courses").Where("id = ?", id).Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r *teacherRepository) TeacherCourseRelationExists(teacherID, courseID int64) (bool, error) {
	var count int64
	if err := r.db.Table("course_teachers").Where("teacher_id = ? AND course_id = ?", teacherID, courseID).Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r *teacherRepository) RecalculateTeacherStats(teacherID int64) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec(`
			UPDATE teachers
			SET
				avg_teaching_score = stats.avg_teaching_score,
				avg_grading_score = stats.avg_grading_score,
				avg_attendance_score = stats.avg_attendance_score,
				approval_rate = stats.approval_rate,
				eval_count = stats.eval_count,
				updated_at = CURRENT_TIMESTAMP
			FROM (
				SELECT
					COALESCE(ROUND(AVG(teaching_score)::numeric, 2), 0) AS avg_teaching_score,
					COALESCE(ROUND(AVG(grading_score)::numeric, 2), 0) AS avg_grading_score,
					COALESCE(ROUND(AVG(attendance_score)::numeric, 2), 0) AS avg_attendance_score,
					COALESCE(ROUND(AVG(CASE WHEN (teaching_score + grading_score + attendance_score) / 3.0 >= 4 THEN 100 ELSE 0 END)::numeric, 2), 0) AS approval_rate,
					COUNT(*) AS eval_count
				FROM teacher_evaluations
				WHERE teacher_id = ? AND status = 'approved'
			) AS stats
			WHERE teachers.id = ?`, teacherID, teacherID, teacherID).Error; err != nil {
			return err
		}
		return nil
	})
}

func teacherSortExpr(sort string) string {
	switch sort {
	case "avg_quality":
		return "COALESCE(teachers.avg_teaching_score, 0)"
	case "avg_grading":
		return "COALESCE(teachers.avg_grading_score, 0)"
	case "avg_attendance":
		return "COALESCE(teachers.avg_attendance_score, 0)"
	case "good_rate":
		return "COALESCE(teachers.approval_rate, 0)"
	case "resource_count":
		return "COALESCE(teachers.resource_count, 0)"
	case "eval_count":
		return "COALESCE(teachers.eval_count, 0)"
	default:
		return "ROUND((COALESCE(teachers.avg_teaching_score, 0) + COALESCE(teachers.avg_grading_score, 0) + COALESCE(teachers.avg_attendance_score, 0)) / 3.0, 2)"
	}
}

func teacherEvaluationSortExpr(sort string) string {
	switch sort {
	case "avg_rating":
		return "ROUND((teacher_evaluations.teaching_score + teacher_evaluations.grading_score + teacher_evaluations.attendance_score) / 3.0, 2) DESC"
	case "likes":
		return `(
			SELECT COUNT(*)
			FROM likes
			WHERE likes.target_type = 'teacher_evaluation' AND likes.target_id = teacher_evaluations.id
		) DESC`
	default:
		return "teacher_evaluations.created_at DESC"
	}
}
