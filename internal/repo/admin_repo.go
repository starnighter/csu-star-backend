package repo

import (
	"csu-star-backend/internal/model"
	"csu-star-backend/internal/realtime"
	"encoding/json"
	"strings"
	"time"

	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type AdminStatistics struct {
	TotalUsers              int64 `json:"total_users"`
	TotalResources          int64 `json:"total_resources"`
	TotalEvaluations        int64 `json:"total_evaluations"`
	TotalCourses            int64 `json:"total_courses"`
	TotalTeachers           int64 `json:"total_teachers"`
	TodayNewUsers           int64 `json:"today_new_users"`
	TodayNewResources       int64 `json:"today_new_resources"`
	TodayNewEvaluations     int64 `json:"today_new_evaluations"`
	PendingReportsCount     int64 `json:"pending_reports_count"`
	PendingCorrectionsCount int64 `json:"pending_corrections_count"`
	PendingFeedbacksCount   int64 `json:"pending_feedbacks_count"`
	PendingSupplementsCount int64 `json:"pending_supplement_requests_count"`
	DeletedResourcesCount   int64 `json:"deleted_resources_count"`
	DeletedCoursesCount     int64 `json:"deleted_courses_count"`
	DeletedTeachersCount    int64 `json:"deleted_teachers_count"`
}

type AdminReportItem struct {
	ID              int64                     `json:"id,string"`
	UserID          int64                     `json:"user_id,string"`
	User            *UserBrief                `json:"user,omitempty" gorm:"-"`
	TargetPreview   *AdminReportTargetPreview `json:"target_preview,omitempty" gorm:"-"`
	Processor       *UserBrief                `json:"processor,omitempty" gorm:"-"`
	TargetType      string                    `json:"target_type"`
	TargetID        int64                     `json:"target_id,string"`
	Reason          string                    `json:"reason"`
	Description     string                    `json:"description"`
	Status          string                    `json:"status"`
	ProcessorID     *int64                    `json:"processor_id,omitempty,string"`
	ProcessAt       *time.Time                `json:"processed_at,omitempty"`
	ProcessNote     string                    `json:"process_note"`
	CreatedAt       time.Time                 `json:"created_at"`
	UpdatedAt       time.Time                 `json:"updated_at"`
	UserNickname    string                    `json:"-" gorm:"column:user_nickname"`
	UserAvatarURL   string                    `json:"-" gorm:"column:user_avatar_url"`
	UserRole        string                    `json:"-" gorm:"column:user_role"`
	ProcessorName   string                    `json:"-" gorm:"column:processor_name"`
	ProcessorAvatar string                    `json:"-" gorm:"column:processor_avatar"`
	ProcessorRole   string                    `json:"-" gorm:"column:processor_role"`
}

type AdminReportTargetPreview struct {
	Title      string     `json:"title"`
	Subtitle   string     `json:"subtitle,omitempty"`
	Content    string     `json:"content,omitempty"`
	Status     string     `json:"status,omitempty"`
	AuthorName string     `json:"author_name,omitempty"`
	CreatedAt  *time.Time `json:"created_at,omitempty"`
	Missing    bool       `json:"missing,omitempty"`
}

type AdminCorrectionItem struct {
	ID         int64      `json:"id,string"`
	UserID     int64      `json:"user_id,string"`
	User       *UserBrief `json:"user,omitempty" gorm:"-"`
	Processor  *UserBrief `json:"processor,omitempty" gorm:"-"`
	TargetType string     `json:"target_type"`
	TargetID   int64      `json:"target_id,string"`
	Field      string     `json:"field"`
	// CurrentValue 是纠错目标当前的字段值，供审核员对比「改成什么」之前先看到「原本是什么」。
	// 取值逻辑必须与 service.applyCourseCorrection / applyTeacherCorrection 的 field switch 保持一致。
	CurrentValue string `json:"current_value"`
	// CurrentValueDisplay / SuggestedValueDisplay 仅在字段本身是外键（当前只有 department_id）时非空，
	// 用于把裸 ID 翻译成可读名称，否则审核员看到的是两个无意义的数字。
	CurrentValueDisplay   string     `json:"current_value_display,omitempty"`
	SuggestedValue        string     `json:"suggested_value"`
	SuggestedValueDisplay string     `json:"suggested_value_display,omitempty"`
	TargetMissing         bool       `json:"target_missing,omitempty"`
	Status                string     `json:"status"`
	ProcessorID           *int64     `json:"processor_id,omitempty,string"`
	ProcessAt             *time.Time `json:"processed_at,omitempty"`
	ProcessNote           string     `json:"process_note"`
	CreatedAt             time.Time  `json:"created_at"`
	UpdatedAt             time.Time  `json:"updated_at"`
	UserNickname          string     `json:"-" gorm:"column:user_nickname"`
	UserAvatarURL         string     `json:"-" gorm:"column:user_avatar_url"`
	UserRole              string     `json:"-" gorm:"column:user_role"`
	ProcessorName         string     `json:"-" gorm:"column:processor_name"`
	ProcessorAvatar       string     `json:"-" gorm:"column:processor_avatar"`
	ProcessorRole         string     `json:"-" gorm:"column:processor_role"`
}

type AdminFeedbackItem struct {
	ID             int64          `json:"id,string"`
	UserID         int64          `json:"user_id,string"`
	User           *UserBrief     `json:"user,omitempty" gorm:"-"`
	Replier        *UserBrief     `json:"replier,omitempty" gorm:"-"`
	Type           string         `json:"type"`
	Title          string         `json:"title"`
	Content        string         `json:"content"`
	Attachments    []string       `json:"attachments"`
	Status         string         `json:"status"`
	RepliedBy      *int64         `json:"replied_by,omitempty,string"`
	RepliedAt      *time.Time     `json:"replied_at,omitempty"`
	Reply          string         `json:"reply"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
	RawAttachments datatypes.JSON `json:"-" gorm:"column:attachments"`
	UserNickname   string         `json:"-" gorm:"column:user_nickname"`
	UserAvatarURL  string         `json:"-" gorm:"column:user_avatar_url"`
	UserRole       string         `json:"-" gorm:"column:user_role"`
	ReplierName    string         `json:"-" gorm:"column:replier_name"`
	ReplierAvatar  string         `json:"-" gorm:"column:replier_avatar"`
	ReplierRole    string         `json:"-" gorm:"column:replier_role"`
}

type AdminUserItem struct {
	ID                int64      `json:"id,string"`
	Email             *string    `json:"email,omitempty"`
	StudentID         *string    `json:"student_id,omitempty"`
	Nickname          string     `json:"nickname"`
	AvatarURL         string     `json:"avatar_url"`
	Role              string     `json:"role"`
	Status            string     `json:"status"`
	BanUntil          *time.Time `json:"ban_until,omitempty"`
	BanReason         string     `json:"ban_reason"`
	BanSource         string     `json:"ban_source"`
	ViolationCount    int        `json:"violation_count"`
	LastViolationAt   *time.Time `json:"last_violation_at,omitempty"`
	EmailVerified     bool       `json:"email_verified"`
	Points            int        `json:"points"`
	FreeDownloadCount int        `json:"free_download_count"`
	LastLoginAt       *time.Time `json:"last_login_at,omitempty"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
}

type AdminUserViolationItem struct {
	ID                 int64          `json:"id,string"`
	UserID             int64          `json:"user_id,string"`
	Scope              string         `json:"scope"`
	TriggerKey         string         `json:"trigger_key"`
	Reason             string         `json:"reason"`
	Evidence           datatypes.JSON `json:"evidence"`
	PenaltyLevel       int            `json:"penalty_level"`
	BanDurationSeconds int64          `json:"ban_duration_seconds"`
	CreatedAt          time.Time      `json:"created_at"`
}

type AdminAnnouncementItem struct {
	ID          int64      `json:"id,string"`
	Title       string     `json:"title"`
	Content     string     `json:"content"`
	Type        string     `json:"type"`
	IsPinned    bool       `json:"is_pinned"`
	IsPublished bool       `json:"is_published"`
	PublishedAt *time.Time `json:"published_at,omitempty"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	DeletedAt   *time.Time `json:"deleted_at,omitempty"`
}

type AdminCourseItem struct {
	ID            int64     `json:"id,string"`
	Name          string    `json:"name"`
	CourseType    string    `json:"course_type"`
	Description   string    `json:"description"`
	Credits       float64   `json:"credits"`
	ResourceCount int       `json:"resource_count"`
	EvalCount     int       `json:"eval_count"`
	FavoriteCount int       `json:"favorite_count"`
	Status        string    `json:"status"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
	// 课程是「改 status」的软删除，表里没有 deleted_at 列，判断删除请用 status == "deleted"。
}

type AdminTeacherItem struct {
	ID             int64     `json:"id,string"`
	Name           string    `json:"name"`
	Title          string    `json:"title"`
	DepartmentID   int16     `json:"department_id"`
	DepartmentName string    `json:"department_name"`
	AvatarURL      string    `json:"avatar_url"`
	Bio            string    `json:"bio"`
	TutorType      string    `json:"tutor_type"`
	HomepageURL    string    `json:"homepage_url"`
	FavoriteCount  int       `json:"favorite_count"`
	EvalCount      int64     `json:"eval_count"`
	Status         string    `json:"status"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type AdminCourseRelationItem struct {
	CourseID    int64      `json:"course_id,string"`
	TeacherID   int64      `json:"teacher_id,string"`
	TeacherName string     `json:"teacher_name"`
	Title       string     `json:"title"`
	AvatarURL   string     `json:"avatar_url"`
	Status      string     `json:"relation_status"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	CanceledAt  *time.Time `json:"canceled_at,omitempty"`
}

type AdminTeacherRelationItem struct {
	TeacherID  int64      `json:"teacher_id,string"`
	CourseID   int64      `json:"course_id,string"`
	CourseName string     `json:"course_name"`
	CourseType string     `json:"course_type"`
	Status     string     `json:"relation_status"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
	CanceledAt *time.Time `json:"canceled_at,omitempty"`
}

type AdminResourceItem struct {
	ID           int64     `json:"id,string"`
	Title        string    `json:"title"`
	CourseID     int64     `json:"course_id,string"`
	CourseName   string    `json:"course_name"`
	UploaderID   int64     `json:"uploader_id,string"`
	UploaderName string    `json:"uploader_name"`
	ResourceType string    `json:"resource_type"`
	Status       string    `json:"status"`
	Downloads    int       `json:"downloads"`
	Views        int       `json:"views"`
	Likes        int       `json:"likes"`
	CommentCount int       `json:"comment_count"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type AdminAuditLogItem struct {
	ID         int64      `json:"id,string"`
	OperatorID int64      `json:"operator_id,string"`
	Operator   *UserBrief `json:"operator,omitempty" gorm:"-"`
	Action     string     `json:"action"`
	TargetType string     `json:"target_type"`
	TargetID   int64      `json:"target_id,string"`
	// TargetName 由 hydrateAuditTargetNames 按 target_type 批量回填；
	// 对 report/correction/feedback/notification/relation 这类本身没有名称的对象为空。
	TargetName     string         `json:"target_name,omitempty" gorm:"-"`
	OldValues      datatypes.JSON `json:"old_values"`
	NewValues      datatypes.JSON `json:"new_values"`
	Reason         string         `json:"reason"`
	CreatedAt      time.Time      `json:"created_at"`
	OperatorName   string         `json:"-" gorm:"column:operator_name"`
	OperatorAvatar string         `json:"-" gorm:"column:operator_avatar"`
	OperatorRole   string         `json:"-" gorm:"column:operator_role"`
}

type AdminRepository interface {
	GetStatistics() (*AdminStatistics, error)
	ListReports(status string, page, size int) ([]AdminReportItem, int64, error)
	GetReportByID(id int64) (*model.Reports, error)
	UpdateReport(report *model.Reports) error
	ListCorrections(status string, page, size int) ([]AdminCorrectionItem, int64, error)
	GetCorrectionByID(id int64) (*model.Corrections, error)
	UpdateCorrection(correction *model.Corrections) error
	ListFeedbacks(status string, page, size int) ([]AdminFeedbackItem, int64, error)
	GetFeedbackByID(id int64) (*model.Feedbacks, error)
	UpdateFeedback(feedback *model.Feedbacks) error
	ListUsers(status, role, keyword string, page, size int) ([]AdminUserItem, int64, error)
	ListUserViolations(userID int64, page, size int) ([]AdminUserViolationItem, int64, error)
	GetUserByID(id int64) (*model.Users, error)
	UpdateUser(user *model.Users) error
	AdjustUserPoints(userID int64, delta int, reason string) (int, error)
	ListAnnouncements(page, size int) ([]AdminAnnouncementItem, int64, error)
	CreateAnnouncement(item *model.Announcements) error
	GetAdminAnnouncementItem(id int64) (*AdminAnnouncementItem, error)
	GetAdminCourseItem(id int64) (*AdminCourseItem, error)
	GetAdminTeacherItem(id int64) (*AdminTeacherItem, error)
	GetAnnouncementByID(id int64) (*model.Announcements, error)
	UpdateAnnouncement(item *model.Announcements) error
	DeleteAnnouncement(id int64) error
	ListCourses(status, courseType, keyword string, page, size int) ([]AdminCourseItem, int64, error)
	CreateCourse(course *model.Courses) error
	GetCourseByID(id int64) (*model.Courses, error)
	UpdateCourseFields(id int64, updates map[string]interface{}) error
	ListCourseRelations(courseID int64) ([]AdminCourseRelationItem, error)
	ListTeachers(status, keyword string, departmentID *int16, page, size int) ([]AdminTeacherItem, int64, error)
	CreateTeacher(teacher *model.Teachers) error
	GetTeacherByID(id int64) (*model.Teachers, error)
	UpdateTeacherFields(id int64, updates map[string]interface{}) error
	ListTeacherRelations(teacherID int64) ([]AdminTeacherRelationItem, error)
	ListResources(status, keyword, resourceType string, courseID int64, page, size int) ([]AdminResourceItem, int64, error)
	GetResourceByID(id int64) (*model.Resources, error)
	UpdateResourceStatus(id int64, status model.ResourceStatus) error
	ListAuditLogs(action string, operatorID int64, targetType string, page, size int) ([]AdminAuditLogItem, int64, error)
	CreateAuditLog(log *model.AuditLogs) error
	CreateNotification(notification *model.Notifications) error
}

type adminRepository struct {
	db *gorm.DB
}

func NewAdminRepository(db *gorm.DB) AdminRepository {
	return &adminRepository{db: db}
}

func (r *adminRepository) WithTx(tx *gorm.DB) AdminRepository {
	return &adminRepository{db: tx}
}

func (r *adminRepository) GetStatistics() (*AdminStatistics, error) {
	var item AdminStatistics
	err := r.db.Raw(`
		SELECT
			(SELECT COUNT(*) FROM users) AS total_users,
			(SELECT COUNT(*) FROM resources WHERE status <> 'deleted') AS total_resources,
			((SELECT COUNT(*) FROM teacher_evaluations) + (SELECT COUNT(*) FROM course_evaluations)) AS total_evaluations,
			(SELECT COUNT(*) FROM courses WHERE status = 'active') AS total_courses,
			(SELECT COUNT(*) FROM teachers WHERE status = 'active') AS total_teachers,
			(SELECT COUNT(*) FROM users WHERE created_at >= CURRENT_DATE) AS today_new_users,
			(SELECT COUNT(*) FROM resources WHERE created_at >= CURRENT_DATE) AS today_new_resources,
			((SELECT COUNT(*) FROM teacher_evaluations WHERE created_at >= CURRENT_DATE) + (SELECT COUNT(*) FROM course_evaluations WHERE created_at >= CURRENT_DATE)) AS today_new_evaluations,
			(SELECT COUNT(*) FROM reports WHERE status = 'pending') AS pending_reports_count,
			(SELECT COUNT(*) FROM corrections WHERE status = 'pending') AS pending_corrections_count,
			(SELECT COUNT(*) FROM feedbacks WHERE status IN ('pending', 'processing')) AS pending_feedbacks_count,
			(
				(SELECT COUNT(*) FROM teacher_supplement_requests WHERE status = 'pending')
				+
				(SELECT COUNT(*) FROM course_supplement_requests WHERE status = 'pending')
			) AS pending_supplement_requests_count,
			(SELECT COUNT(*) FROM resources WHERE status = 'deleted') AS deleted_resources_count,
			(SELECT COUNT(*) FROM courses WHERE status = 'deleted') AS deleted_courses_count,
			(SELECT COUNT(*) FROM teachers WHERE status = 'deleted') AS deleted_teachers_count
	`).Scan(&item).Error
	if err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *adminRepository) ListReports(status string, page, size int) ([]AdminReportItem, int64, error) {
	var items []AdminReportItem
	var total int64
	base := r.db.Table("reports")
	if status != "" {
		base = base.Where("reports.status = ?", status)
	}
	if err := base.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	err := base.
		Joins("JOIN users reporter ON reporter.id = reports.user_id").
		Joins("LEFT JOIN users processor ON processor.id = reports.processor_id").
		Select(`
			reports.id,
			reports.user_id,
			reports.target_type,
			reports.target_id,
			reports.reason,
			reports.description,
			reports.status,
			NULLIF(reports.processor_id, 0) AS processor_id,
			NULLIF(reports.process_at, '0001-01-01 00:00:00+00'::timestamptz) AS process_at,
			reports.process_note,
			reports.created_at,
			reports.updated_at,
			reporter.nickname AS user_nickname,
			reporter.avatar_url AS user_avatar_url,
			reporter.role::text AS user_role,
			COALESCE(processor.nickname, '') AS processor_name,
			COALESCE(processor.avatar_url, '') AS processor_avatar,
			COALESCE(processor.role::text, '') AS processor_role`).
		Order("reports.created_at DESC").
		Offset((page - 1) * size).
		Limit(size).
		Scan(&items).Error
	if err != nil {
		return nil, 0, err
	}
	for i := range items {
		hydrateAdminUserBrief(&items[i].User, items[i].UserID, items[i].UserNickname, items[i].UserAvatarURL, items[i].UserRole)
		hydrateOptionalUserBrief(&items[i].Processor, items[i].ProcessorID, items[i].ProcessorName, items[i].ProcessorAvatar, items[i].ProcessorRole)
		items[i].TargetPreview, err = r.getReportTargetPreview(items[i].TargetType, items[i].TargetID)
		if err != nil {
			return nil, 0, err
		}
	}
	return items, total, nil
}

func (r *adminRepository) GetReportByID(id int64) (*model.Reports, error) {
	var item model.Reports
	if err := r.db.First(&item, id).Error; err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *adminRepository) UpdateReport(report *model.Reports) error {
	return r.db.Save(report).Error
}

// correctionCurrentValueSQL 取出纠错目标当前的字段值，让审核员能看到「原本是什么 → 要改成什么」。
// 字段清单必须覆盖 model.CorrectionFieldsByTargetType，
// 由 TestCorrectionCurrentValueSQLCoversAllFields 守卫，漏一个就会在这里静默返回空字符串。
const correctionCurrentValueSQL = `,
			CASE
				WHEN corrections.target_type = 'course' THEN CASE corrections.field
					WHEN 'name'        THEN COALESCE(cc.name, '')
					WHEN 'description' THEN COALESCE(cc.description, '')
					WHEN 'course_type' THEN COALESCE(cc.course_type::text, '')
					ELSE '' END
				WHEN corrections.target_type = 'teacher' THEN CASE corrections.field
					WHEN 'name'          THEN COALESCE(tt.name, '')
					WHEN 'title'         THEN COALESCE(tt.title, '')
					WHEN 'avatar_url'    THEN COALESCE(tt.avatar_url, '')
					WHEN 'department_id' THEN COALESCE(tt.department_id::text, '')
					WHEN 'bio'           THEN COALESCE(tt.metadata->>'bio', '')
					WHEN 'tutor_type'    THEN COALESCE(tt.metadata->>'tutor_type', '')
					WHEN 'homepage_url'  THEN COALESCE(tt.metadata->>'homepage_url', '')
					ELSE '' END
				ELSE '' END AS current_value,
			COALESCE(cur_dept.name, '') AS current_value_display,
			COALESCE(sug_dept.name, '') AS suggested_value_display,
			(cc.id IS NULL AND tt.id IS NULL) AS target_missing`

func (r *adminRepository) ListCorrections(status string, page, size int) ([]AdminCorrectionItem, int64, error) {
	var items []AdminCorrectionItem
	var total int64
	base := r.db.Table("corrections")
	if status != "" {
		base = base.Where("corrections.status = ?", status)
	}
	if err := base.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	err := base.
		Joins("JOIN users reporter ON reporter.id = corrections.user_id").
		Joins("LEFT JOIN users processor ON processor.id = corrections.processor_id").
		// 按 target_type 守卫的两条 JOIN：一条纠错只会命中其中一张表，另一张恒为 NULL。
		Joins("LEFT JOIN courses cc ON corrections.target_type = 'course' AND cc.id = corrections.target_id").
		Joins("LEFT JOIN teachers tt ON corrections.target_type = 'teacher' AND tt.id = corrections.target_id").
		Joins("LEFT JOIN departments cur_dept ON cur_dept.id = tt.department_id").
		// 用 id::text = suggested_value 而不是反向 cast：suggested_value 是自由文本，
		// 反向 cast 遇到非数字会让整条查询报错。
		Joins("LEFT JOIN departments sug_dept ON corrections.field = 'department_id' AND sug_dept.id::text = corrections.suggested_value").
		Select(`
			corrections.id,
			corrections.user_id,
			corrections.target_type,
			corrections.target_id,
			corrections.field,
			corrections.suggested_value,
			corrections.status,
			NULLIF(corrections.processor_id, 0) AS processor_id,
			NULLIF(corrections.process_at, '0001-01-01 00:00:00+00'::timestamptz) AS process_at,
			corrections.process_note,
			corrections.created_at,
			corrections.updated_at,
			reporter.nickname AS user_nickname,
			reporter.avatar_url AS user_avatar_url,
			reporter.role::text AS user_role,
			COALESCE(processor.nickname, '') AS processor_name,
			COALESCE(processor.avatar_url, '') AS processor_avatar,
			COALESCE(processor.role::text, '') AS processor_role` + correctionCurrentValueSQL).
		Order("corrections.created_at DESC").
		Offset((page - 1) * size).
		Limit(size).
		Scan(&items).Error
	if err != nil {
		return nil, 0, err
	}
	for i := range items {
		hydrateAdminUserBrief(&items[i].User, items[i].UserID, items[i].UserNickname, items[i].UserAvatarURL, items[i].UserRole)
		hydrateOptionalUserBrief(&items[i].Processor, items[i].ProcessorID, items[i].ProcessorName, items[i].ProcessorAvatar, items[i].ProcessorRole)
		// *_display 只对指向其它表的字段有意义，其余字段清空避免误导前端。
		if items[i].Field != "department_id" {
			items[i].CurrentValueDisplay = ""
			items[i].SuggestedValueDisplay = ""
		}
	}
	return items, total, nil
}

func (r *adminRepository) GetCorrectionByID(id int64) (*model.Corrections, error) {
	var item model.Corrections
	if err := r.db.First(&item, id).Error; err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *adminRepository) UpdateCorrection(correction *model.Corrections) error {
	return r.db.Save(correction).Error
}

func (r *adminRepository) ListFeedbacks(status string, page, size int) ([]AdminFeedbackItem, int64, error) {
	var items []AdminFeedbackItem
	var total int64
	base := r.db.Table("feedbacks")
	if status != "" {
		base = base.Where("feedbacks.status = ?", status)
	}
	if err := base.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	err := base.
		Joins("JOIN users reporter ON reporter.id = feedbacks.user_id").
		Joins("LEFT JOIN users replier ON replier.id = feedbacks.replied_by").
		Select(`
			feedbacks.id,
			feedbacks.user_id,
			feedbacks.type,
			feedbacks.title,
			feedbacks.content,
			feedbacks.attachments,
			feedbacks.status,
			NULLIF(feedbacks.replied_by, 0) AS replied_by,
			NULLIF(feedbacks.replied_at, '0001-01-01 00:00:00+00'::timestamptz) AS replied_at,
			feedbacks.reply,
			feedbacks.created_at,
			feedbacks.updated_at,
			reporter.nickname AS user_nickname,
			reporter.avatar_url AS user_avatar_url,
			reporter.role::text AS user_role,
			COALESCE(replier.nickname, '') AS replier_name,
			COALESCE(replier.avatar_url, '') AS replier_avatar,
			COALESCE(replier.role::text, '') AS replier_role`).
		Order("feedbacks.created_at DESC").
		Offset((page - 1) * size).
		Limit(size).
		Scan(&items).Error
	if err != nil {
		return nil, 0, err
	}
	for i := range items {
		hydrateAdminUserBrief(&items[i].User, items[i].UserID, items[i].UserNickname, items[i].UserAvatarURL, items[i].UserRole)
		hydrateOptionalUserBrief(&items[i].Replier, items[i].RepliedBy, items[i].ReplierName, items[i].ReplierAvatar, items[i].ReplierRole)
		if len(items[i].RawAttachments) > 0 {
			_ = json.Unmarshal(items[i].RawAttachments, &items[i].Attachments)
		}
	}
	return items, total, nil
}

func (r *adminRepository) GetFeedbackByID(id int64) (*model.Feedbacks, error) {
	var item model.Feedbacks
	if err := r.db.First(&item, id).Error; err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *adminRepository) UpdateFeedback(feedback *model.Feedbacks) error {
	return r.db.Save(feedback).Error
}

func (r *adminRepository) ListUsers(status, role, keyword string, page, size int) ([]AdminUserItem, int64, error) {
	var items []AdminUserItem
	var total int64
	base := r.db.Table("users")
	if status != "" {
		base = base.Where("users.status = ?", status)
	}
	if role != "" {
		base = base.Where("users.role = ?", role)
	}
	if keyword = strings.TrimSpace(keyword); keyword != "" {
		like := "%" + keyword + "%"
		base = base.Where("users.nickname ILIKE ? OR COALESCE(users.email, '') ILIKE ? OR COALESCE(users.student_id, '') ILIKE ?", like, like, like)
	}
	if err := base.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	err := base.
		Select("id, email, student_id, nickname, avatar_url, role, status, ban_until, ban_reason, ban_source, violation_count, last_violation_at, email_verified, points, free_download_count, last_login_at, created_at, updated_at").
		Order("users.created_at DESC").
		Offset((page - 1) * size).
		Limit(size).
		Scan(&items).Error
	return items, total, err
}

func (r *adminRepository) ListUserViolations(userID int64, page, size int) ([]AdminUserViolationItem, int64, error) {
	var items []AdminUserViolationItem
	var total int64
	base := r.db.Table("user_violations").Where("user_id = ?", userID)
	if err := base.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	err := base.
		Select("id, user_id, scope, trigger_key, reason, evidence, penalty_level, ban_duration_seconds, created_at").
		Order("created_at DESC").
		Offset((page - 1) * size).
		Limit(size).
		Scan(&items).Error
	return items, total, err
}

func (r *adminRepository) GetUserByID(id int64) (*model.Users, error) {
	var item model.Users
	if err := r.db.First(&item, id).Error; err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *adminRepository) UpdateUser(user *model.Users) error {
	return r.db.Save(user).Error
}

func (r *adminRepository) AdjustUserPoints(userID int64, delta int, reason string) (int, error) {
	var balance int
	err := r.db.Transaction(func(tx *gorm.DB) error {
		var user model.Users
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", userID).First(&user).Error; err != nil {
			return err
		}
		balance = user.Points + delta
		if balance < 0 {
			balance = 0
			delta = -user.Points
		}
		if err := tx.Model(&model.Users{}).Where("id = ?", userID).Update("points", balance).Error; err != nil {
			return err
		}
		return tx.Create(&model.PointsRecords{
			UserID:    userID,
			Type:      model.PointsTypeManual,
			Delta:     delta,
			Balance:   balance,
			Reason:    reason,
			RelatedID: 0,
		}).Error
	})
	return balance, err
}

func (r *adminRepository) ListAnnouncements(page, size int) ([]AdminAnnouncementItem, int64, error) {
	var items []AdminAnnouncementItem
	var total int64
	base := r.db.Unscoped().Table("announcements")
	if err := base.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	err := base.Select(adminAnnouncementItemColumns).
		Order("is_pinned DESC").
		Order("created_at DESC").
		Offset((page - 1) * size).
		Limit(size).
		Scan(&items).Error
	return items, total, err
}

func (r *adminRepository) CreateAnnouncement(item *model.Announcements) error {
	return r.db.Create(item).Error
}

func (r *adminRepository) GetAnnouncementByID(id int64) (*model.Announcements, error) {
	var item model.Announcements
	if err := r.db.First(&item, id).Error; err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *adminRepository) UpdateAnnouncement(item *model.Announcements) error {
	return r.db.Save(item).Error
}

func (r *adminRepository) DeleteAnnouncement(id int64) error {
	return r.db.Delete(&model.Announcements{}, id).Error
}

func (r *adminRepository) ListCourses(status, courseType, keyword string, page, size int) ([]AdminCourseItem, int64, error) {
	var items []AdminCourseItem
	var total int64
	base := r.db.Table("courses")
	if status != "" {
		base = base.Where("courses.status = ?", status)
	}
	if courseType != "" {
		base = base.Where("courses.course_type = ?", courseType)
	}
	if keyword = strings.TrimSpace(keyword); keyword != "" {
		base = base.Where("courses.name ILIKE ?", "%"+keyword+"%")
	}
	if err := base.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	err := base.Select(adminCourseItemColumns).
		Order("courses.created_at DESC").
		Offset((page - 1) * size).
		Limit(size).
		Scan(&items).Error
	return items, total, err
}

func (r *adminRepository) CreateCourse(course *model.Courses) error {
	return r.db.Create(course).Error
}

func (r *adminRepository) GetCourseByID(id int64) (*model.Courses, error) {
	var item model.Courses
	if err := r.db.First(&item, id).Error; err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *adminRepository) UpdateCourseFields(id int64, updates map[string]interface{}) error {
	return r.db.Model(&model.Courses{}).Where("id = ?", id).Updates(updates).Error
}

func (r *adminRepository) ListCourseRelations(courseID int64) ([]AdminCourseRelationItem, error) {
	var items []AdminCourseRelationItem
	err := r.db.Table("course_teachers").
		Joins("JOIN teachers ON teachers.id = course_teachers.teacher_id").
		Where("course_teachers.course_id = ? AND course_teachers.status = ? AND teachers.status = ?", courseID, model.CourseTeacherRelationStatusActive, "active").
		Select(`
			course_teachers.course_id,
			course_teachers.teacher_id,
			teachers.name AS teacher_name,
			teachers.title,
			teachers.avatar_url,
			course_teachers.status AS relation_status,
			course_teachers.created_at,
			course_teachers.updated_at,
			course_teachers.canceled_at`).
		Order("course_teachers.updated_at DESC, teachers.id ASC").
		Scan(&items).Error
	return items, err
}

func (r *adminRepository) ListTeachers(status, keyword string, departmentID *int16, page, size int) ([]AdminTeacherItem, int64, error) {
	var items []AdminTeacherItem
	var total int64
	base := r.db.Table("teachers").Joins("LEFT JOIN departments ON departments.id = teachers.department_id")
	if status != "" {
		base = base.Where("teachers.status = ?", status)
	}
	if departmentID != nil {
		base = base.Where("teachers.department_id = ?", *departmentID)
	}
	if keyword = strings.TrimSpace(keyword); keyword != "" {
		base = base.Where("teachers.name ILIKE ?", "%"+keyword+"%")
	}
	if err := base.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	err := base.Select(adminTeacherItemColumns).
		Order("teachers.created_at DESC").
		Offset((page - 1) * size).
		Limit(size).
		Scan(&items).Error
	return items, total, err
}

func (r *adminRepository) CreateTeacher(teacher *model.Teachers) error {
	return r.db.Create(teacher).Error
}

func (r *adminRepository) GetTeacherByID(id int64) (*model.Teachers, error) {
	var item model.Teachers
	if err := r.db.First(&item, id).Error; err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *adminRepository) UpdateTeacherFields(id int64, updates map[string]interface{}) error {
	return r.db.Model(&model.Teachers{}).Where("id = ?", id).Updates(updates).Error
}

func (r *adminRepository) ListTeacherRelations(teacherID int64) ([]AdminTeacherRelationItem, error) {
	var items []AdminTeacherRelationItem
	err := r.db.Table("course_teachers").
		Joins("JOIN courses ON courses.id = course_teachers.course_id").
		Where("course_teachers.teacher_id = ? AND course_teachers.status = ? AND courses.status = ?", teacherID, model.CourseTeacherRelationStatusActive, model.CourseStatusActive).
		Select(`
			course_teachers.teacher_id,
			course_teachers.course_id,
			courses.name AS course_name,
			courses.course_type,
			course_teachers.status AS relation_status,
			course_teachers.created_at,
			course_teachers.updated_at,
			course_teachers.canceled_at`).
		Order("course_teachers.updated_at DESC, courses.id ASC").
		Scan(&items).Error
	return items, err
}

func (r *adminRepository) ListResources(status, keyword, resourceType string, courseID int64, page, size int) ([]AdminResourceItem, int64, error) {
	var items []AdminResourceItem
	var total int64
	// 用 LEFT JOIN 而非 INNER JOIN 纯粹是防御性的：
	// 当前 resources.course_id / uploader_id 都有外键约束，且课程与用户都是软删除，
	// 所以孤儿资源不可能出现，两种写法结果一致（已在本地库核对过 190 == 190）。
	// 但外键没有 ON DELETE 行为，一旦将来迁移改成级联删除，INNER JOIN 会让孤儿资源
	// 同时从 items 和 total 中静默消失——管理员无从发现，也就无法清理。
	base := r.db.Table("resources").
		Joins("LEFT JOIN courses ON courses.id = resources.course_id").
		Joins("LEFT JOIN users ON users.id = resources.uploader_id")
	if status != "" {
		base = base.Where("resources.status = ?", status)
	}
	if courseID > 0 {
		base = base.Where("resources.course_id = ?", courseID)
	}
	if resourceType != "" {
		base = base.Where("resources.type = ?", resourceType)
	}
	if keyword = strings.TrimSpace(keyword); keyword != "" {
		like := "%" + keyword + "%"
		base = base.Where("resources.title ILIKE ? OR courses.name ILIKE ? OR users.nickname ILIKE ?", like, like, like)
	}
	if err := base.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	err := base.Select(`
		resources.id,
		resources.title,
		resources.course_id,
		COALESCE(courses.name, '') AS course_name,
		resources.uploader_id,
		COALESCE(users.nickname, '') AS uploader_name,
		resources.type AS resource_type,
		resources.status,
		resources.download_count AS downloads,
		resources.view_count AS views,
		resources.like_count AS likes,
		resources.comment_count,
		resources.created_at,
		resources.updated_at`).
		Order("resources.created_at DESC").
		Offset((page - 1) * size).
		Limit(size).
		Scan(&items).Error
	return items, total, err
}

func (r *adminRepository) GetResourceByID(id int64) (*model.Resources, error) {
	var item model.Resources
	if err := r.db.First(&item, id).Error; err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *adminRepository) UpdateResourceStatus(id int64, status model.ResourceStatus) error {
	return r.db.Model(&model.Resources{}).Where("id = ?", id).Updates(map[string]interface{}{
		"status":     status,
		"updated_at": time.Now(),
	}).Error
}

func (r *adminRepository) ListAuditLogs(action string, operatorID int64, targetType string, page, size int) ([]AdminAuditLogItem, int64, error) {
	var items []AdminAuditLogItem
	var total int64
	base := r.db.Table("audit_logs")
	if action != "" {
		base = base.Where("audit_logs.action = ?", action)
	}
	if operatorID > 0 {
		base = base.Where("audit_logs.operator_id = ?", operatorID)
	}
	if targetType != "" {
		base = base.Where("audit_logs.target_type = ?", targetType)
	}
	if err := base.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	err := base.
		Joins("LEFT JOIN users ON users.id = audit_logs.operator_id").
		Select(`
			audit_logs.id,
			audit_logs.operator_id,
			audit_logs.action,
			audit_logs.target_type,
			audit_logs.target_id,
			audit_logs.old_values,
			audit_logs.new_values,
			audit_logs.reason,
			audit_logs.created_at,
			COALESCE(users.nickname, '') AS operator_name,
			COALESCE(users.avatar_url, '') AS operator_avatar,
			COALESCE(users.role::text, '') AS operator_role`).
		Order("audit_logs.created_at DESC").
		Offset((page - 1) * size).
		Limit(size).
		Scan(&items).Error
	if err != nil {
		return nil, 0, err
	}
	for i := range items {
		hydrateAdminUserBrief(&items[i].Operator, items[i].OperatorID, items[i].OperatorName, items[i].OperatorAvatar, items[i].OperatorRole)
	}
	if err := r.hydrateAuditTargetNames(items); err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

// auditTargetNameSources 把 target_type 映射到「去哪张表、取哪一列当名称」。
// 未列出的类型（report / correction / feedback / notification / course_teacher_relation）
// 本身没有可读名称，保持 target_name 为空即可。
var auditTargetNameSources = map[string]struct{ table, column string }{
	"announcement": {"announcements", "title"},
	"course":       {"courses", "name"},
	"teacher":      {"teachers", "name"},
	"user":         {"users", "nickname"},
	"resource":     {"resources", "title"},
}

// hydrateAuditTargetNames 按 target_type 分组后每组一条 IN 查询回填名称。
// 必须批量：审计日志一页最多 50 行，逐行查会变成 50 次往返。
func (r *adminRepository) hydrateAuditTargetNames(items []AdminAuditLogItem) error {
	if len(items) == 0 {
		return nil
	}
	idsByType := make(map[string][]int64)
	for i := range items {
		if _, ok := auditTargetNameSources[items[i].TargetType]; !ok || items[i].TargetID == 0 {
			continue
		}
		idsByType[items[i].TargetType] = append(idsByType[items[i].TargetType], items[i].TargetID)
	}
	if len(idsByType) == 0 {
		return nil
	}

	names := make(map[string]map[int64]string, len(idsByType))
	for targetType, ids := range idsByType {
		source := auditTargetNameSources[targetType]
		var rows []struct {
			ID   int64
			Name string
		}
		err := r.db.Table(source.table).
			Select("id, COALESCE("+source.column+", '') AS name").
			Where("id IN ?", ids).
			Scan(&rows).Error
		if err != nil {
			return err
		}
		lookup := make(map[int64]string, len(rows))
		for _, row := range rows {
			lookup[row.ID] = row.Name
		}
		names[targetType] = lookup
	}

	for i := range items {
		if lookup, ok := names[items[i].TargetType]; ok {
			items[i].TargetName = lookup[items[i].TargetID]
		}
	}
	return nil
}

func (r *adminRepository) CreateAuditLog(log *model.AuditLogs) error {
	return r.db.Create(log).Error
}

func (r *adminRepository) CreateNotification(notification *model.Notifications) error {
	if err := r.db.Create(notification).Error; err != nil {
		return err
	}
	realtime.PublishNewNotification(notification)
	return nil
}

/*
 * 列表与「创建/更新后回读」共用同一套 SELECT。
 *
 * 此前 create/update 直接返回原始 GORM model，字段集与列表 DTO 不一致：
 * teacher 的 bio/tutor_type/homepage_url 只在列表里有（从 metadata 拍平），
 * course.code 只在 create/update 里有，公告的 expires_at 在 create 时还会返回
 * 零值 "0001-01-01T00:00:00Z" 而列表里是直接省略——同一条记录两种表示。
 */
const adminCourseItemColumns = "id, name, course_type, description, credits, resource_count, eval_count, favorite_count, status, created_at, updated_at"

const adminAnnouncementItemColumns = "id, title, content, type, is_pinned, is_published, published_at, expires_at, created_at, updated_at, deleted_at"

const adminTeacherItemColumns = `
		teachers.id,
		teachers.name,
		teachers.title,
		teachers.department_id,
		COALESCE(departments.name, '') AS department_name,
		teachers.avatar_url,
		COALESCE(teachers.metadata->>'bio', '') AS bio,
		COALESCE(teachers.metadata->>'tutor_type', '') AS tutor_type,
		COALESCE(teachers.metadata->>'homepage_url', '') AS homepage_url,
		teachers.favorite_count,
		teachers.eval_count,
		teachers.status,
		teachers.created_at,
		teachers.updated_at`

func (r *adminRepository) GetAdminCourseItem(id int64) (*AdminCourseItem, error) {
	var item AdminCourseItem
	err := r.db.Table("courses").Where("id = ?", id).Select(adminCourseItemColumns).Scan(&item).Error
	if err != nil {
		return nil, err
	}
	if item.ID == 0 {
		return nil, gorm.ErrRecordNotFound
	}
	return &item, nil
}

func (r *adminRepository) GetAdminTeacherItem(id int64) (*AdminTeacherItem, error) {
	var item AdminTeacherItem
	err := r.db.Table("teachers").
		Joins("LEFT JOIN departments ON departments.id = teachers.department_id").
		Where("teachers.id = ?", id).
		Select(adminTeacherItemColumns).
		Scan(&item).Error
	if err != nil {
		return nil, err
	}
	if item.ID == 0 {
		return nil, gorm.ErrRecordNotFound
	}
	return &item, nil
}

func (r *adminRepository) GetAdminAnnouncementItem(id int64) (*AdminAnnouncementItem, error) {
	var item AdminAnnouncementItem
	// Unscoped：软删除的公告也要能回读，否则删除后无法确认结果
	err := r.db.Unscoped().Table("announcements").Where("id = ?", id).
		Select(adminAnnouncementItemColumns).Scan(&item).Error
	if err != nil {
		return nil, err
	}
	if item.ID == 0 {
		return nil, gorm.ErrRecordNotFound
	}
	return &item, nil
}

func hydrateAdminUserBrief(target **UserBrief, id int64, nickname, avatarURL, role string) {
	*target = &UserBrief{
		ID:        id,
		Nickname:  nickname,
		AvatarURL: avatarURL,
		Role:      role,
	}
}

// hydrateOptionalUserBrief 用于处理人 / 回复人这类可空的关联用户。
// 与 hydrateAdminUserBrief 的区别是：id 为空时把指针置 nil，让 JSON 直接省略该字段，
// 而不是输出 {"id":"0","nickname":""} —— 后者会让前端把「尚未处理」渲染成一个空白的处理人。
func hydrateOptionalUserBrief(target **UserBrief, id *int64, nickname, avatarURL, role string) {
	if id == nil || *id == 0 {
		*target = nil
		return
	}
	*target = &UserBrief{
		ID:        *id,
		Nickname:  nickname,
		AvatarURL: avatarURL,
		Role:      role,
	}
}

func (r *adminRepository) getReportTargetPreview(targetType string, targetID int64) (*AdminReportTargetPreview, error) {
	switch model.ReportTargetType(targetType) {
	case model.ReportTargetTypeResource:
		return r.getResourceReportPreview(targetID)
	case model.ReportTargetTypeCourse:
		return r.getCourseReportPreview(targetID)
	case model.ReportTargetTypeTeacherEvaluation:
		return r.getTeacherEvaluationReportPreview(targetID)
	case model.ReportTargetTypeCourseEvaluation:
		return r.getCourseEvaluationReportPreview(targetID)
	case model.ReportTargetTypeTeacherReply:
		return r.getTeacherReplyReportPreview(targetID)
	case model.ReportTargetTypeCourseReply:
		return r.getCourseReplyReportPreview(targetID)
	case model.ReportTargetTypeComment:
		return r.getCommentReportPreview(targetID)
	default:
		return missingReportTargetPreview("暂不支持预览该举报目标"), nil
	}
}

func (r *adminRepository) getResourceReportPreview(targetID int64) (*AdminReportTargetPreview, error) {
	var item struct {
		Title        string
		Description  string
		Status       string
		UploaderName string
		CreatedAt    time.Time
	}

	err := r.db.Table("resources").
		Joins("LEFT JOIN users uploader ON uploader.id = resources.uploader_id").
		Select(`
			resources.title,
			COALESCE(resources.description, '') AS description,
			resources.status::text AS status,
			COALESCE(uploader.nickname, '') AS uploader_name,
			resources.created_at`).
		Where("resources.id = ?", targetID).
		Take(&item).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return missingReportTargetPreview("被举报资源不存在或已不可见"), nil
		}
		return nil, err
	}

	return &AdminReportTargetPreview{
		Title:      fallbackText(item.Title, "未命名资源"),
		Subtitle:   joinPreviewParts("资源", item.UploaderName),
		Content:    fallbackText(summarizePreviewText(item.Description, 180), "该资源未填写说明。"),
		Status:     item.Status,
		AuthorName: item.UploaderName,
		CreatedAt:  &item.CreatedAt,
	}, nil
}

