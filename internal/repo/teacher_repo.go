package repo

import (
	"csu-star-backend/internal/model"
	"fmt"
	"time"

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

type TeacherSimpleItem struct {
	ID         int64  `json:"id,string"`
	Name       string `json:"name"`
	Department string `json:"department,omitempty"`
}

type TeacherListItem struct {
	ID                 int64   `json:"id,string"`
	Name               string  `json:"name"`
	Title              string  `json:"title"`
	DepartmentID       int16   `json:"department_id"`
	DepartmentName     string  `json:"department_name"`
	AvatarURL          string  `json:"avatar_url"`
	AvgTeachingScore   float64 `json:"avg_quality"`
	AvgGradingScore    float64 `json:"avg_grading"`
	AvgAttendanceScore float64 `json:"avg_attendance"`
	AvgScore           float64 `json:"avg_score"`
	ApprovalRate       float64 `json:"good_rate"`
	EvalCount          int64   `json:"eval_count"`
	FavoriteCount      int64   `json:"favorite_count"`
}

type TeacherCourseItem struct {
	ID          int64   `json:"id,string"`
	Name        string  `json:"name"`
	Code        string  `json:"code,omitempty"`
	Credits     float64 `json:"-"`
	CourseType  string  `json:"-"`
	Description string  `json:"-"`
}

type TeacherMetadata struct {
	TutorType   string `json:"tutor_type,omitempty"`
	HomepageURL string `json:"homepage_url,omitempty"`
}

type TeacherDetail struct {
	ID                 int64               `json:"id,string"`
	Name               string              `json:"name"`
	Title              string              `json:"title"`
	DepartmentID       int16               `json:"department_id"`
	AvatarURL          string              `json:"avatar_url"`
	Bio                string              `json:"bio,omitempty"`
	Metadata           *TeacherMetadata    `json:"metadata,omitempty" gorm:"-"`
	TutorType          string              `json:"-" gorm:"column:tutor_type"`
	HomepageURL        string              `json:"-" gorm:"column:homepage_url"`
	AvgTeachingScore   float64             `json:"avg_quality"`
	AvgGradingScore    float64             `json:"avg_grading"`
	AvgAttendanceScore float64             `json:"avg_attendance"`
	AvgScore           float64             `json:"avg_score"`
	ApprovalRate       float64             `json:"good_rate"`
	EvalCount          int64               `json:"eval_count"`
	FavoriteCount      int64               `json:"favorite_count"`
	CreatedAt          time.Time           `json:"-"`
	UpdatedAt          time.Time           `json:"-"`
	Courses            []TeacherCourseItem `json:"courses" gorm:"-"`
	IsFavorited        bool                `json:"is_favorited"`
}

type TeacherEvaluationItem struct {
	ID                  int64             `json:"id,string"`
	UserID              int64             `json:"-"`
	User                *UserBrief        `json:"user,omitempty" gorm:"-"`
	TeacherID           int64             `json:"teacher_id,string"`
	CourseID            *int64            `json:"course_id,omitempty,string"`
	CourseName          string            `json:"course_name,omitempty"`
	Mode                string            `json:"mode"`
	MirrorEvaluationID  *int64            `json:"mirror_evaluation_id,omitempty,string"`
	MirrorEntityType    string            `json:"mirror_entity_type,omitempty"`
	TeachingScore       int               `json:"rating_quality"`
	GradingScore        int               `json:"rating_grading"`
	AttendanceScore     int               `json:"rating_attendance"`
	HomeworkScore       *int              `json:"rating_homework,omitempty"`
	GainScore           *int              `json:"rating_gain,omitempty"`
	ExamDifficultyScore *int              `json:"rating_exam_difficulty,omitempty"`
	AvgRating           float64           `json:"avg_rating"`
	Comment             string            `json:"comment"`
	IsAnonymous         bool              `json:"is_anonymous"`
	Status              string            `json:"status"`
	LikeCount           int64             `json:"likes"`
	IsLiked             bool              `json:"is_liked"`
	AuthorID            int64             `json:"-"`
	AuthorName          string            `json:"-"`
	AuthorAvatarURL     string            `json:"-"`
	AuthorRole          string            `json:"-"`
	ReplyCount          int64             `json:"reply_count"`
	Replies             []EvaluationReply `json:"replies,omitempty" gorm:"-"`
	CreatedAt           time.Time         `json:"created_at"`
	UpdatedAt           time.Time         `json:"updated_at"`
}

type MyTeacherEvaluationItem struct {
	ID                  int64     `json:"id,string"`
	TeacherID           int64     `json:"teacher_id,string"`
	TeacherName         string    `json:"teacher_name"`
	CourseID            *int64    `json:"course_id,omitempty,string"`
	CourseName          string    `json:"course_name,omitempty"`
	Mode                string    `json:"mode"`
	MirrorEvaluationID  *int64    `json:"mirror_evaluation_id,omitempty,string"`
	MirrorEntityType    string    `json:"mirror_entity_type,omitempty"`
	TeachingScore       int       `json:"rating_quality"`
	GradingScore        int       `json:"rating_grading"`
	AttendanceScore     int       `json:"rating_attendance"`
	HomeworkScore       *int      `json:"rating_homework,omitempty"`
	GainScore           *int      `json:"rating_gain,omitempty"`
	ExamDifficultyScore *int      `json:"rating_exam_difficulty,omitempty"`
	AvgRating           float64   `json:"avg_rating"`
	Comment             string    `json:"comment"`
	IsAnonymous         bool      `json:"is_anonymous"`
	Status              string    `json:"status"`
	CreatedAt           time.Time `json:"created_at"`
	UpdatedAt           time.Time `json:"updated_at"`
}

type TeacherRankingItem struct {
	ID             int64         `json:"id,string"`
	Name           string        `json:"name"`
	Title          string        `json:"title"`
	DepartmentID   int16         `json:"department_id"`
	DepartmentName string        `json:"department_name"`
	AvatarURL      string        `json:"avatar_url"`
	Score          float64       `json:"score"`
	AvgScore       float64       `json:"avg_score"`
	AvgQuality     float64       `json:"avg_quality"`
	AvgGrading     float64       `json:"avg_grading"`
	AvgAttendance  float64       `json:"avg_attendance"`
	EvalCount      int64         `json:"eval_count"`
	FavoriteCount  int64         `json:"favorite_count"`
	Rank           int64         `json:"rank"`
	DetailPath     string        `json:"detail_path,omitempty"`
	Courses        []CourseBrief `json:"courses,omitempty" gorm:"-"`
}

type RandomTeachers struct {
	Items []RandomTeacherItem `json:"items"`
}

type RandomTeacherItem struct {
	ID             int64   `json:"id,string"`
	Name           string  `json:"name"`
	Title          string  `json:"title"`
	DepartmentName string  `json:"department_name"`
	AvatarURL      string  `json:"avatar_url"`
	TutorType      string  `json:"tutor_type,omitempty"`
	AvgScore       float64 `json:"avg_score"`
	AvgQuality     float64 `json:"avg_quality"`
	AvgGrading     float64 `json:"avg_grading"`
	AvgAttendance  float64 `json:"avg_attendance"`
	GoodRate       float64 `json:"good_rate"`
	EvalCount      int64   `json:"eval_count"`
	FavoriteCount  int64   `json:"favorite_count"`
}

type TempRandomTeacherItem struct {
	ID             int64   `json:"id" gorm:"column:id"`
	Name           string  `json:"name" gorm:"column:name"`
	Title          string  `json:"title" gorm:"column:title"`
	DepartmentName string  `json:"department_name" gorm:"column:department_name"`
	AvatarURL      string  `json:"avatar_url" gorm:"column:avatar_url"`
	TutorType      string  `json:"tutor_type" gorm:"column:tutor_type"`
	AvgQuality     float64 `json:"avg_quality" gorm:"column:avg_quality"`
	AvgGrading     float64 `json:"avg_grading" gorm:"column:avg_grading"`
	AvgAttendance  float64 `json:"avg_attendance" gorm:"column:avg_attendance"`
	GoodRate       float64 `json:"good_rate" gorm:"column:good_rate"`
	EvalCount      int64   `json:"eval_count" gorm:"column:eval_count"`
	FavoriteCount  int64   `json:"favorite_count" gorm:"column:favorite_count"`
}

type TeacherRepository interface {
	FindTeachers(query TeacherListQuery) ([]TeacherListItem, int64, error)
	FindTeacherDetail(id int64) (*TeacherDetail, error)
	ListSimpleTeachers(q string, limit int) ([]TeacherSimpleItem, error)
	FindTeacherRankings(query TeacherRankingQuery) ([]TeacherRankingItem, int64, error)
	FindTeacherRankingItemsByIDs(ids []int64) ([]TeacherRankingItem, error)
	ListCourseBriefsByTeacherIDs(ids []int64) (map[int64][]CourseBrief, error)
	ListTeacherEvaluations(query TeacherEvaluationQuery) ([]TeacherEvaluationItem, int64, error)
	GetTeacherEvaluationItemByID(id int64) (*TeacherEvaluationItem, error)
	CreateTeacherEvaluationReply(reply *model.TeacherEvaluationReplies) error
	GetTeacherEvaluationReplyByID(id int64) (*model.TeacherEvaluationReplies, error)
	GetTeacherEvaluationReplyDetailByID(id int64) (*EvaluationReply, error)
	UpdateTeacherEvaluationReply(reply *model.TeacherEvaluationReplies) error
	DeleteTeacherEvaluationReply(id int64) error
	ListMyTeacherEvaluations(userID int64, page, size int) ([]MyTeacherEvaluationItem, int64, error)
	CreateTeacherEvaluation(evaluation *model.TeacherEvaluations) error
	UpdateTeacherEvaluation(evaluation *model.TeacherEvaluations) error
	DeleteTeacherEvaluation(id int64) error
	GetTeacherEvaluationByID(id int64) (*model.TeacherEvaluations, error)
	FindTeacherEvaluationByContext(userID, teacherID int64, courseID *int64, mode model.EvaluationMode) (*model.TeacherEvaluations, error)
	TeacherExists(id int64) (bool, error)
	CourseExists(id int64) (bool, error)
	TeacherCourseRelationExists(teacherID, courseID int64) (bool, error)
	GetCourseTeacherRelation(courseID, teacherID int64) (*model.CourseTeachers, error)
	CreateCourseTeacherRelation(relation *model.CourseTeachers) error
	UpdateCourseTeacherRelation(relation *model.CourseTeachers) error
	AdjustTeacherAggregates(teacherID int64, favoriteDelta, evalDelta int) error
	RecalculateTeacherStats(teacherID int64) error
	FindRandomTeachers(ids []int64) (RandomTeachers, error)
}

type teacherRepository struct {
	db *gorm.DB
}

func NewTeacherRepository(db *gorm.DB) TeacherRepository {
	return &teacherRepository{db: db}
}

func BuildTeacherMetadata(tutorType, homepageURL string) *TeacherMetadata {
	if tutorType == "" && homepageURL == "" {
		return nil
	}
	return &TeacherMetadata{
		TutorType:   tutorType,
		HomepageURL: homepageURL,
	}
}

func (r *teacherRepository) WithTx(tx *gorm.DB) TeacherRepository {
	return &teacherRepository{db: tx}
}

func (r *teacherRepository) FindTeachers(query TeacherListQuery) ([]TeacherListItem, int64, error) {
	var items []TeacherListItem
	var total int64

	base := r.db.Table("teachers").
		Joins("LEFT JOIN departments ON departments.id = teachers.department_id").
		Where("teachers.status = ?", "active")

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
			teachers.avatar_url,
			COALESCE(teachers.avg_teaching_score, 0) AS avg_teaching_score,
		COALESCE(teachers.avg_grading_score, 0) AS avg_grading_score,
		COALESCE(teachers.avg_attendance_score, 0) AS avg_attendance_score,
		ROUND((COALESCE(teachers.avg_teaching_score, 0) + COALESCE(teachers.avg_grading_score, 0) + COALESCE(teachers.avg_attendance_score, 0)) / 3.0, 2) AS avg_score,
		COALESCE(teachers.approval_rate, 0) AS approval_rate,
		COALESCE(teachers.eval_count, 0) AS eval_count,
		COALESCE(teachers.favorite_count, 0) AS favorite_count`).
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
			teachers.avatar_url,
			COALESCE(teachers.metadata->>'bio', '') AS bio,
			COALESCE(teachers.metadata->>'tutor_type', '') AS tutor_type,
			COALESCE(teachers.metadata->>'homepage_url', '') AS homepage_url,
			COALESCE(teachers.avg_teaching_score, 0) AS avg_teaching_score,
			COALESCE(teachers.avg_grading_score, 0) AS avg_grading_score,
			COALESCE(teachers.avg_attendance_score, 0) AS avg_attendance_score,
			ROUND((COALESCE(teachers.avg_teaching_score, 0) + COALESCE(teachers.avg_grading_score, 0) + COALESCE(teachers.avg_attendance_score, 0)) / 3.0, 2) AS avg_score,
			COALESCE(teachers.approval_rate, 0) AS approval_rate,
			COALESCE(teachers.eval_count, 0) AS eval_count,
			COALESCE(teachers.favorite_count, 0) AS favorite_count,
			teachers.created_at,
			teachers.updated_at`).
		Where("teachers.id = ? AND teachers.status = ?", id, "active").
		Scan(&detail).Error
	if err != nil {
		return nil, err
	}
	if detail.ID == 0 {
		return nil, gorm.ErrRecordNotFound
	}
	detail.Metadata = BuildTeacherMetadata(detail.TutorType, detail.HomepageURL)

	var courses []TeacherCourseItem
	err = r.db.Table("course_teachers").
		Joins("JOIN courses ON courses.id = course_teachers.course_id AND courses.status = ?", model.CourseStatusActive).
		Where("course_teachers.teacher_id = ? AND course_teachers.status = ?", id, model.CourseTeacherRelationStatusActive).
		Distinct("courses.id, courses.name, courses.credits, courses.course_type, courses.description").
		Order("courses.id ASC").
		Scan(&courses).Error
	if err != nil {
		return nil, err
	}
	detail.Courses = courses

	return &detail, nil
}

