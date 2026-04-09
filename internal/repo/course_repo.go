package repo

import (
	"csu-star-backend/internal/model"
	"fmt"
	"math/rand"
	"strconv"
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

type CourseResourceCollectionQuery struct {
	CourseID     int64
	Sort         string
	ResourceType string
	Page         int
	Size         int
}

func courseFavoriteCountExpr(courseTable string) string {
	return "COALESCE(" + courseTable + ".favorite_count, 0)"
}

func courseResourceFavoriteTotalExpr(courseTable string) string {
	return "COALESCE(" + courseTable + ".resource_favorite_count, 0)"
}

type CourseListItem struct {
	ID                 int64   `json:"id,string"`
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
	FavoriteCount      int     `json:"favorite_count"`
}

type CourseSimpleItem struct {
	ID   int64  `json:"id,string"`
	Name string `json:"name"`
}

type CourseTeacherItem struct {
	ID           int64  `json:"id,string"`
	Name         string `json:"name"`
	Title        string `json:"title"`
	DepartmentID int16  `json:"-"`
	AvatarURL    string `json:"-"`
}

type CourseDetail struct {
	ID                 int64               `json:"id,string"`
	Code               string              `json:"code,omitempty"`
	Name               string              `json:"name"`
	Credits            float64             `json:"credits"`
	CourseType         string              `json:"course_type"`
	Description        string              `json:"description"`
	Metadata           datatypes.JSON      `json:"-"`
	AvgWorkloadScore   float64             `json:"avg_homework"`
	AvgGainScore       float64             `json:"avg_gain"`
	AvgDifficultyScore float64             `json:"avg_exam_diff"`
	AvgScore           float64             `json:"avg_score"`
	ResourceCount      int                 `json:"resource_count"`
	EvalCount          int                 `json:"eval_count"`
	FavoriteCount      int                 `json:"favorite_count"`
	CreatedAt          time.Time           `json:"-"`
	UpdatedAt          time.Time           `json:"-"`
	Teachers           []CourseTeacherItem `json:"teachers" gorm:"-"`
	IsFavorited        bool                `json:"is_favorited"`
}

type CourseEvaluationItem struct {
	ID                 int64             `json:"id,string"`
	UserID             int64             `json:"-"`
	User               *UserBrief        `json:"user,omitempty" gorm:"-"`
	CourseID           int64             `json:"course_id,string"`
	TeacherID          *int64            `json:"teacher_id,omitempty,string"`
	TeacherName        string            `json:"teacher_name,omitempty"`
	Mode               string            `json:"mode"`
	MirrorEvaluationID *int64            `json:"mirror_evaluation_id,omitempty,string"`
	MirrorEntityType   string            `json:"mirror_entity_type,omitempty"`
	WorkloadScore      int               `json:"rating_homework"`
	GainScore          int               `json:"rating_gain"`
	DifficultyScore    int               `json:"rating_exam_difficulty"`
	TeachingScore      *int              `json:"rating_quality,omitempty"`
	GradingScore       *int              `json:"rating_grading,omitempty"`
	AttendanceScore    *int              `json:"rating_attendance,omitempty"`
	AvgRating          float64           `json:"avg_rating"`
	Comment            string            `json:"comment"`
	IsAnonymous        bool              `json:"is_anonymous"`
	Status             string            `json:"status"`
	LikeCount          int64             `json:"likes"`
	IsLiked            bool              `json:"is_liked"`
	AuthorID           int64             `json:"-"`
	AuthorName         string            `json:"-"`
	AuthorAvatarURL    string            `json:"-"`
	AuthorRole         string            `json:"-"`
	ReplyCount         int64             `json:"reply_count"`
	Replies            []EvaluationReply `json:"replies,omitempty" gorm:"-"`
	CreatedAt          time.Time         `json:"created_at"`
	UpdatedAt          time.Time         `json:"updated_at"`
}

type MyCourseEvaluationItem struct {
	ID                 int64     `json:"id,string"`
	CourseID           int64     `json:"course_id,string"`
	CourseName         string    `json:"course_name"`
	TeacherID          *int64    `json:"teacher_id,omitempty,string"`
	TeacherName        string    `json:"teacher_name,omitempty"`
	Mode               string    `json:"mode"`
	MirrorEvaluationID *int64    `json:"mirror_evaluation_id,omitempty,string"`
	MirrorEntityType   string    `json:"mirror_entity_type,omitempty"`
	WorkloadScore      int       `json:"rating_homework"`
	GainScore          int       `json:"rating_gain"`
	DifficultyScore    int       `json:"rating_exam_difficulty"`
	TeachingScore      *int      `json:"rating_quality,omitempty"`
	GradingScore       *int      `json:"rating_grading,omitempty"`
	AttendanceScore    *int      `json:"rating_attendance,omitempty"`
	AvgRating          float64   `json:"avg_rating"`
	Comment            string    `json:"comment"`
	IsAnonymous        bool      `json:"is_anonymous"`
	Status             string    `json:"status"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}

type CourseRankingItem struct {
	ID            int64          `json:"id,string"`
	Name          string         `json:"name"`
	CourseType    string         `json:"course_type"`
	Score         float64        `json:"score"`
	AvgScore      float64        `json:"avg_score"`
	AvgHomework   float64        `json:"avg_homework"`
	AvgGain       float64        `json:"avg_gain"`
	AvgExamDiff   float64        `json:"avg_exam_diff"`
	EvalCount     int64          `json:"eval_count"`
	ResourceCount int64          `json:"resource_count"`
	FavoriteCount int64          `json:"favorite_count"`
	Rank          int64          `json:"rank"`
	DetailPath    string         `json:"detail_path,omitempty"`
	Teachers      []TeacherBrief `json:"teachers,omitempty" gorm:"-"`
}

type RandomCourses struct {
	Items []RandomCourseItem `json:"items"`
}

type RandomCourseItem struct {
	ID            int64          `json:"id,string"`
	Name          string         `json:"name"`
	CourseType    string         `json:"course_type"`
	AvgScore      float64        `json:"avg_score,omitempty"`
	AvgHomework   float64        `json:"avg_homework,omitempty"`
	AvgGain       float64        `json:"avg_gain,omitempty"`
	AvgExamDiff   float64        `json:"avg_exam_diff,omitempty"`
	EvalCount     int            `json:"eval_count,omitempty"`
	ResourceCount int            `json:"resource_count,omitempty"`
	TeacherCount  int            `json:"teacher_count,omitempty"`
	Teachers      []TeacherBrief `json:"teachers,omitempty"`
	CacheTTLSec   int            `json:"cache_ttl_sec,omitempty"`
}

type TempRandomCourseItem struct {
	ID               int64            `json:"id" gorm:"column:id"`
	Name             string           `json:"name" gorm:"column:name"`
	CourseType       model.CourseType `json:"course_type" gorm:"column:course_type;type:course_type"`
	AvgHomework      float64          `json:"avg_homework" gorm:"column:avg_homework"`
	AvgGain          float64          `json:"avg_gain" gorm:"column:avg_gain"`
	AvgExamDiff      float64          `json:"avg_exam_diff" gorm:"column:avg_exam_diff"`
	EvalCount        int              `json:"eval_count" gorm:"column:eval_count"`
	ResourceCount    int              `json:"resource_count" gorm:"column:resource_count"`
	TeacherID        *int64           `json:"teacher_id" gorm:"column:teacher_id"`
	TeacherName      string           `json:"teacher_name" gorm:"column:teacher_name"`
	TeacherTitle     string           `json:"teacher_title" gorm:"column:teacher_title"`
	TeacherAvatarUrl string           `json:"teacher_avatar_url" gorm:"column:teacher_avatar_url"`
	TeacherMeta      datatypes.JSON   `json:"teacher_meta" gorm:"column:teacher_meta;type:json"`
}

type CourseRepository interface {
	FindCourses(query CourseListQuery) ([]CourseListItem, int64, error)
	ListSimpleCourses(q string, limit int) ([]CourseSimpleItem, error)
	FindCourseDetail(id int64) (*CourseDetail, error)
	FindCourseResourceCollectionDetail(query CourseResourceCollectionQuery) (*CourseResourceCollectionDetail, error)
	FindCourseRankings(query CourseRankingQuery) ([]CourseRankingItem, int64, error)
	FindCourseRankingItemsByIDs(ids []int64) ([]CourseRankingItem, error)
	ListTeacherBriefsByCourseIDs(ids []int64) (map[int64][]TeacherBrief, error)
	ListCourseEvaluations(query CourseEvaluationQuery) ([]CourseEvaluationItem, int64, error)
	GetCourseEvaluationItemByID(id int64) (*CourseEvaluationItem, error)
	CreateCourseEvaluationReply(reply *model.CourseEvaluationReplies) error
	GetCourseEvaluationReplyByID(id int64) (*model.CourseEvaluationReplies, error)
	GetCourseEvaluationReplyDetailByID(id int64) (*EvaluationReply, error)
	UpdateCourseEvaluationReply(reply *model.CourseEvaluationReplies) error
	DeleteCourseEvaluationReply(id int64) error
	ListMyCourseEvaluations(userID int64, page, size int) ([]MyCourseEvaluationItem, int64, error)
	CreateCourseEvaluation(evaluation *model.CourseEvaluations) error
	UpdateCourseEvaluation(evaluation *model.CourseEvaluations) error
	DeleteCourseEvaluation(id int64) error
	GetCourseEvaluationByID(id int64) (*model.CourseEvaluations, error)
	FindCourseEvaluationByContext(userID, courseID int64, teacherID *int64, mode model.EvaluationMode) (*model.CourseEvaluations, error)
	CourseExists(id int64) (bool, error)
	ListRandomCourseIDs(limit int) ([]int64, error)
	AdjustCourseAggregates(courseID int64, resourceDelta, downloadDelta, viewDelta, likeDelta, favoriteDelta, resourceFavoriteDelta, evalDelta int) error
	RecalculateCourseStats(courseID int64) error
	FindRandomCourses(ids []int64) (RandomCourses, error)
}

type courseRepository struct {
	db *gorm.DB
}

func NewCourseRepository(db *gorm.DB) CourseRepository {
	return &courseRepository{db: db}
}

func (r *courseRepository) WithTx(tx *gorm.DB) CourseRepository {
	return &courseRepository{db: tx}
}

func (r *courseRepository) FindCourses(query CourseListQuery) ([]CourseListItem, int64, error) {
	var items []CourseListItem
	var total int64

	base := r.db.Table("courses").Where("courses.status = ?", model.CourseStatusActive)

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

	err := base.Select(fmt.Sprintf(`
		courses.id,
		courses.code,
		courses.name,
		courses.credits,
		courses.course_type,
		courses.description,
		COALESCE(courses.avg_workload_score, 0) AS avg_workload_score,
		COALESCE(courses.avg_gain_score, 0) AS avg_gain_score,
		COALESCE(courses.avg_difficulty_score, 0) AS avg_difficulty_score,
		ROUND((COALESCE(courses.avg_workload_score, 0) + COALESCE(courses.avg_gain_score, 0) + COALESCE(courses.avg_difficulty_score, 0)) / 3.0, 2) AS avg_score,
		COALESCE(courses.resource_count, 0) AS resource_count,
		COALESCE(courses.eval_count, 0) AS eval_count,
		%s AS favorite_count`, courseFavoriteCountExpr("courses"))).
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

func (r *courseRepository) ListSimpleCourses(q string, limit int) ([]CourseSimpleItem, error) {
	if limit <= 0 || limit > 50 {
		limit = 20
	}

	base := r.db.Table("courses").Where("status = ?", model.CourseStatusActive)
	if q != "" {
		base = base.Where("name ILIKE ?", "%"+q+"%")
	}

	var items []CourseSimpleItem
	err := base.Select("id, name").
		Order("COALESCE(favorite_count, 0) DESC").
		Order("COALESCE(eval_count, 0) DESC").
		Order("id ASC").
		Limit(limit).
		Scan(&items).Error
	return items, err
}

func (r *courseRepository) FindCourseDetail(id int64) (*CourseDetail, error) {
	var detail CourseDetail
	err := r.db.Table("courses").
		Select(fmt.Sprintf(`
			courses.id,
			courses.name,
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
			%s AS favorite_count,
			courses.created_at,
			courses.updated_at`, courseFavoriteCountExpr("courses"))).
		Where("courses.id = ? AND courses.status = ?", id, model.CourseStatusActive).
		Scan(&detail).Error
	if err != nil {
		return nil, err
	}
	if detail.ID == 0 {
		return nil, gorm.ErrRecordNotFound
	}

	var teachers []CourseTeacherItem
	err = r.db.Table("course_teachers").
		Joins("JOIN teachers ON teachers.id = course_teachers.teacher_id AND teachers.status = 'active'").
		Where("course_teachers.course_id = ? AND course_teachers.status = ?", id, model.CourseTeacherRelationStatusActive).
		Distinct("teachers.id, teachers.name, teachers.title, teachers.department_id, teachers.avatar_url").
		Order("teachers.id ASC").
		Scan(&teachers).Error
	if err != nil {
		return nil, err
	}
	detail.Teachers = teachers

	return &detail, nil
}

func (r *courseRepository) FindCourseResourceCollectionDetail(query CourseResourceCollectionQuery) (*CourseResourceCollectionDetail, error) {
	var course CourseBrief
	err := r.db.Table("courses").
		Select("id, name, credits, course_type").
		Where("id = ? AND status = ?", query.CourseID, model.CourseStatusActive).
		Scan(&course).Error
	if err != nil {
		return nil, err
	}
	if course.ID == 0 {
		return nil, gorm.ErrRecordNotFound
	}
	course.DetailPath = CourseDetailPath(course.ID)
	course.Code = course.Code
	course.ResourceCollectionPath = CourseResourceCollectionPath(course.ID)

	detail := &CourseResourceCollectionDetail{
		Course:           course,
		EvaluationAnchor: CourseEvaluationAnchorPath(query.CourseID),
	}

	type statsRow struct {
		ResourceCount int
		DownloadCount int
		LikeCount     int
		FavoriteCount int
	}
	var stats statsRow
	err = r.db.Table("courses").
		Select(fmt.Sprintf(`
			COALESCE(resource_count, 0) AS resource_count,
			COALESCE(download_total, 0) AS download_count,
			COALESCE(like_total, 0) AS like_count,
			%s AS favorite_count`, courseResourceFavoriteTotalExpr("courses"))).
		Where("id = ?", query.CourseID).
		Scan(&stats).Error
	if err != nil {
		return nil, err
	}
	detail.ResourceCount = stats.ResourceCount
	detail.DownloadCount = stats.DownloadCount
	detail.LikeCount = stats.LikeCount
	detail.FavoriteCount = stats.FavoriteCount

	var total int64
	base := r.db.Table("resources").Where("course_id = ? AND status = ?", query.CourseID, model.ResourceStatusApproved)
	if query.ResourceType != "" {
		base = base.Where("type = ?", query.ResourceType)
	}
	if err := base.Count(&total).Error; err != nil {
		return nil, err
	}
	var resources []ResourceCard
	if query.Sort == "" {
		query.Sort = "created_at"
	}
	if query.Page <= 0 {
		query.Page = 1
	}
	if query.Size <= 0 {
		query.Size = 20
	}
	err = base.
		Select(`
			id,
			title,
			description,
			type,
			download_count,
			like_count,
			comment_count,
			view_count,
			created_at,
			updated_at`).
		Order(resourceSortExpr(query.Sort)).
		Offset((query.Page - 1) * query.Size).
		Limit(query.Size).
		Scan(&resources).Error
	if err != nil {
		return nil, err
	}
	for i := range resources {
		resources[i].DetailPath = ResourceDetailPath(resources[i].ID)
	}
	if err := r.attachResourceCardFilePreview(resources); err != nil {
		return nil, err
	}
	detail.Items = CourseResourceCollectionItemsPage{
		Items: resources,
		Total: total,
	}

	return detail, nil
}

func (r *courseRepository) attachResourceCardFilePreview(resources []ResourceCard) error {
	if len(resources) == 0 {
		return nil
	}

	resourceIDs := make([]int64, 0, len(resources))
	indexByResourceID := make(map[int64]int, len(resources))
	for index, resource := range resources {
		resourceIDs = append(resourceIDs, resource.ID)
		indexByResourceID[resource.ID] = index
	}

	type resourceFilePreviewRow struct {
		ResourceID int64  `gorm:"column:resource_id"`
		Filename   string `gorm:"column:filename"`
		MimeType   string `gorm:"column:mime_type"`
		FileSize   int64  `gorm:"column:file_size"`
		FileCount  int    `gorm:"column:file_count"`
	}

	var rows []resourceFilePreviewRow
	if err := r.db.Table("(?) AS ranked_files",
		r.db.Table("resource_files").
			Select(`
				resource_id,
				filename,
				mime_type,
				file_size,
				COUNT(*) OVER (PARTITION BY resource_id) AS file_count,
				ROW_NUMBER() OVER (PARTITION BY resource_id ORDER BY id ASC) AS row_num`).
			Where("resource_id IN ?", resourceIDs),
	).
		Select("resource_id, filename, mime_type, file_size, file_count").
		Where("row_num = 1").
		Scan(&rows).Error; err != nil {
		return err
	}

	for _, row := range rows {
		index, ok := indexByResourceID[row.ResourceID]
		if !ok {
			continue
		}

		resources[index].FileCount = row.FileCount
		resources[index].FirstFile = &ResourceCardFilePreview{
			Filename:  row.Filename,
			Mime:      row.MimeType,
			SizeBytes: row.FileSize,
		}
	}

	type resourceFavoriteCountRow struct {
		TargetID      int64 `gorm:"column:target_id"`
		FavoriteCount int   `gorm:"column:favorite_count"`
	}

	var favoriteRows []resourceFavoriteCountRow
	if err := r.db.Table("favorites").
		Select("target_id, COUNT(*) AS favorite_count").
		Where("target_type = ? AND target_id IN ?", model.FavoriteTargetTypeResource, resourceIDs).
		Group("target_id").
		Scan(&favoriteRows).Error; err != nil {
		return err
	}

	for _, row := range favoriteRows {
		index, ok := indexByResourceID[row.TargetID]
		if !ok {
			continue
		}
		resources[index].FavoriteCount = row.FavoriteCount
	}

	return nil
}

func (r *courseRepository) FindCourseRankings(query CourseRankingQuery) ([]CourseRankingItem, int64, error) {
	var total int64
	rankingBase := r.db.Table("course_rankings").
		Where("dimension = ? AND period = ?", query.RankType, query.Period)
	if query.IsIncreased {
		rankingBase = rankingBase.Where("score > 0")
	}
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
			Joins("JOIN courses ON courses.id = course_rankings.course_id AND courses.status = ?", model.CourseStatusActive).
			Select(fmt.Sprintf(`
				courses.id,
				courses.name,
				courses.course_type,
				ROUND((COALESCE(courses.avg_workload_score, 0) + COALESCE(courses.avg_gain_score, 0) + COALESCE(courses.avg_difficulty_score, 0)) / 3.0, 2) AS avg_score,
				COALESCE(courses.avg_workload_score, 0) AS avg_homework,
				COALESCE(courses.avg_gain_score, 0) AS avg_gain,
				COALESCE(courses.avg_difficulty_score, 0) AS avg_exam_diff,
				COALESCE(courses.eval_count, 0) AS eval_count,
				COALESCE(courses.resource_count, 0) AS resource_count,
				%s AS favorite_count,
				COALESCE(course_rankings.score, 0) AS score`, courseFavoriteCountExpr("courses"))).
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
			items[i].DetailPath = CourseDetailPath(items[i].ID)
		}
		return items, total, nil
	}

	base := r.db.Table("courses").Where("courses.status = ?", model.CourseStatusActive)
	if query.IsIncreased {
		base = base.Where(courseRankingExpr(query.RankType) + " > 0")
	}
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
		courses.course_type,
		ROUND((COALESCE(courses.avg_workload_score, 0) + COALESCE(courses.avg_gain_score, 0) + COALESCE(courses.avg_difficulty_score, 0)) / 3.0, 2) AS avg_score,
		COALESCE(courses.avg_workload_score, 0) AS avg_homework,
		COALESCE(courses.avg_gain_score, 0) AS avg_gain,
		COALESCE(courses.avg_difficulty_score, 0) AS avg_exam_diff,
		COALESCE(courses.eval_count, 0) AS eval_count,
		COALESCE(courses.resource_count, 0) AS resource_count,
		%s AS favorite_count,
		%s AS score`, courseFavoriteCountExpr("courses"), courseRankingExpr(query.RankType))).
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
		items[i].DetailPath = CourseDetailPath(items[i].ID)
	}

	return items, total, nil
}