func (r *adminRepository) getCourseReportPreview(targetID int64) (*AdminReportTargetPreview, error) {
	var item struct {
		Name        string
		Description string
		Status      string
		CreatedAt   time.Time
	}

	err := r.db.Table("courses").
		Select(`
			courses.name,
			COALESCE(courses.description, '') AS description,
			courses.status::text AS status,
			courses.created_at`).
		Where("courses.id = ?", targetID).
		Take(&item).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return missingReportTargetPreview("被举报课程不存在或已不可见"), nil
		}
		return nil, err
	}

	return &AdminReportTargetPreview{
		Title:     fallbackText(item.Name, "未命名课程"),
		Subtitle:  "课程",
		Content:   fallbackText(summarizePreviewText(item.Description, 180), "该课程未填写简介。"),
		Status:    item.Status,
		CreatedAt: &item.CreatedAt,
	}, nil
}

func (r *adminRepository) getTeacherEvaluationReportPreview(targetID int64) (*AdminReportTargetPreview, error) {
	var item struct {
		Comment     string
		Status      string
		AuthorName  string
		TeacherName string
		CourseName  string
		CreatedAt   time.Time
	}

	err := r.db.Table("teacher_evaluations").
		Joins("LEFT JOIN users author ON author.id = teacher_evaluations.user_id").
		Joins("LEFT JOIN teachers ON teachers.id = teacher_evaluations.teacher_id").
		Joins("LEFT JOIN courses ON courses.id = teacher_evaluations.course_id").
		Select(`
			COALESCE(teacher_evaluations.comment, '') AS comment,
			teacher_evaluations.status::text AS status,
			COALESCE(author.nickname, '') AS author_name,
			COALESCE(teachers.name, '') AS teacher_name,
			COALESCE(courses.name, '') AS course_name,
			teacher_evaluations.created_at`).
		Where("teacher_evaluations.id = ?", targetID).
		Take(&item).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return missingReportTargetPreview("被举报评价不存在或已不可见"), nil
		}
		return nil, err
	}

	return &AdminReportTargetPreview{
		Title:      "教师评价",
		Subtitle:   joinPreviewParts(item.TeacherName, item.CourseName),
		Content:    fallbackText(summarizePreviewText(item.Comment, 180), "该评价未填写文字内容。"),
		Status:     item.Status,
		AuthorName: item.AuthorName,
		CreatedAt:  &item.CreatedAt,
	}, nil
}