func (r *teacherRepository) ListSimpleTeachers(q string, limit int) ([]TeacherSimpleItem, error) {
	if limit <= 0 || limit > 50 {
		limit = 20
	}

	base := r.db.Table("teachers").
		Joins("LEFT JOIN departments ON departments.id = teachers.department_id").
		Where("teachers.status = ?", "active")
	if q != "" {
		base = base.Where("teachers.name ILIKE ?", "%"+q+"%")
	}

	var items []TeacherSimpleItem
	err := base.Select(`
		teachers.id,
		teachers.name,
		COALESCE(departments.name, '') AS department`).
		Order("COALESCE(teachers.favorite_count, 0) DESC").
		Order("COALESCE(teachers.eval_count, 0) DESC").
		Order("teachers.id ASC").
		Limit(limit).
		Scan(&items).Error
	return items, err
}

func (r *teacherRepository) FindTeacherRankings(query TeacherRankingQuery) ([]TeacherRankingItem, int64, error) {
	var total int64
	rankingBase := r.db.Table("teacher_rankings").
		Where("dimension = ? AND period = ?", query.RankType, query.Period)
	if query.DepartmentID > 0 {
		rankingBase = rankingBase.Where("department_id = ?", query.DepartmentID)
	}
	if query.IsIncreased {
		rankingBase = rankingBase.Where("score > 0")
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
			Joins("JOIN teachers ON teachers.id = teacher_rankings.teacher_id AND teachers.status = ?", "active").
			Joins("LEFT JOIN departments ON departments.id = teachers.department_id").
			Select(`
				teachers.id,
				teachers.name,
				teachers.title,
				teachers.department_id,
				departments.name AS department_name,
				teachers.avatar_url,
				ROUND((COALESCE(teachers.avg_teaching_score, 0) + COALESCE(teachers.avg_grading_score, 0) + COALESCE(teachers.avg_attendance_score, 0)) / 3.0, 2) AS avg_score,
				COALESCE(teachers.avg_teaching_score, 0) AS avg_quality,
				COALESCE(teachers.avg_grading_score, 0) AS avg_grading,
				COALESCE(teachers.avg_attendance_score, 0) AS avg_attendance,
				COALESCE(teachers.eval_count, 0) AS eval_count,
				COALESCE(teachers.favorite_count, 0) AS favorite_count,
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
			items[i].DetailPath = TeacherDetailPath(items[i].ID)
		}
		return items, total, nil
	}

	var items []TeacherRankingItem
	base := r.db.Table("teachers").
		Joins("LEFT JOIN departments ON departments.id = teachers.department_id").
		Where("teachers.status = ?", "active")
	if query.DepartmentID > 0 {
		base = base.Where("teachers.department_id = ?", query.DepartmentID)
	}
	if query.IsIncreased {
		base = base.Where(teacherSortExpr(query.RankType) + " > 0")
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
		teachers.avatar_url,
		ROUND((COALESCE(teachers.avg_teaching_score, 0) + COALESCE(teachers.avg_grading_score, 0) + COALESCE(teachers.avg_attendance_score, 0)) / 3.0, 2) AS avg_score,
		COALESCE(teachers.avg_teaching_score, 0) AS avg_quality,
		COALESCE(teachers.avg_grading_score, 0) AS avg_grading,
		COALESCE(teachers.avg_attendance_score, 0) AS avg_attendance,
		COALESCE(teachers.eval_count, 0) AS eval_count,
		COALESCE(teachers.favorite_count, 0) AS favorite_count,
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
		items[i].DetailPath = TeacherDetailPath(items[i].ID)
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
			departments.name AS department_name,
			teachers.avatar_url,
			ROUND((COALESCE(teachers.avg_teaching_score, 0) + COALESCE(teachers.avg_grading_score, 0) + COALESCE(teachers.avg_attendance_score, 0)) / 3.0, 2) AS avg_score,
			COALESCE(teachers.avg_teaching_score, 0) AS avg_quality,
			COALESCE(teachers.avg_grading_score, 0) AS avg_grading,
			COALESCE(teachers.avg_attendance_score, 0) AS avg_attendance,
			COALESCE(teachers.eval_count, 0) AS eval_count,
			COALESCE(teachers.favorite_count, 0) AS favorite_count`).
		Where("teachers.id IN ? AND teachers.status = ?", ids, "active").
		Scan(&items).Error
	if err != nil {
		return nil, err
	}
	for i := range items {
		items[i].DetailPath = TeacherDetailPath(items[i].ID)
	}
	return items, nil
}

func (r *teacherRepository) ListCourseBriefsByTeacherIDs(ids []int64) (map[int64][]CourseBrief, error) {
	result := make(map[int64][]CourseBrief)
	if len(ids) == 0 {
		return result, nil
	}

	type row struct {
		TeacherID  int64
		CourseID   int64
		Name       string
		CourseType string
	}

	var rows []row
	err := r.db.Table("course_teachers").
		Joins("JOIN courses ON courses.id = course_teachers.course_id AND courses.status = ?", model.CourseStatusActive).
		Where("course_teachers.teacher_id IN ? AND course_teachers.status = ?", ids, model.CourseTeacherRelationStatusActive).
		Select(`
			course_teachers.teacher_id,
			courses.id AS course_id,
			courses.name,
			courses.course_type`).
		Order("course_teachers.teacher_id ASC, courses.id ASC").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}

	for _, row := range rows {
		result[row.TeacherID] = append(result[row.TeacherID], CourseBrief{
			ID:                     row.CourseID,
			Name:                   row.Name,
			CourseType:             row.CourseType,
			DetailPath:             CourseDetailPath(row.CourseID),
			ResourceCollectionPath: CourseResourceCollectionPath(row.CourseID),
		})
	}
	return result, nil
}

