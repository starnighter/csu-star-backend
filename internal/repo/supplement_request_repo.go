package repo

import (
	"csu-star-backend/internal/model"
	"strings"
	"time"

	"gorm.io/gorm"
)

type SupplementRequestListQuery struct {
	Status      string
	RequestType string
	Keyword     string
	Page        int
	Size        int
}

type SupplementRequestItem struct {
	ID                 int64      `json:"id,string"`
	UserID             int64      `json:"user_id,string"`
	User               *UserBrief `json:"user,omitempty"`
	RequestType        string     `json:"request_type"`
	Status             string     `json:"status"`
	Contact            string     `json:"contact"`
	TeacherName        string     `json:"teacher_name,omitempty"`
	DepartmentID       *int16     `json:"department_id,omitempty"`
	DepartmentName     string     `json:"department_name,omitempty"`
	CourseName         string     `json:"course_name,omitempty"`
	CourseType         string     `json:"course_type,omitempty"`
	Remark             string     `json:"remark,omitempty"`
	ReviewedBy         *int64     `json:"reviewed_by,omitempty,string"`
	ReviewedAt         *time.Time `json:"reviewed_at,omitempty"`
	ReviewNote         string     `json:"review_note,omitempty"`
	ApprovedTargetType string     `json:"approved_target_type,omitempty"`
	ApprovedTargetID   *int64     `json:"approved_target_id,omitempty,string"`
	CreatedAt          time.Time  `json:"created_at"`
	UpdatedAt          time.Time  `json:"updated_at"`
}

type TeacherSupplementRequestItem struct {
	ID                int64      `json:"id,string"`
	UserID            int64      `json:"user_id,string"`
	User              *UserBrief `json:"user,omitempty" gorm:"-"`
	Status            string     `json:"status"`
	Contact           string     `json:"contact"`
	TeacherName       string     `json:"teacher_name"`
	DepartmentID      int16      `json:"department_id"`
	DepartmentName    string     `json:"department_name"`
	Remark            string     `json:"remark,omitempty"`
	ReviewedBy        *int64     `json:"reviewed_by,omitempty,string"`
	ReviewedAt        *time.Time `json:"reviewed_at,omitempty"`
	ReviewNote        string     `json:"review_note,omitempty"`
	ApprovedTeacherID *int64     `json:"approved_teacher_id,omitempty,string"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`

	ApplicantNickname  string `json:"-" gorm:"column:applicant_nickname"`
	ApplicantAvatarURL string `json:"-" gorm:"column:applicant_avatar_url"`
	ApplicantRole      string `json:"-" gorm:"column:applicant_role"`
}

type CourseSupplementRequestItem struct {
	ID               int64      `json:"id,string"`
	UserID           int64      `json:"user_id,string"`
	User             *UserBrief `json:"user,omitempty" gorm:"-"`
	Status           string     `json:"status"`
	Contact          string     `json:"contact"`
	CourseName       string     `json:"course_name"`
	CourseType       string     `json:"course_type"`
	Remark           string     `json:"remark,omitempty"`
	ReviewedBy       *int64     `json:"reviewed_by,omitempty,string"`
	ReviewedAt       *time.Time `json:"reviewed_at,omitempty"`
	ReviewNote       string     `json:"review_note,omitempty"`
	ApprovedCourseID *int64     `json:"approved_course_id,omitempty,string"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`

	ApplicantNickname  string `json:"-" gorm:"column:applicant_nickname"`
	ApplicantAvatarURL string `json:"-" gorm:"column:applicant_avatar_url"`
	ApplicantRole      string `json:"-" gorm:"column:applicant_role"`
}

func (r *miscRepository) CreateTeacherSupplementRequest(request *model.TeacherSupplementRequests) error {
	return r.db.Create(request).Error
}

func (r *miscRepository) GetTeacherSupplementRequestByID(id int64) (*TeacherSupplementRequestItem, error) {
	var item TeacherSupplementRequestItem
	err := r.baseTeacherSupplementRequestQuery().
		Where("teacher_supplement_requests.id = ?", id).
		Scan(&item).Error
	if err != nil {
		return nil, err
	}
	if item.ID == 0 {
		return nil, gorm.ErrRecordNotFound
	}
	hydrateTeacherSupplementRequestUser(&item)
	return &item, nil
}

func (r *miscRepository) ListTeacherSupplementRequests(query SupplementRequestListQuery) ([]TeacherSupplementRequestItem, int64, error) {
	var items []TeacherSupplementRequestItem
	var total int64

	base := r.baseTeacherSupplementRequestQuery()
	if status := strings.TrimSpace(query.Status); status != "" {
		base = base.Where("teacher_supplement_requests.status = ?", status)
	}
	if keyword := strings.TrimSpace(query.Keyword); keyword != "" {
		like := "%" + keyword + "%"
		base = base.Where(`
			teacher_supplement_requests.teacher_name ILIKE ?
			OR departments.name ILIKE ?
			OR users.nickname ILIKE ?
			OR teacher_supplement_requests.contact ILIKE ?
		`, like, like, like, like)
	}

	if err := base.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	scoped := base.
		Order("teacher_supplement_requests.created_at DESC").
		Order("teacher_supplement_requests.id DESC")
	if query.Size > 0 {
		scoped = scoped.Offset((query.Page - 1) * query.Size).Limit(query.Size)
	}
	err := scoped.Scan(&items).Error
	if err != nil {
		return nil, 0, err
	}
	for i := range items {
		hydrateTeacherSupplementRequestUser(&items[i])
	}
	return items, total, nil
}