func (r *adminRepository) getCourseEvaluationReportPreview(targetID int64) (*AdminReportTargetPreview, error) {
	var item struct {
		Comment     string
		Status      string
		AuthorName  string
		CourseName  string
		TeacherName string
		CreatedAt   time.Time
	}

	err := r.db.Table("course_evaluations").
		Joins("LEFT JOIN users author ON author.id = course_evaluations.user_id").
		Joins("LEFT JOIN courses ON courses.id = course_evaluations.course_id").
		Joins("LEFT JOIN teachers ON teachers.id = course_evaluations.teacher_id").
		Select(`
			COALESCE(course_evaluations.comment, '') AS comment,
			course_evaluations.status::text AS status,
			COALESCE(author.nickname, '') AS author_name,
			COALESCE(courses.name, '') AS course_name,
			COALESCE(teachers.name, '') AS teacher_name,
			course_evaluations.created_at`).
		Where("course_evaluations.id = ?", targetID).
		Take(&item).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return missingReportTargetPreview("被举报评价不存在或已不可见"), nil
		}
		return nil, err
	}

	return &AdminReportTargetPreview{
		Title:      "课程评价",
		Subtitle:   joinPreviewParts(item.CourseName, item.TeacherName),
		Content:    fallbackText(summarizePreviewText(item.Comment, 180), "该评价未填写文字内容。"),
		Status:     item.Status,
		AuthorName: item.AuthorName,
		CreatedAt:  &item.CreatedAt,
	}, nil
}