func (r *courseRepository) FindCourseRankingItemsByIDs(ids []int64) ([]CourseRankingItem, error) {
	if len(ids) == 0 {
		return []CourseRankingItem{}, nil
	}

	var items []CourseRankingItem
	err := r.db.Table("courses").
		Select(fmt.Sprintf(`
			courses.id,
			courses.name,
			courses.course_type,
			ROUND((COALESCE(courses.avg_workload_score, 0) + COALESCE(courses.avg_gain_score, 0) + COALESCE(courses.avg_difficulty_score, 0)) / 3.0, 2) AS avg_score,
			COALESCE(courses.avg_workload_score, 0) AS avg_homework,
			COALESCE(courses.avg_gain_score, 0) AS avg_gain,
			COALESCE(courses.avg_difficulty_score, 0) AS avg_exam_diff,
			COALESCE(courses.eval_count, 0) AS eval_count,
			COALESCE(courses.resource_count, 0) AS resource_count,
			%s AS favorite_count`, courseFavoriteCountExpr("courses"))).
		Where("courses.id IN ? AND courses.status = ?", ids, model.CourseStatusActive).
		Scan(&items).Error
	if err != nil {
		return nil, err
	}
	for i := range items {
		items[i].DetailPath = CourseDetailPath(items[i].ID)
	}
	return items, nil
}