func (r *teacherRepository) ListTeacherEvaluations(query TeacherEvaluationQuery) ([]TeacherEvaluationItem, int64, error) {
	var items []TeacherEvaluationItem
	var total int64

	base := applyVisibleTeacherEvaluationFilter(r.db.Table("teacher_evaluations"), "teacher_evaluations").
		Where("teacher_evaluations.teacher_id = ?", query.TeacherID)

	if err := base.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	err := base.
		Joins("JOIN users ON users.id = teacher_evaluations.user_id").
		Joins("LEFT JOIN courses ON courses.id = teacher_evaluations.course_id").
		Joins(`LEFT JOIN (
			SELECT target_id, COUNT(*) AS like_count
			FROM likes
			WHERE target_type = 'teacher_evaluation'
			GROUP BY target_id
		) AS eval_likes ON eval_likes.target_id = teacher_evaluations.id`).
		Select(`
			teacher_evaluations.id,
			teacher_evaluations.user_id,
			teacher_evaluations.teacher_id,
			teacher_evaluations.course_id,
			teacher_evaluations.mode,
			teacher_evaluations.mirror_evaluation_id,
			teacher_evaluations.mirror_entity_type,
			courses.name AS course_name,
			teacher_evaluations.teaching_score,
			teacher_evaluations.grading_score,
			teacher_evaluations.attendance_score,
			teacher_evaluations.workload_score AS homework_score,
			teacher_evaluations.gain_score,
			teacher_evaluations.difficulty_score AS exam_difficulty_score,
			ROUND((
				teacher_evaluations.teaching_score +
				teacher_evaluations.grading_score +
				teacher_evaluations.attendance_score +
				COALESCE(teacher_evaluations.workload_score, 0) +
				COALESCE(teacher_evaluations.gain_score, 0) +
				COALESCE(teacher_evaluations.difficulty_score, 0)
			) / CASE WHEN teacher_evaluations.mode = 'linked' THEN 6.0 ELSE 3.0 END, 2) AS avg_rating,
			teacher_evaluations.comment,
			teacher_evaluations.is_anonymous,
			teacher_evaluations.status,
			COALESCE(eval_likes.like_count, 0) AS like_count,
			FALSE AS is_liked,
			users.id AS author_id,
			users.nickname AS author_name,
			users.avatar_url AS author_avatar_url,
			users.role AS author_role,
			0 AS reply_count,
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
	replyMap, err := r.listTeacherEvaluationRepliesByEvaluationIDs(extractTeacherEvaluationIDs(items))
	if err != nil {
		return nil, 0, err
	}
	for i := range items {
		items[i].Replies = replyMap[items[i].ID]
		items[i].ReplyCount = int64(len(items[i].Replies))
	}

	return items, total, nil
}

func (r *teacherRepository) GetTeacherEvaluationItemByID(id int64) (*TeacherEvaluationItem, error) {
	var item TeacherEvaluationItem
	err := applyVisibleTeacherEvaluationFilter(r.db.Table("teacher_evaluations"), "teacher_evaluations").
		Joins("JOIN users ON users.id = teacher_evaluations.user_id").
		Joins("LEFT JOIN courses ON courses.id = teacher_evaluations.course_id").
		Joins(`LEFT JOIN (
			SELECT target_id, COUNT(*) AS like_count
			FROM likes
			WHERE target_type = 'teacher_evaluation'
			GROUP BY target_id
		) AS eval_likes ON eval_likes.target_id = teacher_evaluations.id`).
		Select(`
			teacher_evaluations.id,
			teacher_evaluations.user_id,
			teacher_evaluations.teacher_id,
			teacher_evaluations.course_id,
			teacher_evaluations.mode,
			teacher_evaluations.mirror_evaluation_id,
			teacher_evaluations.mirror_entity_type,
			courses.name AS course_name,
			teacher_evaluations.teaching_score,
			teacher_evaluations.grading_score,
			teacher_evaluations.attendance_score,
			teacher_evaluations.workload_score AS homework_score,
			teacher_evaluations.gain_score,
			teacher_evaluations.difficulty_score AS exam_difficulty_score,
			ROUND((
				teacher_evaluations.teaching_score +
				teacher_evaluations.grading_score +
				teacher_evaluations.attendance_score +
				COALESCE(teacher_evaluations.workload_score, 0) +
				COALESCE(teacher_evaluations.gain_score, 0) +
				COALESCE(teacher_evaluations.difficulty_score, 0)
			) / CASE WHEN teacher_evaluations.mode = 'linked' THEN 6.0 ELSE 3.0 END, 2) AS avg_rating,
			teacher_evaluations.comment,
			teacher_evaluations.is_anonymous,
			teacher_evaluations.status,
			COALESCE(eval_likes.like_count, 0) AS like_count,
			FALSE AS is_liked,
			users.id AS author_id,
			users.nickname AS author_name,
			users.avatar_url AS author_avatar_url,
			users.role AS author_role,
			teacher_evaluations.created_at,
			teacher_evaluations.updated_at`).
		Where("teacher_evaluations.id = ?", id).
		Scan(&item).Error
	if err != nil {
		return nil, err
	}
	if item.ID == 0 {
		return nil, gorm.ErrRecordNotFound
	}
	replyMap, err := r.listTeacherEvaluationRepliesByEvaluationIDs([]int64{id})
	if err != nil {
		return nil, err
	}
	item.Replies = replyMap[item.ID]
	item.ReplyCount = int64(len(item.Replies))
	return &item, nil
}

func (r *teacherRepository) CreateTeacherEvaluationReply(reply *model.TeacherEvaluationReplies) error {
	return r.db.Create(reply).Error
}

func (r *teacherRepository) GetTeacherEvaluationReplyByID(id int64) (*model.TeacherEvaluationReplies, error) {
	var reply model.TeacherEvaluationReplies
	if err := r.db.First(&reply, id).Error; err != nil {
		return nil, err
	}
	return &reply, nil
}

func (r *teacherRepository) GetTeacherEvaluationReplyDetailByID(id int64) (*EvaluationReply, error) {
	var reply EvaluationReply
	err := r.db.Table("teacher_evaluation_replies").
		Joins("JOIN users ON users.id = teacher_evaluation_replies.user_id").
		Joins("LEFT JOIN users AS reply_users ON reply_users.id = teacher_evaluation_replies.reply_to_user_id").
		Joins(`LEFT JOIN (
			SELECT target_id, COUNT(*) AS like_count
			FROM likes
			WHERE target_type = 'teacher_evaluation_reply'
			GROUP BY target_id
		) AS reply_likes ON reply_likes.target_id = teacher_evaluation_replies.id`).
		Select(`
			teacher_evaluation_replies.id,
			teacher_evaluation_replies.evaluation_id,
			teacher_evaluation_replies.user_id,
			teacher_evaluation_replies.content,
			teacher_evaluation_replies.is_anonymous,
			teacher_evaluation_replies.reply_to_reply_id,
			teacher_evaluation_replies.reply_to_user_id,
			COALESCE(reply_likes.like_count, 0) AS likes,
			FALSE AS is_liked,
			COALESCE(reply_users.nickname, '') AS reply_to_user_name,
			COALESCE(reply_users.role, 'user') AS reply_to_user_role,
			users.nickname AS author_name,
			users.avatar_url AS author_avatar,
			users.role AS author_role,
			teacher_evaluation_replies.created_at,
			teacher_evaluation_replies.updated_at`).
		Where("teacher_evaluation_replies.id = ?", id).
		Scan(&reply).Error
	if err != nil {
		return nil, err
	}
	if reply.ID == 0 {
		return nil, gorm.ErrRecordNotFound
	}
	return &reply, nil
}

func (r *teacherRepository) UpdateTeacherEvaluationReply(reply *model.TeacherEvaluationReplies) error {
	return r.db.Model(&model.TeacherEvaluationReplies{}).
		Where("id = ?", reply.ID).
		Updates(map[string]interface{}{
			"content":      reply.Content,
			"is_anonymous": reply.IsAnonymous,
			"updated_at":   time.Now(),
		}).Error
}

func (r *teacherRepository) DeleteTeacherEvaluationReply(id int64) error {
	return r.db.Delete(&model.TeacherEvaluationReplies{}, id).Error
}

func (r *teacherRepository) ListMyTeacherEvaluations(userID int64, page, size int) ([]MyTeacherEvaluationItem, int64, error) {
	var items []MyTeacherEvaluationItem
	var total int64

	base := applyVisibleTeacherEvaluationFilter(r.db.Table("teacher_evaluations"), "teacher_evaluations").
		Where("teacher_evaluations.user_id = ?", userID)
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
			teacher_evaluations.mode,
			teacher_evaluations.mirror_evaluation_id,
			teacher_evaluations.mirror_entity_type,
			courses.name AS course_name,
			teacher_evaluations.teaching_score,
			teacher_evaluations.grading_score,
			teacher_evaluations.attendance_score,
			teacher_evaluations.workload_score AS homework_score,
			teacher_evaluations.gain_score,
			teacher_evaluations.difficulty_score AS exam_difficulty_score,
			ROUND((
				teacher_evaluations.teaching_score +
				teacher_evaluations.grading_score +
				teacher_evaluations.attendance_score +
				COALESCE(teacher_evaluations.workload_score, 0) +
				COALESCE(teacher_evaluations.gain_score, 0) +
				COALESCE(teacher_evaluations.difficulty_score, 0)
			) / CASE WHEN teacher_evaluations.mode = 'linked' THEN 6.0 ELSE 3.0 END, 2) AS avg_rating,
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
			"course_id":            evaluation.CourseID,
			"mode":                 evaluation.Mode,
			"mirror_evaluation_id": evaluation.MirrorEvaluationID,
			"mirror_entity_type":   evaluation.MirrorEntityType,
			"teaching_score":       evaluation.TeachingScore,
			"grading_score":        evaluation.GradingScore,
			"attendance_score":     evaluation.AttendanceScore,
			"workload_score":       evaluation.HomeworkScore,
			"gain_score":           evaluation.GainScore,
			"difficulty_score":     evaluation.ExamDifficultyScore,
			"comment":              evaluation.Comment,
			"is_anonymous":         evaluation.IsAnonymous,
			"status":               evaluation.Status,
			"updated_at":           time.Now(),
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

func (r *teacherRepository) FindTeacherEvaluationByContext(userID, teacherID int64, courseID *int64, mode model.EvaluationMode) (*model.TeacherEvaluations, error) {
	var evaluation model.TeacherEvaluations
	query := r.db.Where("user_id = ? AND teacher_id = ? AND mode = ?", userID, teacherID, mode)
	if courseID == nil {
		query = query.Where("course_id IS NULL")
	} else {
		query = query.Where("course_id = ?", *courseID)
	}
	if err := query.First(&evaluation).Error; err != nil {
		return nil, err
	}
	return &evaluation, nil
}

func (r *teacherRepository) TeacherExists(id int64) (bool, error) {
	var count int64
	if err := r.db.Table("teachers").Where("id = ? AND status = ?", id, "active").Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r *teacherRepository) CourseExists(id int64) (bool, error) {
	var count int64
	if err := r.db.Table("courses").Where("id = ? AND status = ?", id, model.CourseStatusActive).Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r *teacherRepository) TeacherCourseRelationExists(teacherID, courseID int64) (bool, error) {
	var count int64
	if err := r.db.Table("course_teachers").
		Where("teacher_id = ? AND course_id = ? AND status = ?", teacherID, courseID, model.CourseTeacherRelationStatusActive).
		Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r *teacherRepository) GetCourseTeacherRelation(courseID, teacherID int64) (*model.CourseTeachers, error) {
	var relation model.CourseTeachers
	if err := r.db.Where("course_id = ? AND teacher_id = ?", courseID, teacherID).First(&relation).Error; err != nil {
		return nil, err
	}
	return &relation, nil
}

func (r *teacherRepository) CreateCourseTeacherRelation(relation *model.CourseTeachers) error {
	return r.db.Create(relation).Error
}

func (r *teacherRepository) UpdateCourseTeacherRelation(relation *model.CourseTeachers) error {
	return r.db.Save(relation).Error
}

func (r *teacherRepository) AdjustTeacherAggregates(teacherID int64, favoriteDelta, evalDelta int) error {
	updates := map[string]interface{}{
		"updated_at": time.Now(),
	}
	if favoriteDelta != 0 {
		updates["favorite_count"] = gorm.Expr("GREATEST(favorite_count + ?, 0)", favoriteDelta)
	}
	if evalDelta != 0 {
		updates["eval_count"] = gorm.Expr("GREATEST(eval_count + ?, 0)", evalDelta)
	}
	return r.db.Model(&model.Teachers{}).Where("id = ?", teacherID).Updates(updates).Error
}

func (r *teacherRepository) RecalculateTeacherStats(teacherID int64) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec(fmt.Sprintf(`
			UPDATE teachers
			SET
				avg_teaching_score = stats.avg_teaching_score,
				avg_grading_score = stats.avg_grading_score,
				avg_attendance_score = stats.avg_attendance_score,
				approval_rate = stats.approval_rate,
				eval_count = stats.eval_count,
				favorite_count = favorite_stats.favorite_count,
				updated_at = CURRENT_TIMESTAMP
			FROM (
				SELECT
					COALESCE(ROUND(AVG(teaching_score)::numeric, 2), 0) AS avg_teaching_score,
					COALESCE(ROUND(AVG(grading_score)::numeric, 2), 0) AS avg_grading_score,
					COALESCE(ROUND(AVG(attendance_score)::numeric, 2), 0) AS avg_attendance_score,
					COALESCE(ROUND(AVG(CASE WHEN (teaching_score + grading_score + attendance_score) / 3.0 >= 4 THEN 100 ELSE 0 END)::numeric, 2), 0) AS approval_rate,
					COUNT(*) AS eval_count
				FROM teacher_evaluations
				WHERE teacher_id = ? AND %s
			) AS stats,
			(
				SELECT COUNT(*) AS favorite_count
				FROM favorites
				WHERE target_type = 'teacher' AND target_id = ?
			) AS favorite_stats
			WHERE teachers.id = ?`, visibleTeacherEvaluationCondition("teacher_evaluations")), teacherID, teacherID, teacherID).Error; err != nil {
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
	case "favorite_count":
		return "COALESCE(teachers.favorite_count, 0)"
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

func (r *teacherRepository) listTeacherEvaluationRepliesByEvaluationIDs(ids []int64) (map[int64][]EvaluationReply, error) {
	result := make(map[int64][]EvaluationReply)
	if len(ids) == 0 {
		return result, nil
	}
	var replies []EvaluationReply
	err := r.db.Table("teacher_evaluation_replies").
		Joins("JOIN users ON users.id = teacher_evaluation_replies.user_id").
		Joins("LEFT JOIN users AS reply_users ON reply_users.id = teacher_evaluation_replies.reply_to_user_id").
		Select(`
			teacher_evaluation_replies.id,
			teacher_evaluation_replies.evaluation_id,
			teacher_evaluation_replies.user_id,
			teacher_evaluation_replies.content,
			teacher_evaluation_replies.is_anonymous,
			teacher_evaluation_replies.reply_to_reply_id,
			teacher_evaluation_replies.reply_to_user_id,
			COALESCE((
				SELECT COUNT(*)
				FROM likes
				WHERE likes.target_type = 'teacher_evaluation_reply' AND likes.target_id = teacher_evaluation_replies.id
			), 0) AS likes,
			FALSE AS is_liked,
			COALESCE(reply_users.nickname, '') AS reply_to_user_name,
			COALESCE(reply_users.role, 'user') AS reply_to_user_role,
			users.nickname AS author_name,
			users.avatar_url AS author_avatar,
			users.role AS author_role,
			teacher_evaluation_replies.created_at,
			teacher_evaluation_replies.updated_at`).
		Where("teacher_evaluation_replies.evaluation_id IN ?", ids).
		Order("teacher_evaluation_replies.created_at ASC, teacher_evaluation_replies.id ASC").
		Scan(&replies).Error
	if err != nil {
		return nil, err
	}
	for _, reply := range replies {
		result[reply.EvaluationID] = append(result[reply.EvaluationID], reply)
	}
	return result, nil
}

func (r *teacherRepository) FindRandomTeachers(ids []int64) (RandomTeachers, error) {
	var tempRandomTeacherItems []TempRandomTeacherItem
	teacherMap := make(map[int64]*RandomTeacherItem)
	var randomTeachers RandomTeachers

	err := r.db.Table("teachers t").
		Select(fmt.Sprintf(`
		t.id,
		t.name,
		t.title,
		(SELECT departments.name FROM departments
		WHERE t.department_id = departments.id) AS department_name,
		t.avatar_url,
		COALESCE(t.metadata->>'tutor_type', '') AS tutor_type,
		t.avg_teaching_score AS avg_quality,
		t.avg_grading_score AS avg_grading,
		t.avg_attendance_score AS avg_attendance,
		t.approval_rate AS good_rate,
		(SELECT COUNT(*) FROM teacher_evaluations 
		WHERE t.id = teacher_evaluations.teacher_id
			AND %s) AS eval_count,
		COALESCE(t.favorite_count, 0) AS favorite_count
		`, visibleTeacherEvaluationCondition("teacher_evaluations"))).
		Where("t.id IN ? AND t.status = ?", ids, "active").
		Find(&tempRandomTeacherItems).Error
	if err != nil {
		return randomTeachers, err
	}

	for _, row := range tempRandomTeacherItems {
		teacherItem, exists := teacherMap[row.ID]
		if !exists {
			teacherItem = &RandomTeacherItem{
				ID:             row.ID,
				Name:           row.Name,
				Title:          row.Title,
				DepartmentName: row.DepartmentName,
				AvatarURL:      row.AvatarURL,
				TutorType:      row.TutorType,
				AvgScore:       (row.AvgAttendance + row.AvgGrading + row.AvgQuality) / 3.0,
				AvgGrading:     row.AvgGrading,
				AvgAttendance:  row.AvgAttendance,
				AvgQuality:     row.AvgQuality,
				GoodRate:       row.GoodRate,
				EvalCount:      row.EvalCount,
				FavoriteCount:  row.FavoriteCount,
			}
			teacherMap[row.ID] = teacherItem
		}
	}

	for _, item := range teacherMap {
		randomTeachers.Items = append(randomTeachers.Items, *item)
	}

	return randomTeachers, nil
}

func extractTeacherEvaluationIDs(items []TeacherEvaluationItem) []int64 {
	ids := make([]int64, 0, len(items))
	for _, item := range items {
		ids = append(ids, item.ID)
	}
	return ids
}