func (r *adminRepository) getTeacherReplyReportPreview(targetID int64) (*AdminReportTargetPreview, error) {
	var item struct {
		Content     string
		AuthorName  string
		TeacherName string
		CourseName  string
		CreatedAt   time.Time
	}

	err := r.db.Table("teacher_evaluation_replies").
		Joins("LEFT JOIN users author ON author.id = teacher_evaluation_replies.user_id").
		Joins("LEFT JOIN teacher_evaluations ON teacher_evaluations.id = teacher_evaluation_replies.evaluation_id").
		Joins("LEFT JOIN teachers ON teachers.id = teacher_evaluations.teacher_id").
		Joins("LEFT JOIN courses ON courses.id = teacher_evaluations.course_id").
		Select(`
			teacher_evaluation_replies.content,
			COALESCE(author.nickname, '') AS author_name,
			COALESCE(teachers.name, '') AS teacher_name,
			COALESCE(courses.name, '') AS course_name,
			teacher_evaluation_replies.created_at`).
		Where("teacher_evaluation_replies.id = ?", targetID).
		Take(&item).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return missingReportTargetPreview("被举报回复不存在或已不可见"), nil
		}
		return nil, err
	}

	return &AdminReportTargetPreview{
		Title:      "教师评价回复",
		Subtitle:   joinPreviewParts(item.TeacherName, item.CourseName),
		Content:    fallbackText(summarizePreviewText(item.Content, 180), "该回复未填写文字内容。"),
		AuthorName: item.AuthorName,
		CreatedAt:  &item.CreatedAt,
	}, nil
}