func (r *courseRepository) ListTeacherBriefsByCourseIDs(ids []int64) (map[int64][]TeacherBrief, error) {
	result := make(map[int64][]TeacherBrief)
	if len(ids) == 0 {
		return result, nil
	}

	type row struct {
		CourseID  int64
		TeacherID int64
		Name      string
		Title     string
		AvatarURL string
	}

	var rows []row
	err := r.db.Table("course_teachers").
		Joins("JOIN teachers ON teachers.id = course_teachers.teacher_id AND teachers.status = 'active'").
		Where("course_teachers.course_id IN ? AND course_teachers.status = ?", ids, model.CourseTeacherRelationStatusActive).
		Select(`
			course_teachers.course_id,
			teachers.id AS teacher_id,
			teachers.name,
			teachers.title,
			teachers.avatar_url`).
		Order("course_teachers.course_id ASC, teachers.id ASC").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}

	for _, row := range rows {
		result[row.CourseID] = append(result[row.CourseID], TeacherBrief{
			ID:         row.TeacherID,
			Name:       row.Name,
			Title:      row.Title,
			AvatarURL:  row.AvatarURL,
			DetailPath: TeacherDetailPath(row.TeacherID),
		})
	}
	return result, nil
}

func (r *courseRepository) ListCourseEvaluations(query CourseEvaluationQuery) ([]CourseEvaluationItem, int64, error) {
	var items []CourseEvaluationItem
	var total int64

	base := applyVisibleCourseEvaluationFilter(r.db.Table("course_evaluations"), "course_evaluations").
		Where("course_evaluations.course_id = ?", query.CourseID)

	if err := base.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	err := base.
		Joins("JOIN users ON users.id = course_evaluations.user_id").
		Joins("LEFT JOIN teachers ON teachers.id = course_evaluations.teacher_id").
		Joins(`LEFT JOIN (
			SELECT target_id, COUNT(*) AS like_count
			FROM likes
			WHERE target_type = 'course_evaluation'
			GROUP BY target_id
		) AS eval_likes ON eval_likes.target_id = course_evaluations.id`).
		Select(`
			course_evaluations.id,
			course_evaluations.user_id,
			course_evaluations.course_id,
			course_evaluations.teacher_id,
			teachers.name AS teacher_name,
			course_evaluations.mode,
			course_evaluations.mirror_evaluation_id,
			course_evaluations.mirror_entity_type,
			course_evaluations.workload_score,
			course_evaluations.gain_score,
			course_evaluations.difficulty_score,
			course_evaluations.teaching_score,
			course_evaluations.grading_score,
			course_evaluations.attendance_score,
			ROUND((
				course_evaluations.workload_score +
				course_evaluations.gain_score +
				course_evaluations.difficulty_score +
				COALESCE(course_evaluations.teaching_score, 0) +
				COALESCE(course_evaluations.grading_score, 0) +
				COALESCE(course_evaluations.attendance_score, 0)
			) / CASE WHEN course_evaluations.mode = 'linked' THEN 6.0 ELSE 3.0 END, 2) AS avg_rating,
			course_evaluations.comment,
			course_evaluations.is_anonymous,
			course_evaluations.status,
			COALESCE(eval_likes.like_count, 0) AS like_count,
			FALSE AS is_liked,
			users.id AS author_id,
			users.nickname AS author_name,
			users.avatar_url AS author_avatar_url,
			users.role AS author_role,
			0 AS reply_count,
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
	replyMap, err := r.listCourseEvaluationRepliesByEvaluationIDs(extractCourseEvaluationIDs(items))
	if err != nil {
		return nil, 0, err
	}
	for i := range items {
		items[i].Replies = replyMap[items[i].ID]
		items[i].ReplyCount = int64(len(items[i].Replies))
	}

	return items, total, nil
}

func (r *courseRepository) GetCourseEvaluationItemByID(id int64) (*CourseEvaluationItem, error) {
	var item CourseEvaluationItem
	err := applyVisibleCourseEvaluationFilter(r.db.Table("course_evaluations"), "course_evaluations").
		Joins("JOIN users ON users.id = course_evaluations.user_id").
		Joins("LEFT JOIN teachers ON teachers.id = course_evaluations.teacher_id").
		Joins(`LEFT JOIN (
			SELECT target_id, COUNT(*) AS like_count
			FROM likes
			WHERE target_type = 'course_evaluation'
			GROUP BY target_id
		) AS eval_likes ON eval_likes.target_id = course_evaluations.id`).
		Select(`
			course_evaluations.id,
			course_evaluations.user_id,
			course_evaluations.course_id,
			course_evaluations.teacher_id,
			teachers.name AS teacher_name,
			course_evaluations.mode,
			course_evaluations.mirror_evaluation_id,
			course_evaluations.mirror_entity_type,
			course_evaluations.workload_score,
			course_evaluations.gain_score,
			course_evaluations.difficulty_score,
			course_evaluations.teaching_score,
			course_evaluations.grading_score,
			course_evaluations.attendance_score,
			ROUND((
				course_evaluations.workload_score +
				course_evaluations.gain_score +
				course_evaluations.difficulty_score +
				COALESCE(course_evaluations.teaching_score, 0) +
				COALESCE(course_evaluations.grading_score, 0) +
				COALESCE(course_evaluations.attendance_score, 0)
			) / CASE WHEN course_evaluations.mode = 'linked' THEN 6.0 ELSE 3.0 END, 2) AS avg_rating,
			course_evaluations.comment,
			course_evaluations.is_anonymous,
			course_evaluations.status,
			COALESCE(eval_likes.like_count, 0) AS like_count,
			FALSE AS is_liked,
			users.id AS author_id,
			users.nickname AS author_name,
			users.avatar_url AS author_avatar_url,
			users.role AS author_role,
			course_evaluations.created_at,
			course_evaluations.updated_at`).
		Where("course_evaluations.id = ?", id).
		Scan(&item).Error
	if err != nil {
		return nil, err
	}
	if item.ID == 0 {
		return nil, gorm.ErrRecordNotFound
	}
	replyMap, err := r.listCourseEvaluationRepliesByEvaluationIDs([]int64{id})
	if err != nil {
		return nil, err
	}
	item.Replies = replyMap[item.ID]
	item.ReplyCount = int64(len(item.Replies))
	return &item, nil
}

func (r *courseRepository) CreateCourseEvaluationReply(reply *model.CourseEvaluationReplies) error {
	return r.db.Create(reply).Error
}

func (r *courseRepository) GetCourseEvaluationReplyByID(id int64) (*model.CourseEvaluationReplies, error) {
	var reply model.CourseEvaluationReplies
	if err := r.db.First(&reply, id).Error; err != nil {
		return nil, err
	}
	return &reply, nil
}

func (r *courseRepository) GetCourseEvaluationReplyDetailByID(id int64) (*EvaluationReply, error) {
	var reply EvaluationReply
	err := r.db.Table("course_evaluation_replies").
		Joins("JOIN users ON users.id = course_evaluation_replies.user_id").
		Joins("LEFT JOIN users AS reply_users ON reply_users.id = course_evaluation_replies.reply_to_user_id").
		Joins(`LEFT JOIN (
			SELECT target_id, COUNT(*) AS like_count
			FROM likes
			WHERE target_type = 'course_evaluation_reply'
			GROUP BY target_id
		) AS reply_likes ON reply_likes.target_id = course_evaluation_replies.id`).
		Select(`
			course_evaluation_replies.id,
			course_evaluation_replies.evaluation_id,
			course_evaluation_replies.user_id,
			course_evaluation_replies.content,
			course_evaluation_replies.is_anonymous,
			course_evaluation_replies.reply_to_reply_id,
			course_evaluation_replies.reply_to_user_id,
			COALESCE(reply_likes.like_count, 0) AS likes,
			FALSE AS is_liked,
				COALESCE(reply_users.nickname, '') AS reply_to_user_name,
				COALESCE(reply_users.role::text, '') AS reply_to_user_role,
				users.nickname AS author_name,
				users.avatar_url AS author_avatar,
				users.role::text AS author_role,
			course_evaluation_replies.created_at,
			course_evaluation_replies.updated_at`).
		Where("course_evaluation_replies.id = ?", id).
		Scan(&reply).Error
	if err != nil {
		return nil, err
	}
	if reply.ID == 0 {
		return nil, gorm.ErrRecordNotFound
	}
	return &reply, nil
}

func (r *courseRepository) UpdateCourseEvaluationReply(reply *model.CourseEvaluationReplies) error {
	return r.db.Model(&model.CourseEvaluationReplies{}).
		Where("id = ?", reply.ID).
		Updates(map[string]interface{}{
			"content":      reply.Content,
			"is_anonymous": reply.IsAnonymous,
			"updated_at":   time.Now(),
		}).Error
}

func (r *courseRepository) DeleteCourseEvaluationReply(id int64) error {
	return r.db.Delete(&model.CourseEvaluationReplies{}, id).Error
}

func (r *courseRepository) ListMyCourseEvaluations(userID int64, page, size int) ([]MyCourseEvaluationItem, int64, error) {
	var items []MyCourseEvaluationItem
	var total int64

	base := applyVisibleCourseEvaluationFilter(r.db.Table("course_evaluations"), "course_evaluations").
		Where("course_evaluations.user_id = ?", userID)
	if err := base.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	err := base.
		Joins("JOIN courses ON courses.id = course_evaluations.course_id").
		Joins("LEFT JOIN teachers ON teachers.id = course_evaluations.teacher_id").
		Select(`
			course_evaluations.id,
			course_evaluations.course_id,
			courses.name AS course_name,
			course_evaluations.teacher_id,
			teachers.name AS teacher_name,
			course_evaluations.mode,
			course_evaluations.mirror_evaluation_id,
			course_evaluations.mirror_entity_type,
			course_evaluations.workload_score,
			course_evaluations.gain_score,
			course_evaluations.difficulty_score,
			course_evaluations.teaching_score,
			course_evaluations.grading_score,
			course_evaluations.attendance_score,
			ROUND((
				course_evaluations.workload_score +
				course_evaluations.gain_score +
				course_evaluations.difficulty_score +
				COALESCE(course_evaluations.teaching_score, 0) +
				COALESCE(course_evaluations.grading_score, 0) +
				COALESCE(course_evaluations.attendance_score, 0)
			) / CASE WHEN course_evaluations.mode = 'linked' THEN 6.0 ELSE 3.0 END, 2) AS avg_rating,
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
			"teacher_id":           evaluation.TeacherID,
			"mode":                 evaluation.Mode,
			"mirror_evaluation_id": evaluation.MirrorEvaluationID,
			"mirror_entity_type":   evaluation.MirrorEntityType,
			"workload_score":       evaluation.WorkloadScore,
			"gain_score":           evaluation.GainScore,
			"difficulty_score":     evaluation.DifficultyScore,
			"teaching_score":       evaluation.TeachingScore,
			"grading_score":        evaluation.GradingScore,
			"attendance_score":     evaluation.AttendanceScore,
			"comment":              evaluation.Comment,
			"is_anonymous":         evaluation.IsAnonymous,
			"status":               evaluation.Status,
			"updated_at":           time.Now(),
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

func (r *courseRepository) FindCourseEvaluationByContext(userID, courseID int64, teacherID *int64, mode model.EvaluationMode) (*model.CourseEvaluations, error) {
	var evaluation model.CourseEvaluations
	query := r.db.Where("user_id = ? AND course_id = ? AND mode = ?", userID, courseID, mode)
	if teacherID == nil {
		query = query.Where("teacher_id IS NULL")
	} else {
		query = query.Where("teacher_id = ?", *teacherID)
	}
	if err := query.First(&evaluation).Error; err != nil {
		return nil, err
	}
	return &evaluation, nil
}

func (r *courseRepository) CourseExists(id int64) (bool, error) {
	var count int64
	if err := r.db.Table("courses").Where("id = ? AND status = ?", id, model.CourseStatusActive).Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r *courseRepository) ListRandomCourseIDs(limit int) ([]int64, error) {
	if limit <= 0 {
		return nil, nil
	}

	var total int64
	base := r.db.Table("courses").Where("status = ?", model.CourseStatusActive)
	if err := base.Count(&total).Error; err != nil {
		return nil, err
	}
	if total == 0 {
		return nil, nil
	}

	randomizer := rand.New(rand.NewSource(time.Now().UnixNano()))
	windows := buildRandomCourseWindows(int(total), limit, randomizer.Intn(int(total)))
	ids := make([]int64, 0, min(limit, int(total)))
	for _, window := range windows {
		var chunk []int64
		if err := base.Order("id ASC").Offset(window.Offset).Limit(window.Limit).Pluck("id", &chunk).Error; err != nil {
			return nil, err
		}
		ids = append(ids, chunk...)
	}

	return ids, nil
}

type randomCourseWindow struct {
	Offset int
	Limit  int
}

func buildRandomCourseWindows(total, limit, start int) []randomCourseWindow {
	if total <= 0 || limit <= 0 {
		return nil
	}

	effectiveLimit := min(total, limit)
	normalizedStart := start % total
	if normalizedStart < 0 {
		normalizedStart += total
	}

	firstLimit := min(effectiveLimit, total-normalizedStart)
	windows := []randomCourseWindow{{
		Offset: normalizedStart,
		Limit:  firstLimit,
	}}
	if firstLimit == effectiveLimit {
		return windows
	}

	windows = append(windows, randomCourseWindow{
		Offset: 0,
		Limit:  effectiveLimit - firstLimit,
	})
	return windows
}

func (r *courseRepository) AdjustCourseAggregates(courseID int64, resourceDelta, downloadDelta, viewDelta, likeDelta, favoriteDelta, resourceFavoriteDelta, evalDelta int) error {
	updates := map[string]interface{}{
		"updated_at": time.Now(),
	}
	if resourceDelta != 0 {
		updates["resource_count"] = gorm.Expr("GREATEST(resource_count + ?, 0)", resourceDelta)
	}
	if downloadDelta != 0 {
		updates["download_total"] = gorm.Expr("GREATEST(download_total + ?, 0)", downloadDelta)
	}
	if viewDelta != 0 {
		updates["view_total"] = gorm.Expr("GREATEST(view_total + ?, 0)", viewDelta)
	}
	if likeDelta != 0 {
		updates["like_total"] = gorm.Expr("GREATEST(like_total + ?, 0)", likeDelta)
	}
	if favoriteDelta != 0 {
		updates["favorite_count"] = gorm.Expr("GREATEST(favorite_count + ?, 0)", favoriteDelta)
	}
	if resourceFavoriteDelta != 0 {
		updates["resource_favorite_count"] = gorm.Expr("GREATEST(resource_favorite_count + ?, 0)", resourceFavoriteDelta)
	}
	if evalDelta != 0 {
		updates["eval_count"] = gorm.Expr("GREATEST(eval_count + ?, 0)", evalDelta)
	}
	return r.db.Model(&model.Courses{}).Where("id = ?", courseID).Updates(updates).Error
}

func (r *courseRepository) RecalculateCourseStats(courseID int64) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		return tx.Exec(fmt.Sprintf(`
			UPDATE courses
			SET
				avg_workload_score = stats.avg_workload_score,
				avg_gain_score = stats.avg_gain_score,
				avg_difficulty_score = stats.avg_difficulty_score,
				eval_count = stats.eval_count,
				resource_count = resource_stats.resource_count,
				download_total = resource_stats.download_total,
				view_total = resource_stats.view_total,
				like_total = resource_stats.like_total,
				favorite_count = favorite_stats.favorite_count,
				resource_favorite_count = resource_favorite_stats.favorite_count,
				updated_at = CURRENT_TIMESTAMP
			FROM (
				SELECT
					COALESCE(ROUND(AVG(workload_score)::numeric, 2), 0) AS avg_workload_score,
					COALESCE(ROUND(AVG(gain_score)::numeric, 2), 0) AS avg_gain_score,
					COALESCE(ROUND(AVG(difficulty_score)::numeric, 2), 0) AS avg_difficulty_score,
					COUNT(*) AS eval_count
				FROM course_evaluations
				WHERE course_id = ? AND %s
			) AS stats,
			(
				SELECT
					COUNT(*) AS resource_count,
					COALESCE(SUM(download_count), 0) AS download_total,
					COALESCE(SUM(view_count), 0) AS view_total,
					COALESCE(SUM(like_count), 0) AS like_total
				FROM resources
				WHERE course_id = ? AND status = 'approved'
			) AS resource_stats,
			(
				SELECT COUNT(*) AS favorite_count
				FROM favorites
				WHERE target_type = 'course' AND target_id = ?
			) AS favorite_stats,
			(
				SELECT COUNT(*) AS favorite_count
				FROM favorites
				JOIN resources ON resources.id = favorites.target_id
				WHERE favorites.target_type = 'resource'
					AND resources.course_id = ?
					AND resources.status = 'approved'
			) AS resource_favorite_stats
			WHERE courses.id = ?`, visibleCourseEvaluationCondition("course_evaluations")), courseID, courseID, courseID, courseID, courseID).Error
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
	case "favorite_count":
		return courseFavoriteCountExpr("courses")
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

func (r *courseRepository) listCourseEvaluationRepliesByEvaluationIDs(ids []int64) (map[int64][]EvaluationReply, error) {
	result := make(map[int64][]EvaluationReply)
	if len(ids) == 0 {
		return result, nil
	}
	var replies []EvaluationReply
	err := r.db.Table("course_evaluation_replies").
		Joins("JOIN users ON users.id = course_evaluation_replies.user_id").
		Joins("LEFT JOIN users AS reply_users ON reply_users.id = course_evaluation_replies.reply_to_user_id").
		Select(`
			course_evaluation_replies.id,
			course_evaluation_replies.evaluation_id,
			course_evaluation_replies.user_id,
			course_evaluation_replies.content,
			course_evaluation_replies.is_anonymous,
			course_evaluation_replies.reply_to_reply_id,
			course_evaluation_replies.reply_to_user_id,
			COALESCE((
				SELECT COUNT(*)
				FROM likes
				WHERE likes.target_type = 'course_evaluation_reply' AND likes.target_id = course_evaluation_replies.id
			), 0) AS likes,
			FALSE AS is_liked,
			COALESCE(reply_users.nickname, '') AS reply_to_user_name,
			COALESCE(reply_users.role::text, '') AS reply_to_user_role,
			users.nickname AS author_name,
			users.avatar_url AS author_avatar,
			users.role::text AS author_role,
			course_evaluation_replies.created_at,
			course_evaluation_replies.updated_at`).
		Where("course_evaluation_replies.evaluation_id IN ?", ids).
		Order("course_evaluation_replies.created_at ASC, course_evaluation_replies.id ASC").
		Scan(&replies).Error
	if err != nil {
		return nil, err
	}
	for _, reply := range replies {
		result[reply.EvaluationID] = append(result[reply.EvaluationID], reply)
	}
	return result, nil
}

func (r *courseRepository) FindRandomCourses(ids []int64) (RandomCourses, error) {
	var tempRandomCourses []TempRandomCourseItem
	courseMap := make(map[int64]*RandomCourseItem)
	var randomCourses RandomCourses
	err := r.db.Table("courses c").
		Select(fmt.Sprintf(`
			c.id,
			c.name,
			c.course_type,
			c.avg_workload_score AS avg_homework,
			c.avg_gain_score AS avg_gain,
			c.avg_difficulty_score AS avg_exam_diff,
			(SELECT COUNT(*) FROM course_evaluations 
			WHERE course_evaluations.course_id = c.id
				AND %s) AS eval_count,
			c.resource_count,
			t.id AS teacher_id,
			t.name AS teacher_name,
			t.title AS teacher_title,
			t.avatar_url AS teacher_avatar_url,
			t.metadata AS teacher_metadata
		`, visibleCourseEvaluationCondition("course_evaluations"))).
		Joins("LEFT JOIN course_teachers ct ON c.id = ct.course_id AND ct.status = ?", model.CourseTeacherRelationStatusActive).
		Joins("LEFT JOIN teachers t ON t.id = ct.teacher_id AND t.status = 'active'").
		Where("c.id IN ? AND c.status = ?", ids, model.CourseStatusActive).
		Find(&tempRandomCourses).Error
	if err != nil {
		return randomCourses, err
	}

	for _, row := range tempRandomCourses {
		course, exists := courseMap[row.ID]
		if !exists {
			course = &RandomCourseItem{
				ID:            row.ID,
				Name:          row.Name,
				CourseType:    string(row.CourseType),
				AvgScore:      (row.AvgGain + row.AvgHomework + row.AvgExamDiff) / 3.0,
				AvgHomework:   row.AvgHomework,
				AvgGain:       row.AvgGain,
				AvgExamDiff:   row.AvgExamDiff,
				EvalCount:     row.EvalCount,
				ResourceCount: row.ResourceCount,
				Teachers:      []TeacherBrief{},
			}
			courseMap[row.ID] = course
		}
		if row.TeacherID != nil {
			course.Teachers = append(course.Teachers, TeacherBrief{
				ID:         *row.TeacherID,
				Name:       row.TeacherName,
				Title:      row.TeacherTitle,
				AvatarURL:  row.TeacherAvatarUrl,
				DetailPath: "/teachers/" + strconv.FormatInt(*row.TeacherID, 10),
			})
		}
	}

	for _, course := range courseMap {
		randomCourses.Items = append(randomCourses.Items, *course)
	}
	return randomCourses, nil
}

func extractCourseEvaluationIDs(items []CourseEvaluationItem) []int64 {
	ids := make([]int64, 0, len(items))
	for _, item := range items {
		ids = append(ids, item.ID)
	}
	return ids
}