func (r *miscRepository) baseTeacherSupplementRequestQuery() *gorm.DB {
	return r.db.Table("teacher_supplement_requests").
		Joins("LEFT JOIN users ON users.id = teacher_supplement_requests.user_id").
		Joins("LEFT JOIN departments ON departments.id = teacher_supplement_requests.department_id").
		Select(`
			teacher_supplement_requests.id,
			teacher_supplement_requests.user_id,
			teacher_supplement_requests.status,
			teacher_supplement_requests.contact,
			teacher_supplement_requests.teacher_name,
			teacher_supplement_requests.department_id,
			COALESCE(departments.name, '') AS department_name,
			teacher_supplement_requests.remark,
			teacher_supplement_requests.reviewed_by,
			teacher_supplement_requests.reviewed_at,
			teacher_supplement_requests.review_note,
			teacher_supplement_requests.approved_teacher_id,
			teacher_supplement_requests.created_at,
			teacher_supplement_requests.updated_at,
			COALESCE(users.nickname, '') AS applicant_nickname,
			COALESCE(users.avatar_url, '') AS applicant_avatar_url,
			COALESCE(users.role::text, '') AS applicant_role
		`)
}

func hydrateTeacherSupplementRequestUser(item *TeacherSupplementRequestItem) {
	if item == nil || item.UserID == 0 {
		return
	}
	item.User = buildUserBrief(item.UserID, item.ApplicantNickname, item.ApplicantAvatarURL, item.ApplicantRole)
}

func (r *miscRepository) UpdateTeacherSupplementRequest(id int64, updates map[string]interface{}) error {
	return r.db.Model(&model.TeacherSupplementRequests{}).Where("id = ?", id).Updates(updates).Error
}

func (r *miscRepository) CreateCourseSupplementRequest(request *model.CourseSupplementRequests) error {
	return r.db.Create(request).Error
}

func (r *miscRepository) GetCourseSupplementRequestByID(id int64) (*CourseSupplementRequestItem, error) {
	var item CourseSupplementRequestItem
	err := r.baseCourseSupplementRequestQuery().
		Where("course_supplement_requests.id = ?", id).
		Scan(&item).Error
	if err != nil {
		return nil, err
	}
	if item.ID == 0 {
		return nil, gorm.ErrRecordNotFound
	}
	hydrateCourseSupplementRequestUser(&item)
	return &item, nil
}

func (r *miscRepository) ListCourseSupplementRequests(query SupplementRequestListQuery) ([]CourseSupplementRequestItem, int64, error) {
	var items []CourseSupplementRequestItem
	var total int64

	base := r.baseCourseSupplementRequestQuery()
	if status := strings.TrimSpace(query.Status); status != "" {
		base = base.Where("course_supplement_requests.status = ?", status)
	}
	if keyword := strings.TrimSpace(query.Keyword); keyword != "" {
		like := "%" + keyword + "%"
		base = base.Where(`
			course_supplement_requests.course_name ILIKE ?
			OR users.nickname ILIKE ?
			OR course_supplement_requests.contact ILIKE ?
		`, like, like, like)
	}

	if err := base.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	scoped := base.
		Order("course_supplement_requests.created_at DESC").
		Order("course_supplement_requests.id DESC")
	if query.Size > 0 {
		scoped = scoped.Offset((query.Page - 1) * query.Size).Limit(query.Size)
	}
	err := scoped.Scan(&items).Error
	if err != nil {
		return nil, 0, err
	}
	for i := range items {
		hydrateCourseSupplementRequestUser(&items[i])
	}
	return items, total, nil
}

func (r *miscRepository) baseCourseSupplementRequestQuery() *gorm.DB {
	return r.db.Table("course_supplement_requests").
		Joins("LEFT JOIN users ON users.id = course_supplement_requests.user_id").
		Select(`
			course_supplement_requests.id,
			course_supplement_requests.user_id,
			course_supplement_requests.status,
			course_supplement_requests.contact,
			course_supplement_requests.course_name,
			course_supplement_requests.course_type,
			course_supplement_requests.remark,
			course_supplement_requests.reviewed_by,
			course_supplement_requests.reviewed_at,
			course_supplement_requests.review_note,
			course_supplement_requests.approved_course_id,
			course_supplement_requests.created_at,
			course_supplement_requests.updated_at,
			COALESCE(users.nickname, '') AS applicant_nickname,
			COALESCE(users.avatar_url, '') AS applicant_avatar_url,
			COALESCE(users.role::text, '') AS applicant_role
		`)
}

func hydrateCourseSupplementRequestUser(item *CourseSupplementRequestItem) {
	if item == nil || item.UserID == 0 {
		return
	}
	item.User = buildUserBrief(item.UserID, item.ApplicantNickname, item.ApplicantAvatarURL, item.ApplicantRole)
}

func (r *miscRepository) UpdateCourseSupplementRequest(id int64, updates map[string]interface{}) error {
	return r.db.Model(&model.CourseSupplementRequests{}).Where("id = ?", id).Updates(updates).Error
}

func buildUserBrief(id int64, nickname, avatarURL, role string) *UserBrief {
	return &UserBrief{
		ID:        id,
		Nickname:  nickname,
		AvatarURL: avatarURL,
		Role:      role,
	}
}