func (r *adminRepository) getCourseReplyReportPreview(targetID int64) (*AdminReportTargetPreview, error) {
	var item struct {
		Content     string
		AuthorName  string
		CourseName  string
		TeacherName string
		CreatedAt   time.Time
	}

	err := r.db.Table("course_evaluation_replies").
		Joins("LEFT JOIN users author ON author.id = course_evaluation_replies.user_id").
		Joins("LEFT JOIN course_evaluations ON course_evaluations.id = course_evaluation_replies.evaluation_id").
		Joins("LEFT JOIN courses ON courses.id = course_evaluations.course_id").
		Joins("LEFT JOIN teachers ON teachers.id = course_evaluations.teacher_id").
		Select(`
			course_evaluation_replies.content,
			COALESCE(author.nickname, '') AS author_name,
			COALESCE(courses.name, '') AS course_name,
			COALESCE(teachers.name, '') AS teacher_name,
			course_evaluation_replies.created_at`).
		Where("course_evaluation_replies.id = ?", targetID).
		Take(&item).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return missingReportTargetPreview("被举报回复不存在或已不可见"), nil
		}
		return nil, err
	}

	return &AdminReportTargetPreview{
		Title:      "课程评价回复",
		Subtitle:   joinPreviewParts(item.CourseName, item.TeacherName),
		Content:    fallbackText(summarizePreviewText(item.Content, 180), "该回复未填写文字内容。"),
		AuthorName: item.AuthorName,
		CreatedAt:  &item.CreatedAt,
	}, nil
}

