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
	User               *UserBrief `json:"user,omitempty" gorm:"-"`
	RequestType        string     `json:"request_type"`
	Status             string     `json:"status"`
	Contact            string     `json:"contact"`
	TeacherName        string     `json:"teacher_name,omitempty"`
	DepartmentID       *int16     `json:"department_id,omitempty"`
	DepartmentName     string     `json:"department_name,omitempty"`
	RelatedCourseName  string     `json:"related_course_name,omitempty"`
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

	ApplicantNickname  string `json:"-" gorm:"column:applicant_nickname"`
	ApplicantAvatarURL string `json:"-" gorm:"column:applicant_avatar_url"`
	ApplicantRole      string `json:"-" gorm:"column:applicant_role"`
}

func (r *miscRepository) CreateSupplementRequest(request *model.SupplementRequests) error {
	return r.db.Create(request).Error
}

func (r *miscRepository) GetSupplementRequestByID(id int64) (*SupplementRequestItem, error) {
	var item SupplementRequestItem
	err := r.baseSupplementRequestQuery().
		Where("supplement_requests.id = ?", id).
		Scan(&item).Error
	if err != nil {
		return nil, err
	}
	if item.ID == 0 {
		return nil, gorm.ErrRecordNotFound
	}
	hydrateSupplementRequestUser(&item)
	return &item, nil
}

func (r *miscRepository) ListSupplementRequests(query SupplementRequestListQuery) ([]SupplementRequestItem, int64, error) {
	var items []SupplementRequestItem
	var total int64

	base := r.baseSupplementRequestQuery()

	if status := strings.TrimSpace(query.Status); status != "" {
		base = base.Where("supplement_requests.status = ?", status)
	}
	if requestType := strings.TrimSpace(query.RequestType); requestType != "" {
		base = base.Where("supplement_requests.request_type = ?", requestType)
	}
	if keyword := strings.TrimSpace(query.Keyword); keyword != "" {
		like := "%" + keyword + "%"
		base = base.Where(`
			supplement_requests.teacher_name ILIKE ?
			OR supplement_requests.course_name ILIKE ?
			OR supplement_requests.related_course_name ILIKE ?
			OR users.nickname ILIKE ?
			OR supplement_requests.contact ILIKE ?
		`, like, like, like, like, like)
	}

	if err := base.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	err := base.
		Order("supplement_requests.created_at DESC").
		Order("supplement_requests.id DESC").
		Offset((query.Page - 1) * query.Size).
		Limit(query.Size).
		Scan(&items).Error
	if err != nil {
		return nil, 0, err
	}

	for i := range items {
		hydrateSupplementRequestUser(&items[i])
	}

	return items, total, nil
}

func (r *miscRepository) baseSupplementRequestQuery() *gorm.DB {
	return r.db.Table("supplement_requests").
		Joins("LEFT JOIN users ON users.id = supplement_requests.user_id").
		Joins("LEFT JOIN departments ON departments.id = supplement_requests.department_id").
		Select(`
			supplement_requests.id,
			supplement_requests.user_id,
			supplement_requests.request_type,
			supplement_requests.status,
			supplement_requests.contact,
			supplement_requests.teacher_name,
			supplement_requests.department_id,
			COALESCE(departments.name, '') AS department_name,
			supplement_requests.related_course_name,
			supplement_requests.course_name,
			supplement_requests.course_type,
			supplement_requests.remark,
			supplement_requests.reviewed_by,
			supplement_requests.reviewed_at,
			supplement_requests.review_note,
			supplement_requests.approved_target_type,
			supplement_requests.approved_target_id,
			supplement_requests.created_at,
			supplement_requests.updated_at,
			COALESCE(users.nickname, '') AS applicant_nickname,
			COALESCE(users.avatar_url, '') AS applicant_avatar_url,
			COALESCE(users.role::text, '') AS applicant_role`)
}

func hydrateSupplementRequestUser(item *SupplementRequestItem) {
	item.User = &UserBrief{
		ID:        item.UserID,
		Nickname:  item.ApplicantNickname,
		AvatarURL: item.ApplicantAvatarURL,
		Role:      item.ApplicantRole,
	}
}