func (r *adminRepository) getCommentReportPreview(targetID int64) (*AdminReportTargetPreview, error) {
	var item struct {
		Content       string
		Status        string
		AuthorName    string
		TargetType    string
		ResourceTitle string
		CourseName    string
		TeacherName   string
		CreatedAt     time.Time
	}

	err := r.db.Table("comments").
		Joins("LEFT JOIN users author ON author.id = comments.user_id").
		Joins("LEFT JOIN resources ON resources.id = comments.target_id AND comments.target_type = 'resource'").
		Joins("LEFT JOIN courses ON courses.id = comments.target_id AND comments.target_type = 'course'").
		Joins("LEFT JOIN teachers ON teachers.id = comments.target_id AND comments.target_type = 'teacher'").
		Select(`
			comments.content,
			comments.status::text AS status,
			COALESCE(author.nickname, '') AS author_name,
			comments.target_type::text AS target_type,
			COALESCE(resources.title, '') AS resource_title,
			COALESCE(courses.name, '') AS course_name,
			COALESCE(teachers.name, '') AS teacher_name,
			comments.created_at`).
		Where("comments.id = ?", targetID).
		Take(&item).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return missingReportTargetPreview("被举报评论不存在或已不可见"), nil
		}
		return nil, err
	}

	subtitle := "评论"
	switch item.TargetType {
	case string(model.CommentTargetTypeResource):
		subtitle = joinPreviewParts("资源评论", item.ResourceTitle)
	case string(model.CommentTargetTypeCourse):
		subtitle = joinPreviewParts("课程评论", item.CourseName)
	case string(model.CommentTargetTypeTeacher):
		subtitle = joinPreviewParts("教师评论", item.TeacherName)
	}

	return &AdminReportTargetPreview{
		Title:      "评论内容",
		Subtitle:   subtitle,
		Content:    fallbackText(summarizePreviewText(item.Content, 180), "该评论未填写文字内容。"),
		Status:     item.Status,
		AuthorName: item.AuthorName,
		CreatedAt:  &item.CreatedAt,
	}, nil
}

func missingReportTargetPreview(message string) *AdminReportTargetPreview {
	return &AdminReportTargetPreview{
		Title:   message,
		Missing: true,
	}
}

func summarizePreviewText(value string, max int) string {
	value = strings.Join(strings.Fields(strings.TrimSpace(value)), " ")
	if value == "" || max <= 0 {
		return value
	}
	runes := []rune(value)
	if len(runes) <= max {
		return value
	}
	return string(runes[:max]) + "..."
}

func joinPreviewParts(parts ...string) string {
	filtered := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		filtered = append(filtered, part)
	}
	return strings.Join(filtered, " / ")
}

func fallbackText(value, fallback string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback
	}
	return value
}
