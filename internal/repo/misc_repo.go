package repo

import (
	"csu-star-backend/internal/model"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var ErrAlreadyCheckedIn = errors.New("already checked in")

var contributionDayLocation = loadContributionDayLocation()

type MeProfile struct {
	ID                int64          `json:"id,string"`
	Email             string         `json:"email"`
	EmailVerified     bool           `json:"email_verified"`
	Nickname          string         `json:"nickname"`
	AvatarURL         string         `json:"avatar_url"`
	DepartmentID      *int16         `json:"department_id,omitempty"`
	Grade             *int           `json:"grade,omitempty"`
	Role              string         `json:"role"`
	Points            int            `json:"points"`
	FreeDownloadCount int            `json:"free_download_count"`
	OauthBindings     OauthBindings  `json:"oauth_bindings" gorm:"-"`
	QQ                bool           `json:"-" gorm:"column:qq"`
	Github            bool           `json:"-" gorm:"column:github"`
	Google            bool           `json:"-" gorm:"column:google"`
	Metadata          datatypes.JSON `json:"-" gorm:"column:metadata"`
	CreatedAt         time.Time      `json:"created_at"`
}

type userProfileMetadata struct {
	DepartmentID *int16 `json:"department_id,omitempty"`
	Grade        *int   `json:"grade,omitempty"`
}

type OauthBindings struct {
	QQ     bool `json:"qq"`
	Github bool `json:"github"`
	Google bool `json:"google"`
}

type DownloadHistoryItem struct {
	ID         int64     `json:"id,string"`
	ResourceID int64     `json:"resource_id,string"`
	Title      string    `json:"title"`
	CourseID   int64     `json:"course_id,string"`
	CourseName string    `json:"course_name"`
	CreatedAt  time.Time `json:"created_at"`
}

type FavoriteItem struct {
	ID         int64     `json:"id,string"`
	TargetType string    `json:"target_type"`
	TargetID   int64     `json:"target_id,string"`
	Name       string    `json:"name"`
	CreatedAt  time.Time `json:"created_at"`
}

type PointRecordItem struct {
	ID        int64     `json:"id,string"`
	Type      string    `json:"type"`
	Delta     int       `json:"delta"`
	Balance   int       `json:"balance"`
	Reason    string    `json:"reason"`
	RelatedID int64     `json:"related_id,string"`
	CreatedAt time.Time `json:"created_at"`
}

type ContributionEvent struct {
	EventType  string    `json:"event_type"`
	Label      string    `json:"label"`
	Score      int       `json:"score"`
	OccurredAt time.Time `json:"occurred_at"`
}

type ContributionAction struct {
	Type  string `json:"type"`
	Label string `json:"label"`
	Score int    `json:"score"`
}

type ContributionCell struct {
	Date     string               `json:"date"`
	Score    int                  `json:"score"`
	Level    int                  `json:"level"`
	IsFuture bool                 `json:"is_future"`
	Actions  []ContributionAction `json:"actions"`
}

type ContributionSummary struct {
	Weeks         [][]ContributionCell `json:"weeks"`
	TotalScore    int                  `json:"total_score"`
	ActiveDays    int                  `json:"active_days"`
	CurrentStreak int                  `json:"current_streak"`
	MaxDayScore   int                  `json:"max_day_score"`
}

type UserContributionProfile struct {
	UserID       int64 `json:"user_id,string"`
	Contribution int   `json:"contribution"`
	Level        int   `json:"level"`
}

type AnnouncementItem struct {
	ID          int64     `json:"id,string"`
	Title       string    `json:"title"`
	Content     string    `json:"content"`
	Type        string    `json:"type"`
	IsPinned    bool      `json:"is_pinned"`
	PublishedAt time.Time `json:"published_at"`
}

type SearchResultItem = GlobalSearchItem

type NotificationItem struct {
	ID         int64          `json:"id,string"`
	Type       string         `json:"type"`
	Category   string         `json:"category"`
	Result     string         `json:"result"`
	Title      string         `json:"title"`
	Content    string         `json:"content"`
	RelatedID  int64          `json:"related_id,string"`
	SourceType string         `json:"source_type"`
	SourceID   int64          `json:"source_id,string"`
	IsRead     bool           `json:"is_read"`
	IsGlobal   bool           `json:"is_global"`
	IsPinned   bool           `json:"is_pinned"`
	Metadata   datatypes.JSON `json:"metadata"`
	CreatedAt  time.Time      `json:"created_at"`
}

type HomeNotificationSummary struct {
	Announcements  []NotificationItem `json:"announcements"`
	Interactions   []NotificationItem `json:"interactions"`
	SystemMessages []NotificationItem `json:"system_messages"`
}

type ShowcaseStats struct {
	UserCount       int64 `json:"user_count"`
	ResourceCount   int64 `json:"resource_count"`
	EvaluationCount int64 `json:"evaluation_count"`
	TeacherCount    int64 `json:"teacher_count"`
	CourseCount     int64 `json:"course_count"`
}

type MiscRepository interface {
	GetMe(userID int64) (*MeProfile, error)
	UpdateMe(userID int64, nickname, avatarURL string, departmentID *int16, grade *int) error
	DailyCheckin(userID int64) (int, error)
	ListMyDownloads(userID int64, page, size int) ([]DownloadHistoryItem, int64, error)
	ListMyFavorites(userID int64, targetType string, page, size int) ([]FavoriteItem, int64, error)
	ListMyPoints(userID int64, page, size int) ([]PointRecordItem, int64, error)
	ListMyContributionEvents(userID int64, start, end time.Time) ([]ContributionEvent, error)
	GetUserContributionProfile(userID int64) (*UserContributionProfile, error)
	ListAnnouncements() ([]AnnouncementItem, error)
	GetShowcaseStats() (*ShowcaseStats, error)
	CreateFeedback(feedback *model.Feedbacks) error
	CreateReport(report *model.Reports) error
	CreateCorrection(correction *model.Corrections) error
	Search(q, searchType string, page, size int, relevanceFirst bool) ([]SearchResultItem, int64, error)
	ListNotifications(userID int64, isRead *bool, page, size int) ([]NotificationItem, int64, error)
	ListHomeNotificationSummary(userID int64) (*HomeNotificationSummary, error)
	CountUnreadNotifications(userID int64) (int64, error)
	MarkNotificationRead(userID, notificationID int64) error
	MarkAllNotificationsRead(userID int64) error
	CreateNotification(notification *model.Notifications) error
	PurgeExpiredNotifications(now time.Time) error
	CreateTeacherSupplementRequest(request *model.TeacherSupplementRequests) error
	GetTeacherSupplementRequestByID(id int64) (*TeacherSupplementRequestItem, error)
	ListTeacherSupplementRequests(query SupplementRequestListQuery) ([]TeacherSupplementRequestItem, int64, error)
	UpdateTeacherSupplementRequest(id int64, updates map[string]interface{}) error
	CreateCourseSupplementRequest(request *model.CourseSupplementRequests) error
	GetCourseSupplementRequestByID(id int64) (*CourseSupplementRequestItem, error)
	ListCourseSupplementRequests(query SupplementRequestListQuery) ([]CourseSupplementRequestItem, int64, error)
	UpdateCourseSupplementRequest(id int64, updates map[string]interface{}) error
}

type miscRepository struct {
	db *gorm.DB
}

func NewMiscRepository(db *gorm.DB) MiscRepository {
	return &miscRepository{db: db}
}

func (r *miscRepository) GetMe(userID int64) (*MeProfile, error) {
	var item MeProfile
	err := r.db.Table("users").Select(`
		id,
		email,
		email_verified,
		nickname,
		avatar_url,
		role,
		points,
		free_download_count,
		metadata,
		created_at,
		EXISTS (
			SELECT 1
			FROM user_oauth_bindings
			WHERE user_oauth_bindings.user_id = users.id AND provider = 'qq'
		) AS qq,
		EXISTS (
			SELECT 1
			FROM user_oauth_bindings
			WHERE user_oauth_bindings.user_id = users.id AND provider = 'github'
		) AS github,
		EXISTS (
			SELECT 1
			FROM user_oauth_bindings
			WHERE user_oauth_bindings.user_id = users.id AND provider = 'google'
		) AS google`).
		Where("id = ?", userID).
		Scan(&item).Error
	if err != nil {
		return nil, err
	}
	if item.ID == 0 {
		return nil, gorm.ErrRecordNotFound
	}
	if len(item.Metadata) > 0 {
		var metadata userProfileMetadata
		if err := json.Unmarshal(item.Metadata, &metadata); err == nil {
			item.DepartmentID = metadata.DepartmentID
			item.Grade = metadata.Grade
		}
	}
	item.OauthBindings = OauthBindings{
		QQ:     item.QQ,
		Github: item.Github,
		Google: item.Google,
	}
	return &item, nil
}

func (r *miscRepository) UpdateMe(userID int64, nickname, avatarURL string, departmentID *int16, grade *int) error {
	updates := map[string]interface{}{}
	if nickname != "" {
		updates["nickname"] = nickname
	}
	if avatarURL != "" {
		updates["avatar_url"] = avatarURL
	}

	if departmentID != nil || grade != nil {
		var current model.Users
		if err := r.db.Select("metadata").Where("id = ?", userID).First(&current).Error; err != nil {
			return err
		}

		metadata := userProfileMetadata{}
		if len(current.Metadata) > 0 {
			if err := json.Unmarshal(current.Metadata, &metadata); err != nil {
				return err
			}
		}
		if departmentID != nil {
			metadata.DepartmentID = departmentID
		}
		if grade != nil {
			metadata.Grade = grade
		}

		raw, err := json.Marshal(metadata)
		if err != nil {
			return err
		}
		updates["metadata"] = datatypes.JSON(raw)
	}

	if len(updates) == 0 {
		return nil
	}
	return r.db.Model(&model.Users{}).Where("id = ?", userID).Updates(updates).Error
}

func (r *miscRepository) DailyCheckin(userID int64) (int, error) {
	var balance int
	dayStart := startOfContributionDay(time.Now())
	dayEnd := dayStart.AddDate(0, 0, 1)
	err := r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec("SELECT pg_advisory_xact_lock(?)", userID).Error; err != nil {
			return err
		}

		var count int64
		if err := tx.Table("points_records").
			Where(
				"user_id = ? AND type = ? AND created_at >= ? AND created_at < ?",
				userID,
				model.PointsTypeCheckin,
				dayStart,
				dayEnd,
			).
			Count(&count).Error; err != nil {
			return err
		}
		if count > 0 {
			var user model.Users
			if err := tx.Select("points").Where("id = ?", userID).First(&user).Error; err != nil {
				return err
			}
			balance = user.Points
			return ErrAlreadyCheckedIn
		}

		var user model.Users
		if err := tx.Where("id = ?", userID).First(&user).Error; err != nil {
			return err
		}
		balance = user.Points + 1

		if err := tx.Model(&model.Users{}).
			Where("id = ?", userID).
			Update("points", gorm.Expr("points + 1")).Error; err != nil {
			return err
		}

		if err := tx.Create(&model.PointsRecords{
			UserID:    userID,
			Type:      model.PointsTypeCheckin,
			Delta:     1,
			Balance:   balance,
			Reason:    "每日签到获得积分",
			RelatedID: 0,
		}).Error; err != nil {
			return err
		}

		if err := ApplyUserContributionDeltaTx(tx, userID, 1); err != nil {
			return err
		}

		return tx.Create(&model.Notifications{
			UserID:    userID,
			Type:      model.NotificationPointsChanged,
			Category:  model.NotificationCategoryPoints,
			Result:    model.NotificationResultInform,
			Title:     "签到成功",
			Content:   "你已完成今日签到，获得 1 积分。",
			RelatedID: 0,
			IsRead:    true,
			IsGlobal:  false,
		}).Error
	})
	return balance, err
}

func loadContributionDayLocation() *time.Location {
	location, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		return time.FixedZone("Asia/Shanghai", 8*60*60)
	}
	return location
}

func startOfContributionDay(t time.Time) time.Time {
	localTime := t.In(contributionDayLocation)
	year, month, day := localTime.Date()
	return time.Date(year, month, day, 0, 0, 0, 0, contributionDayLocation)
}

func (r *miscRepository) ListMyDownloads(userID int64, page, size int) ([]DownloadHistoryItem, int64, error) {
	var items []DownloadHistoryItem
	var total int64
	base := r.db.Table("download_records").Where("download_records.user_id = ?", userID)
	if err := base.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	err := base.Joins("JOIN resources ON resources.id = download_records.resource_id").
		Joins("JOIN courses ON courses.id = resources.course_id").
		Select("download_records.id, download_records.resource_id, resources.title, resources.course_id, courses.name AS course_name, download_records.created_at").
		Order("download_records.created_at DESC").Offset((page - 1) * size).Limit(size).Scan(&items).Error
	return items, total, err
}

func (r *miscRepository) ListMyFavorites(userID int64, targetType string, page, size int) ([]FavoriteItem, int64, error) {
	var items []FavoriteItem
	var total int64
	base := r.db.Table("favorites").Where("favorites.user_id = ?", userID)
	if targetType != "" {
		base = base.Where("favorites.target_type = ?", targetType)
	}
	if err := base.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	err := base.Select(`
		favorites.id,
		favorites.target_type,
		favorites.target_id,
		COALESCE(resources.title, courses.name, teachers.name) AS name,
		favorites.created_at`).
		Joins("LEFT JOIN resources ON favorites.target_type = 'resource' AND resources.id = favorites.target_id").
		Joins("LEFT JOIN courses ON favorites.target_type = 'course' AND courses.id = favorites.target_id").
		Joins("LEFT JOIN teachers ON favorites.target_type = 'teacher' AND teachers.id = favorites.target_id").
		Order("favorites.created_at DESC").Offset((page - 1) * size).Limit(size).Scan(&items).Error
	return items, total, err
}

func (r *miscRepository) ListMyPoints(userID int64, page, size int) ([]PointRecordItem, int64, error) {
	var items []PointRecordItem
	var total int64
	base := r.db.Table("points_records").Where("user_id = ?", userID)
	if err := base.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	err := base.Select("id, type, delta, balance, reason, related_id, created_at").Order("created_at DESC").Offset((page - 1) * size).Limit(size).Scan(&items).Error
	return items, total, err
}

func (r *miscRepository) ListMyContributionEvents(userID int64, start, end time.Time) ([]ContributionEvent, error) {
	var items []ContributionEvent
	err := r.db.Raw(`
		SELECT event_type, label, score, occurred_at
		FROM (
			SELECT
				'resource_upload' AS event_type,
				'资源上传' AS label,
				2 AS score,
				created_at AS occurred_at
			FROM resources
			WHERE uploader_id = ? AND created_at >= ? AND created_at < ?

			UNION ALL

			SELECT
				'teacher_evaluation' AS event_type,
				'发布教师评价' AS label,
				1 AS score,
				created_at AS occurred_at
			FROM teacher_evaluations
			WHERE user_id = ? AND created_at >= ? AND created_at < ?

			UNION ALL

			SELECT
				'course_evaluation' AS event_type,
				'发布课程评价' AS label,
				1 AS score,
				created_at AS occurred_at
			FROM course_evaluations
			WHERE user_id = ? AND created_at >= ? AND created_at < ?

			UNION ALL

			SELECT
				'daily_checkin' AS event_type,
				'每日签到' AS label,
				1 AS score,
				created_at AS occurred_at
			FROM points_records
			WHERE user_id = ? AND type = ? AND created_at >= ? AND created_at < ?

			UNION ALL

			SELECT
				'invite_reward' AS event_type,
				'邀请奖励' AS label,
				3 AS score,
				created_at AS occurred_at
			FROM points_records
			WHERE user_id = ? AND type = ? AND created_at >= ? AND created_at < ?
		) AS contribution_events
		ORDER BY occurred_at ASC
	`,
		userID, start, end,
		userID, start, end,
		userID, start, end,
		userID, model.PointsTypeCheckin, start, end,
		userID, model.PointsTypeInvite, start, end,
	).Scan(&items).Error
	return items, err
}

func (r *miscRepository) GetUserContributionProfile(userID int64) (*UserContributionProfile, error) {
	var item UserContributionProfile
	err := r.db.Table("users AS u").
		Select(`
			u.id AS user_id,
			COALESCE(uc.contribution, 0) AS contribution,
			COALESCE(uc.level, 1) AS level
		`).
		Joins("LEFT JOIN user_contributions AS uc ON uc.user_id = u.id").
		Where("u.id = ?", userID).
		Scan(&item).Error
	if err != nil {
		return nil, err
	}
	if item.UserID == 0 {
		return nil, gorm.ErrRecordNotFound
	}
	return &item, nil
}

func (r *miscRepository) ListAnnouncements() ([]AnnouncementItem, error) {
	var items []AnnouncementItem
	err := r.db.Table("announcements").
		Where("deleted_at IS NULL").
		Where("is_published = ? AND (expires_at IS NULL OR expires_at > ?)", true, time.Now()).
		Select("id, title, content, type, is_pinned, published_at").
		Order("is_pinned DESC").Order("published_at DESC").Scan(&items).Error
	return items, err
}

func (r *miscRepository) GetShowcaseStats() (*ShowcaseStats, error) {
	var item ShowcaseStats
	err := r.db.Raw(`
		SELECT
			(SELECT COUNT(*) FROM users) AS user_count,
			(SELECT COUNT(*) FROM resources WHERE status <> 'deleted') AS resource_count,
			(
				(SELECT COUNT(*) FROM teacher_evaluations) +
				(SELECT COUNT(*) FROM course_evaluations)
			) AS evaluation_count,
			(SELECT COUNT(*) FROM teachers WHERE status = 'active') AS teacher_count,
			(SELECT COUNT(*) FROM courses WHERE status = 'active') AS course_count
	`).Scan(&item).Error
	if err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *miscRepository) CreateFeedback(feedback *model.Feedbacks) error {
	return r.db.Create(feedback).Error
}

func (r *miscRepository) CreateReport(report *model.Reports) error {
	return r.db.Create(report).Error
}

func (r *miscRepository) CreateCorrection(correction *model.Corrections) error {
	return r.db.Create(correction).Error
}

func (r *miscRepository) Search(q, searchType string, page, size int, relevanceFirst bool) ([]SearchResultItem, int64, error) {
	var items []SearchResultItem
	var total int64
	like := "%" + q + "%"
	trimmedQ := strings.TrimSpace(q)
	offset := (page - 1) * size

	if trimmedQ == "" {
		switch searchType {
		case "", "all":
			sql := `
				SELECT * FROM (
					SELECT
						'resource' AS type,
						courses.id AS id,
						courses.name AS title,
						'' AS name,
						'课程资源合集' AS subtitle,
						'' AS detail_path,
						'' AS resource_collection_path,
						courses.created_at,
						COALESCE(courses.resource_count, 0) AS metric_primary,
						COALESCE(courses.download_total, 0) AS metric_secondary
					FROM courses
					WHERE courses.status = 'active'
					UNION ALL
					SELECT
						'course' AS type,
						id,
						'' AS title,
						name,
						'' AS subtitle,
						'' AS detail_path,
						'' AS resource_collection_path,
						created_at,
						COALESCE(eval_count, 0) AS metric_primary,
						0 AS metric_secondary
					FROM courses
					WHERE courses.status = 'active'
					UNION ALL
					SELECT
						'teacher' AS type,
						id,
						'' AS title,
						name,
						'' AS subtitle,
						'' AS detail_path,
						'' AS resource_collection_path,
						created_at,
						COALESCE(eval_count, 0) AS metric_primary,
						0 AS metric_secondary
					FROM teachers
					WHERE teachers.status = 'active'
				) AS search_results
				ORDER BY metric_primary DESC, metric_secondary DESC, created_at DESC
				LIMIT ? OFFSET ?`
			countSQL := `
				SELECT
					(SELECT COUNT(*) FROM courses WHERE status = 'active') +
					(SELECT COUNT(*) FROM courses WHERE status = 'active') +
					(SELECT COUNT(*) FROM teachers WHERE status = 'active') AS total`
			if err := r.db.Raw(countSQL).Scan(&total).Error; err != nil {
				return nil, 0, err
			}
			err := r.db.Raw(sql, size, offset).Scan(&items).Error
			for i := range items {
				switch items[i].Type {
				case "resource":
					items[i].Name = items[i].Title
					items[i].ResourceCollectionPath = CourseResourceCollectionPath(items[i].ID)
					items[i].ResourceCollection = &CourseResourceCollectionCard{
						CourseID:               items[i].ID,
						CourseName:             items[i].Title,
						ResourceCollectionPath: CourseResourceCollectionPath(items[i].ID),
						CourseDetailPath:       CourseDetailPath(items[i].ID),
					}
				case "course":
					if items[i].Name == "" {
						items[i].Name = items[i].Title
					}
					items[i].DetailPath = CourseDetailPath(items[i].ID)
				case "teacher":
					if items[i].Name == "" {
						items[i].Name = items[i].Title
					}
					items[i].DetailPath = TeacherDetailPath(items[i].ID)
				}
			}
			if err == nil {
				if enrichErr := r.attachSearchRelations(items); enrichErr != nil {
					return nil, 0, enrichErr
				}
			}
			return items, total, err
		case "resource":
			base := r.db.Table("courses").Where("courses.status = 'active'")
			if err := base.Session(&gorm.Session{}).Distinct("courses.id").Count(&total).Error; err != nil {
				return nil, 0, err
			}
			err := base.Select(`
          'resource' AS type,
          courses.id AS id,
          courses.name AS title,
          '' AS name,
          '课程资源合集' AS subtitle,
          '' AS detail_path,
          '' AS resource_collection_path`).
				Order("COALESCE(courses.resource_count, 0) DESC").
				Order("COALESCE(courses.download_total, 0) DESC").
				Order("courses.created_at DESC").
				Offset(offset).Limit(size).Scan(&items).Error

			if err != nil {
				return nil, 0, err
			}

			for i := range items {
				items[i].Name = items[i].Title
				items[i].ResourceCollectionPath = CourseResourceCollectionPath(items[i].ID)
				items[i].ResourceCollection = &CourseResourceCollectionCard{
					CourseID:               items[i].ID,
					CourseName:             items[i].Title,
					ResourceCollectionPath: CourseResourceCollectionPath(items[i].ID),
					CourseDetailPath:       CourseDetailPath(items[i].ID),
				}
			}
			if enrichErr := r.attachSearchRelations(items); enrichErr != nil {
				return nil, 0, enrichErr
			}
			return items, total, nil
		case "course":
			base := r.db.Table("courses").Where("courses.status = 'active'")
			if err := base.Count(&total).Error; err != nil {
				return nil, 0, err
			}
			err := base.Select(`'course' AS type, id, '' AS title, name, '' AS subtitle, '' AS detail_path, '' AS resource_collection_path`).
				Order("COALESCE(eval_count, 0) DESC").
				Order("created_at DESC").
				Offset(offset).Limit(size).Scan(&items).Error
			for i := range items {
				items[i].DetailPath = CourseDetailPath(items[i].ID)
			}
			if err == nil {
				if enrichErr := r.attachSearchRelations(items); enrichErr != nil {
					return nil, 0, enrichErr
				}
			}
			return items, total, err
		default:
			base := r.db.Table("teachers").Where("teachers.status = 'active'")
			if err := base.Count(&total).Error; err != nil {
				return nil, 0, err
			}
			err := base.Select(`'teacher' AS type, id, '' AS title, name, '' AS subtitle, '' AS detail_path, '' AS resource_collection_path`).
				Order("COALESCE(eval_count, 0) DESC").
				Order("created_at DESC").
				Offset(offset).Limit(size).Scan(&items).Error
			for i := range items {
				items[i].DetailPath = TeacherDetailPath(items[i].ID)
			}
			if err == nil {
				if enrichErr := r.attachSearchRelations(items); enrichErr != nil {
					return nil, 0, enrichErr
				}
			}
			return items, total, err
		}
	}

	if searchType == "" || searchType == "all" {
		sql := `
			SELECT * FROM (
				SELECT
					'resource' AS type,
					courses.id AS id,
					courses.name AS title,
					'' AS name,
					'课程资源合集' AS subtitle,
					'' AS detail_path,
					'' AS resource_collection_path,
					courses.created_at,
					COALESCE(courses.resource_count, 0) AS metric_primary,
					COALESCE(courses.download_total, 0) AS metric_secondary
				FROM courses
				WHERE courses.status = 'active' AND courses.name ILIKE ?
				UNION ALL
				SELECT
					'course' AS type,
					id,
					'' AS title,
					name,
					'' AS subtitle,
					'' AS detail_path,
					'' AS resource_collection_path,
					created_at,
					COALESCE(eval_count, 0) AS metric_primary,
					0 AS metric_secondary
				FROM courses WHERE status = 'active' AND name ILIKE ?
				UNION ALL
				SELECT
					'teacher' AS type,
					id,
					'' AS title,
					name,
					'' AS subtitle,
					'' AS detail_path,
					'' AS resource_collection_path,
					created_at,
					COALESCE(eval_count, 0) AS metric_primary,
					0 AS metric_secondary
				FROM teachers WHERE status = 'active' AND name ILIKE ?
			) AS search_results
			ORDER BY metric_primary DESC, metric_secondary DESC, created_at DESC
			LIMIT ? OFFSET ?`
		var sqlArgs []interface{}
		if relevanceFirst {
			prefixLike := trimmedQ + "%"
			sql = `
				SELECT * FROM (
					SELECT
						'resource' AS type,
						courses.id AS id,
						courses.name AS title,
						'' AS name,
						'课程资源合集' AS subtitle,
						'' AS detail_path,
						'' AS resource_collection_path,
						courses.created_at,
						COALESCE(courses.resource_count, 0) AS metric_primary,
						COALESCE(courses.download_total, 0) AS metric_secondary,
						CASE
							WHEN LOWER(courses.name) = LOWER(?) THEN 1000
							WHEN courses.name ILIKE ? THEN 700
							WHEN courses.name ILIKE ? THEN 400
							ELSE COALESCE(similarity(courses.name, ?), 0) * 100
						END AS relevance_score
					FROM courses
					WHERE courses.status = 'active' AND courses.name ILIKE ?
					UNION ALL
					SELECT
						'course' AS type,
						id,
						'' AS title,
						name,
						'' AS subtitle,
						'' AS detail_path,
						'' AS resource_collection_path,
						created_at,
						COALESCE(eval_count, 0) AS metric_primary,
						0 AS metric_secondary,
						CASE
							WHEN LOWER(name) = LOWER(?) THEN 1000
							WHEN name ILIKE ? THEN 700
							WHEN name ILIKE ? THEN 400
							ELSE COALESCE(similarity(name, ?), 0) * 100
						END AS relevance_score
					FROM courses WHERE status = 'active' AND name ILIKE ?
					UNION ALL
					SELECT
						'teacher' AS type,
						id,
						'' AS title,
						name,
						'' AS subtitle,
						'' AS detail_path,
						'' AS resource_collection_path,
						created_at,
						COALESCE(eval_count, 0) AS metric_primary,
						0 AS metric_secondary,
						CASE
							WHEN LOWER(name) = LOWER(?) THEN 1000
							WHEN name ILIKE ? THEN 700
							WHEN name ILIKE ? THEN 400
							ELSE COALESCE(similarity(name, ?), 0) * 100
						END AS relevance_score
					FROM teachers WHERE status = 'active' AND name ILIKE ?
				) AS search_results
				ORDER BY relevance_score DESC, metric_primary DESC, metric_secondary DESC, created_at DESC
				LIMIT ? OFFSET ?`
			sqlArgs = []interface{}{
				trimmedQ, prefixLike, like, trimmedQ, like,
				trimmedQ, prefixLike, like, trimmedQ, like,
				trimmedQ, prefixLike, like, trimmedQ, like,
				size, offset,
			}
		} else {
			sqlArgs = []interface{}{like, like, like, size, offset}
		}
		countSQL := `
			SELECT COUNT(*) FROM (
				SELECT courses.id
				FROM courses
				WHERE courses.status = 'active' AND courses.name ILIKE ?
				UNION ALL
				SELECT id FROM courses WHERE status = 'active' AND name ILIKE ?
				UNION ALL
				SELECT id FROM teachers WHERE status = 'active' AND name ILIKE ?
			) AS search_results`
		if err := r.db.Raw(countSQL, like, like, like).Scan(&total).Error; err != nil {
			return nil, 0, err
		}
		err := r.db.Raw(sql, sqlArgs...).Scan(&items).Error
		for i := range items {
			switch items[i].Type {
			case "resource":
				items[i].Name = items[i].Title
				items[i].ResourceCollectionPath = CourseResourceCollectionPath(items[i].ID)
				items[i].ResourceCollection = &CourseResourceCollectionCard{
					CourseID:               items[i].ID,
					CourseName:             items[i].Title,
					ResourceCollectionPath: CourseResourceCollectionPath(items[i].ID),
					CourseDetailPath:       CourseDetailPath(items[i].ID),
				}
			case "course":
				if items[i].Name == "" {
					items[i].Name = items[i].Title
				}
				items[i].DetailPath = CourseDetailPath(items[i].ID)
			case "teacher":
				if items[i].Name == "" {
					items[i].Name = items[i].Title
				}
				items[i].DetailPath = TeacherDetailPath(items[i].ID)
			}
		}
		if err == nil {
			if enrichErr := r.attachSearchRelations(items); enrichErr != nil {
				return nil, 0, enrichErr
			}
		}
		return items, total, err
	}

	switch searchType {
	case "resource":
		// 1. 资源搜索按课程资源合集维度返回，课程即使暂无资源也应命中
		base := r.db.Table("courses").
			Where("courses.status = 'active' AND courses.name ILIKE ?", like)

		if err := base.Session(&gorm.Session{}).Distinct("courses.id").Count(&total).Error; err != nil {
			return nil, 0, err
		}

		query := base.Select(`
          'resource' AS type,
          courses.id AS id,
          courses.name AS title,
          '' AS name,
          '课程资源合集' AS subtitle,
          '' AS detail_path,
          '' AS resource_collection_path`)
		if relevanceFirst {
			prefixLike := trimmedQ + "%"
			query = query.Select(`
          'resource' AS type,
          courses.id AS id,
          courses.name AS title,
          '' AS name,
          '课程资源合集' AS subtitle,
          '' AS detail_path,
          '' AS resource_collection_path,
          CASE
            WHEN LOWER(courses.name) = LOWER(?) THEN 1000
            WHEN courses.name ILIKE ? THEN 700
            WHEN courses.name ILIKE ? THEN 400
            ELSE COALESCE(similarity(courses.name, ?), 0) * 100
          END AS relevance_score
        `, trimmedQ, prefixLike, like, trimmedQ).
				Order("relevance_score DESC")
		}
		err := query.
			Order("COALESCE(courses.resource_count, 0) DESC").
			Order("COALESCE(courses.download_total, 0) DESC").
			Order("courses.created_at DESC").
			Offset(offset).Limit(size).Scan(&items).Error

		if err != nil {
			return nil, 0, err
		}

		for i := range items {
			items[i].Name = items[i].Title
			items[i].ResourceCollectionPath = CourseResourceCollectionPath(items[i].ID)
			items[i].ResourceCollection = &CourseResourceCollectionCard{
				CourseID:               items[i].ID,
				CourseName:             items[i].Title,
				ResourceCollectionPath: CourseResourceCollectionPath(items[i].ID),
				CourseDetailPath:       CourseDetailPath(items[i].ID),
			}
		}
		if enrichErr := r.attachSearchRelations(items); enrichErr != nil {
			return nil, 0, enrichErr
		}
		return items, total, nil
	case "course":
		base := r.db.Table("courses").Where("status = 'active' AND name ILIKE ?", like)
		if err := base.Count(&total).Error; err != nil {
			return nil, 0, err
		}
		query := base.Select(`'course' AS type, id, '' AS title, name, '' AS subtitle, '' AS detail_path, '' AS resource_collection_path`)
		if relevanceFirst {
			prefixLike := trimmedQ + "%"
			query = query.Select(`
        'course' AS type,
        id,
        '' AS title,
        name,
        '' AS subtitle,
        '' AS detail_path,
        '' AS resource_collection_path,
        CASE
          WHEN LOWER(name) = LOWER(?) THEN 1000
          WHEN name ILIKE ? THEN 700
          WHEN name ILIKE ? THEN 400
          ELSE COALESCE(similarity(name, ?), 0) * 100
        END AS relevance_score
      `, trimmedQ, prefixLike, like, trimmedQ).
				Order("relevance_score DESC")
		}
		err := query.
			Order("COALESCE(eval_count, 0) DESC").
			Order("created_at DESC").
			Offset(offset).
			Limit(size).
			Scan(&items).Error
		for i := range items {
			items[i].DetailPath = CourseDetailPath(items[i].ID)
		}
		if err == nil {
			if enrichErr := r.attachSearchRelations(items); enrichErr != nil {
				return nil, 0, enrichErr
			}
		}
		return items, total, err
	default:
		base := r.db.Table("teachers").Where("status = 'active' AND name ILIKE ?", like)
		if err := base.Count(&total).Error; err != nil {
			return nil, 0, err
		}
		query := base.Select(`'teacher' AS type, id, '' AS title, name, '' AS subtitle, '' AS detail_path, '' AS resource_collection_path`)
		if relevanceFirst {
			prefixLike := trimmedQ + "%"
			query = query.Select(`
        'teacher' AS type,
        id,
        '' AS title,
        name,
        '' AS subtitle,
        '' AS detail_path,
        '' AS resource_collection_path,
        CASE
          WHEN LOWER(name) = LOWER(?) THEN 1000
          WHEN name ILIKE ? THEN 700
          WHEN name ILIKE ? THEN 400
          ELSE COALESCE(similarity(name, ?), 0) * 100
        END AS relevance_score
      `, trimmedQ, prefixLike, like, trimmedQ).
				Order("relevance_score DESC")
		}
		err := query.
			Order("COALESCE(eval_count, 0) DESC").
			Order("created_at DESC").
			Offset(offset).
			Limit(size).
			Scan(&items).Error
		for i := range items {
			items[i].DetailPath = TeacherDetailPath(items[i].ID)
		}
		if err == nil {
			if enrichErr := r.attachSearchRelations(items); enrichErr != nil {
				return nil, 0, enrichErr
			}
		}
		return items, total, err
	}
}

func (r *miscRepository) ListNotifications(userID int64, isRead *bool, page, size int) ([]NotificationItem, int64, error) {
	var items []NotificationItem
	var total int64
	base := r.notificationsBaseQuery(userID)
	if isRead != nil {
		base = base.Where(fmt.Sprintf("%s = ?", notificationReadExpr("n", "gnr")), *isRead)
	}
	if err := base.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	err := base.Select(notificationSelectColumns("n", "gnr")).
		Order("n.created_at DESC").
		Order("n.id DESC").
		Offset((page - 1) * size).
		Limit(size).
		Scan(&items).Error
	return items, total, err
}

func (r *miscRepository) ListHomeNotificationSummary(userID int64) (*HomeNotificationSummary, error) {
	items, err := r.listUnreadNotifications(userID)
	if err != nil {
		return nil, err
	}

	summary := &HomeNotificationSummary{
		Announcements:  make([]NotificationItem, 0),
		Interactions:   make([]NotificationItem, 0),
		SystemMessages: make([]NotificationItem, 0),
	}
	for _, item := range items {
		switch item.Category {
		case string(model.NotificationCategoryAnnouncement):
			summary.Announcements = append(summary.Announcements, item)
		case string(model.NotificationCategoryInteraction):
			summary.Interactions = append(summary.Interactions, item)
		default:
			summary.SystemMessages = append(summary.SystemMessages, item)
		}
	}
	return summary, nil
}

func (r *miscRepository) CountUnreadNotifications(userID int64) (int64, error) {
	var count int64
	err := r.notificationsBaseQuery(userID).
		Where(fmt.Sprintf("%s = ?", notificationReadExpr("n", "gnr")), false).
		Count(&count).Error
	return count, err
}

func (r *miscRepository) MarkNotificationRead(userID, notificationID int64) error {
	cutoff := NotificationRetentionCutoff(time.Now())
	return r.db.Transaction(func(tx *gorm.DB) error {
		var item model.Notifications
		if err := tx.Select("id", "user_id", "is_global").
			Where(`
				id = ? AND created_at >= ? AND (
					(user_id = ? AND is_global = FALSE) OR (
						is_global = TRUE AND EXISTS (
							SELECT 1 FROM announcements a
							WHERE a.id = notifications.related_id AND a.deleted_at IS NULL
						)
					)
				)
			`, notificationID, cutoff, userID).
			First(&item).Error; err != nil {
			return err
		}
		if item.IsGlobal {
			return tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&model.GlobalNotificationReads{
				NotificationID: item.ID,
				UserID:         userID,
			}).Error
		}
		return tx.Model(&model.Notifications{}).
			Where("id = ? AND user_id = ? AND is_global = FALSE", notificationID, userID).
			Update("is_read", true).Error
	})
}

func (r *miscRepository) MarkAllNotificationsRead(userID int64) error {
	cutoff := NotificationRetentionCutoff(time.Now())
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&model.Notifications{}).
			Where("user_id = ? AND is_global = FALSE AND created_at >= ?", userID, cutoff).
			Update("is_read", true).Error; err != nil {
			return err
		}

		return tx.Exec(`
				INSERT INTO global_notification_reads (notification_id, user_id, created_at)
				SELECT n.id, ?, NOW()
				FROM notifications n
				JOIN announcements a ON a.id = n.related_id AND a.deleted_at IS NULL
				WHERE n.is_global = TRUE AND n.created_at >= ?
				ON CONFLICT (notification_id, user_id) DO NOTHING
			`, userID, cutoff).Error
	})
}

func (r *miscRepository) CreateNotification(notification *model.Notifications) error {
	return r.db.Create(notification).Error
}

func (r *miscRepository) PurgeExpiredNotifications(now time.Time) error {
	cutoff := NotificationRetentionCutoff(now)
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec(`
			DELETE FROM global_notification_reads AS gnr
			USING notifications AS n
			WHERE gnr.notification_id = n.id
				AND n.created_at < ?
		`, cutoff).Error; err != nil {
			return err
		}

		if err := tx.Where("created_at < ?", cutoff).
			Delete(&model.Notifications{}).Error; err != nil {
			return err
		}

		return tx.Exec(`
			DELETE FROM global_notification_reads AS gnr
			WHERE NOT EXISTS (
				SELECT 1
				FROM notifications AS n
				WHERE n.id = gnr.notification_id
			)
		`).Error
	})
}

func (r *miscRepository) listUnreadNotifications(userID int64) ([]NotificationItem, error) {
	var items []NotificationItem
	err := r.notificationsBaseQuery(userID).
		Where(fmt.Sprintf("%s = ?", notificationReadExpr("n", "gnr")), false).
		Select(notificationSelectColumns("n", "gnr")).
		Order("n.created_at DESC").
		Order("n.id DESC").
		Scan(&items).Error
	return items, err
}

func (r *miscRepository) notificationsBaseQuery(userID int64) *gorm.DB {
	cutoff := NotificationRetentionCutoff(time.Now())
	return r.db.Table("notifications AS n").
		Joins("LEFT JOIN global_notification_reads AS gnr ON gnr.notification_id = n.id AND gnr.user_id = ?", userID).
		Joins("LEFT JOIN announcements AS a ON a.id = n.related_id AND n.is_global = TRUE AND a.deleted_at IS NULL").
		Where("n.created_at >= ?", cutoff).
		Where("(n.user_id = ? AND n.is_global = FALSE) OR (n.is_global = TRUE AND a.id IS NOT NULL)", userID)
}

func notificationSelectColumns(notificationAlias, readAlias string) string {
	return fmt.Sprintf(`
		%s.id,
		%s AS type,
		%s AS category,
		%s AS result,
		%s.title,
		%s.content,
		%s.related_id,
		%s AS source_type,
		CASE
			WHEN %s.is_global = TRUE THEN %s.related_id
			ELSE COALESCE((%s.metadata->>'source_id')::bigint, %s.related_id, 0)
		END AS source_id,
		%s AS is_read,
		%s.is_global,
		COALESCE(a.is_pinned, FALSE) AS is_pinned,
		COALESCE(%s.metadata, '{}'::jsonb) AS metadata,
		%s.created_at
	`,
		notificationAlias,
		notificationTypeExpr(notificationAlias),
		notificationCategoryExpr(notificationAlias),
		notificationResultExpr(notificationAlias),
		notificationAlias,
		notificationAlias,
		notificationAlias,
		notificationSourceTypeExpr(notificationAlias),
		notificationAlias, notificationAlias,
		notificationAlias, notificationAlias,
		notificationReadExpr(notificationAlias, readAlias),
		notificationAlias,
		notificationAlias,
		notificationAlias,
	)
}

func notificationReadExpr(notificationAlias, readAlias string) string {
	return fmt.Sprintf(`
		CASE
			WHEN %s.is_global = TRUE THEN %s.notification_id IS NOT NULL
			ELSE %s.is_read
		END
	`, notificationAlias, readAlias, notificationAlias)
}

func notificationTypeExpr(notificationAlias string) string {
	return fmt.Sprintf(`
		CASE
			WHEN %s.type IN ('liked', 'commented', 'system') THEN %s.type::text
			ELSE 'system'
		END
	`, notificationAlias, notificationAlias)
}

func notificationCategoryExpr(notificationAlias string) string {
	return fmt.Sprintf(`
		COALESCE(
			NULLIF(%s.category, ''),
			CASE
				WHEN %s.is_global = TRUE THEN 'announcement'
				WHEN %s.type IN ('liked', 'commented') THEN 'interaction'
				WHEN %s.type = 'report_handled' THEN 'report'
				WHEN %s.type = 'correction_handled' THEN 'correction'
				WHEN %s.type = 'points_changed' THEN 'points'
				ELSE 'admin_message'
			END
		)
	`, notificationAlias, notificationAlias, notificationAlias, notificationAlias, notificationAlias, notificationAlias)
}

func notificationResultExpr(notificationAlias string) string {
	return fmt.Sprintf(`
		COALESCE(
			NULLIF(%s.result, ''),
			CASE
				WHEN %s.is_global = TRUE THEN 'inform'
				WHEN %s.type IN ('liked', 'commented', 'points_changed') THEN 'inform'
				WHEN %s.type = 'report_handled' AND %s.content ILIKE '%%dismissed%%' THEN 'rejected'
				WHEN %s.type = 'report_handled' THEN 'approved'
				WHEN %s.type = 'correction_handled' AND %s.content ILIKE '%%rejected%%' THEN 'rejected'
				WHEN %s.type = 'correction_handled' THEN 'approved'
				WHEN %s.title ILIKE '%%未通过%%' OR %s.content ILIKE '%%未通过%%' THEN 'rejected'
				WHEN %s.title ILIKE '%%通过%%' OR %s.content ILIKE '%%通过%%' THEN 'approved'
				ELSE 'inform'
			END
		)
	`, notificationAlias, notificationAlias, notificationAlias, notificationAlias, notificationAlias, notificationAlias, notificationAlias, notificationAlias, notificationAlias, notificationAlias, notificationAlias, notificationAlias, notificationAlias)
}

func notificationSourceTypeExpr(notificationAlias string) string {
	return fmt.Sprintf(`
		CASE
			WHEN %s.is_global = TRUE THEN 'announcement'
			ELSE COALESCE(NULLIF(%s.metadata->>'source_type', ''), '')
		END
	`, notificationAlias, notificationAlias)
}

func (r *miscRepository) attachSearchRelations(items []SearchResultItem) error {
	if len(items) == 0 {
		return nil
	}

	teacherIDs := make([]int64, 0)
	courseIDs := make([]int64, 0)
	resourceCourseIDs := make([]int64, 0)
	for _, item := range items {
		switch item.Type {
		case "teacher":
			teacherIDs = append(teacherIDs, item.ID)
		case "course":
			courseIDs = append(courseIDs, item.ID)
		case "resource":
			resourceCourseIDs = append(resourceCourseIDs, item.ID)
		}
	}

	courseDetailMap := make(map[int64]struct {
		Name          string
		CourseType    string
		AvgHomework   float64
		AvgGain       float64
		AvgExamDiff   float64
		AvgScore      float64
		EvalCount     int64
		ResourceCount int64
		FavoriteCount int64
		DownloadTotal int64
		LikeTotal     int64
	})
	allCourseIDs := append(append([]int64{}, courseIDs...), resourceCourseIDs...)
	if len(allCourseIDs) > 0 {
		type courseRow struct {
			ID            int64
			Name          string
			CourseType    string
			AvgHomework   float64
			AvgGain       float64
			AvgExamDiff   float64
			AvgScore      float64
			EvalCount     int64
			ResourceCount int64
			FavoriteCount int64
			DownloadTotal int64
			LikeTotal     int64
		}
		var rows []courseRow
		if err := r.db.Table("courses").
			Where("id IN ? AND status = 'active'", allCourseIDs).
			Select(`
				id,
				name,
				course_type,
				COALESCE(avg_workload_score, 0) AS avg_homework,
				COALESCE(avg_gain_score, 0) AS avg_gain,
				COALESCE(avg_difficulty_score, 0) AS avg_exam_diff,
					ROUND((COALESCE(avg_workload_score, 0) + COALESCE(avg_gain_score, 0) + COALESCE(avg_difficulty_score, 0)) / 3.0, 2) AS avg_score,
					COALESCE(eval_count, 0) AS eval_count,
					COALESCE(resource_count, 0) AS resource_count,
					COALESCE(favorite_count, 0) AS favorite_count,
					COALESCE(download_total, 0) AS download_total,
					COALESCE(like_total, 0) AS like_total`).
			Scan(&rows).Error; err != nil {
			return err
		}
		for _, row := range rows {
			courseDetailMap[row.ID] = struct {
				Name          string
				CourseType    string
				AvgHomework   float64
				AvgGain       float64
				AvgExamDiff   float64
				AvgScore      float64
				EvalCount     int64
				ResourceCount int64
				FavoriteCount int64
				DownloadTotal int64
				LikeTotal     int64
			}{
				Name:          row.Name,
				CourseType:    row.CourseType,
				AvgHomework:   row.AvgHomework,
				AvgGain:       row.AvgGain,
				AvgExamDiff:   row.AvgExamDiff,
				AvgScore:      row.AvgScore,
				EvalCount:     row.EvalCount,
				ResourceCount: row.ResourceCount,
				FavoriteCount: row.FavoriteCount,
				DownloadTotal: row.DownloadTotal,
				LikeTotal:     row.LikeTotal,
			}
		}
	}

	teacherDetailMap := make(map[int64]struct {
		Name           string
		DepartmentID   int64
		DepartmentName string
		Title          string
		AvatarURL      string
		AvgQuality     float64
		AvgGrading     float64
		AvgAttendance  float64
		AvgScore       float64
		GoodRate       float64
		EvalCount      int64
		FavoriteCount  int64
	})
	if len(teacherIDs) > 0 {
		type teacherRow struct {
			ID             int64
			Name           string
			DepartmentID   int64
			DepartmentName string
			Title          string
			AvatarURL      string
			AvgQuality     float64
			AvgGrading     float64
			AvgAttendance  float64
			AvgScore       float64
			GoodRate       float64
			EvalCount      int64
			FavoriteCount  int64
		}
		var rows []teacherRow
		if err := r.db.Table("teachers").
			Joins("LEFT JOIN departments ON departments.id = teachers.department_id").
			Where("teachers.id IN ? AND teachers.status = 'active'", teacherIDs).
			Select(`
				teachers.id,
				teachers.name,
				COALESCE(teachers.department_id, 0) AS department_id,
				COALESCE(departments.name, '') AS department_name,
				teachers.title,
				teachers.avatar_url,
				COALESCE(teachers.avg_teaching_score, 0) AS avg_quality,
				COALESCE(teachers.avg_grading_score, 0) AS avg_grading,
				COALESCE(teachers.avg_attendance_score, 0) AS avg_attendance,
				ROUND((COALESCE(teachers.avg_teaching_score, 0) + COALESCE(teachers.avg_grading_score, 0) + COALESCE(teachers.avg_attendance_score, 0)) / 3.0, 2) AS avg_score,
				COALESCE(teachers.approval_rate, 0) AS good_rate,
				COALESCE(teachers.eval_count, 0) AS eval_count,
				COALESCE(teachers.favorite_count, 0) AS favorite_count`).
			Scan(&rows).Error; err != nil {
			return err
		}
		for _, row := range rows {
			teacherDetailMap[row.ID] = struct {
				Name           string
				DepartmentID   int64
				DepartmentName string
				Title          string
				AvatarURL      string
				AvgQuality     float64
				AvgGrading     float64
				AvgAttendance  float64
				AvgScore       float64
				GoodRate       float64
				EvalCount      int64
				FavoriteCount  int64
			}{
				Name:           row.Name,
				DepartmentID:   row.DepartmentID,
				DepartmentName: row.DepartmentName,
				Title:          row.Title,
				AvatarURL:      row.AvatarURL,
				AvgQuality:     row.AvgQuality,
				AvgGrading:     row.AvgGrading,
				AvgAttendance:  row.AvgAttendance,
				AvgScore:       row.AvgScore,
				GoodRate:       row.GoodRate,
				EvalCount:      row.EvalCount,
				FavoriteCount:  row.FavoriteCount,
			}
		}
	}

	teacherCourses := make(map[int64][]CourseBrief)
	if len(teacherIDs) > 0 {
		type teacherCourseRow struct {
			TeacherID  int64
			CourseID   int64
			Name       string
			CourseType string
		}
		var rows []teacherCourseRow
		if err := r.db.Table("course_teachers").
			Joins("JOIN courses ON courses.id = course_teachers.course_id AND courses.status = 'active'").
			Where("course_teachers.teacher_id IN ? AND course_teachers.status = ?", teacherIDs, model.CourseTeacherRelationStatusActive).
			Select(`
				course_teachers.teacher_id,
				courses.id AS course_id,
				courses.name,
				courses.course_type`).
			Order("course_teachers.teacher_id ASC, courses.id ASC").
			Scan(&rows).Error; err != nil {
			return err
		}
		for _, row := range rows {
			teacherCourses[row.TeacherID] = append(teacherCourses[row.TeacherID], CourseBrief{
				ID:                     row.CourseID,
				Name:                   row.Name,
				CourseType:             row.CourseType,
				DetailPath:             CourseDetailPath(row.CourseID),
				ResourceCollectionPath: CourseResourceCollectionPath(row.CourseID),
			})
		}
	}

	courseTeachers := make(map[int64][]TeacherBrief)
	if len(courseIDs) > 0 {
		type courseTeacherRow struct {
			CourseID  int64
			TeacherID int64
			Name      string
			Title     string
			AvatarURL string
		}
		var rows []courseTeacherRow
		if err := r.db.Table("course_teachers").
			Joins("JOIN teachers ON teachers.id = course_teachers.teacher_id AND teachers.status = 'active'").
			Where("course_teachers.course_id IN ? AND course_teachers.status = ?", courseIDs, model.CourseTeacherRelationStatusActive).
			Select(`
				course_teachers.course_id,
				teachers.id AS teacher_id,
				teachers.name,
				teachers.title,
				teachers.avatar_url`).
			Order("course_teachers.course_id ASC, teachers.id ASC").
			Scan(&rows).Error; err != nil {
			return err
		}
		for _, row := range rows {
			courseTeachers[row.CourseID] = append(courseTeachers[row.CourseID], TeacherBrief{
				ID:         row.TeacherID,
				Name:       row.Name,
				Title:      row.Title,
				AvatarURL:  row.AvatarURL,
				DetailPath: TeacherDetailPath(row.TeacherID),
			})
		}
	}

	for i := range items {
		switch items[i].Type {
		case "teacher":
			items[i].Courses = teacherCourses[items[i].ID]
		case "course":
			items[i].Teachers = courseTeachers[items[i].ID]
		}
	}

	resourceCollectionMap := make(map[int64]*CourseResourceCollectionCard)
	if len(resourceCourseIDs) > 0 {
		type resourceCollectionRow struct {
			CourseID      int64
			CourseName    string
			ResourceCount int
			DownloadTotal int
			LikeTotal     int
			FavoriteCount int
		}
		var rows []resourceCollectionRow
		if err := r.db.Table("courses").
			Where("courses.id IN ? AND courses.status = 'active'", resourceCourseIDs).
			Select(`
				courses.id AS course_id,
				courses.name AS course_name,
				COALESCE(courses.resource_count, 0) AS resource_count,
				COALESCE(courses.download_total, 0) AS download_total,
				COALESCE(courses.like_total, 0) AS like_total,
				` + courseResourceFavoriteTotalExpr("courses") + ` AS favorite_count`).
			Scan(&rows).Error; err != nil {
			return err
		}
		for _, row := range rows {
			resourceCollectionMap[row.CourseID] = &CourseResourceCollectionCard{
				CourseID:               row.CourseID,
				CourseName:             row.CourseName,
				ResourceCount:          row.ResourceCount,
				DownloadCount:          row.DownloadTotal,
				LikeCount:              row.LikeTotal,
				FavoriteCount:          row.FavoriteCount,
				ResourceCollectionPath: CourseResourceCollectionPath(row.CourseID),
			}
		}

		type resourcePreviewRow struct {
			CourseID      int64
			ID            int64
			Title         string
			ResourceType  string
			DownloadCount int
			LikeCount     int
			CreatedAt     time.Time
		}
		var previewRows []resourcePreviewRow
		if err := r.db.Table("resources").
			Where("course_id IN ? AND status = ?", resourceCourseIDs, model.ResourceStatusApproved).
			Order("course_id ASC, created_at DESC, id DESC").
			Select(`
				course_id,
				id,
				title,
				type AS resource_type,
				download_count,
				like_count,
				created_at`).
			Scan(&previewRows).Error; err != nil {
			return err
		}
		for _, row := range previewRows {
			card, ok := resourceCollectionMap[row.CourseID]
			if !ok || len(card.ResourcesPreview) >= 3 {
				continue
			}
			card.ResourcesPreview = append(card.ResourcesPreview, ResourceCard{
				ID:            row.ID,
				Title:         row.Title,
				Type:          row.ResourceType,
				DownloadCount: row.DownloadCount,
				LikeCount:     row.LikeCount,
				CreatedAt:     row.CreatedAt,
				DetailPath:    ResourceDetailPath(row.ID),
			})
		}
	}

	for i := range items {
		switch items[i].Type {
		case "teacher":
			if detail, ok := teacherDetailMap[items[i].ID]; ok {
				items[i].Name = detail.Name
				items[i].DepartmentID = detail.DepartmentID
				items[i].DepartmentName = detail.DepartmentName
				items[i].Title = detail.Title
				items[i].AvatarURL = detail.AvatarURL
				items[i].AvgQuality = detail.AvgQuality
				items[i].AvgGrading = detail.AvgGrading
				items[i].AvgAttendance = detail.AvgAttendance
				items[i].AvgScore = detail.AvgScore
				items[i].GoodRate = detail.GoodRate
				items[i].EvalCount = detail.EvalCount
				items[i].FavoriteCount = detail.FavoriteCount
			}
		case "course":
			if detail, ok := courseDetailMap[items[i].ID]; ok {
				items[i].Name = detail.Name
				items[i].CourseType = detail.CourseType
				items[i].AvgHomework = detail.AvgHomework
				items[i].AvgGain = detail.AvgGain
				items[i].AvgExamDiff = detail.AvgExamDiff
				items[i].AvgScore = detail.AvgScore
				items[i].EvalCount = detail.EvalCount
				items[i].ResourceCount = detail.ResourceCount
				items[i].FavoriteCount = detail.FavoriteCount
				items[i].DownloadTotal = detail.DownloadTotal
				items[i].LikeTotal = detail.LikeTotal
				items[i].TeacherCount = int64(len(items[i].Teachers))
			}
		case "resource":
			if detail, ok := courseDetailMap[items[i].ID]; ok {
				items[i].CourseID = items[i].ID
				items[i].CourseName = detail.Name
				items[i].Name = detail.Name
				items[i].CourseType = detail.CourseType
				items[i].AvgHomework = detail.AvgHomework
				items[i].AvgGain = detail.AvgGain
				items[i].AvgExamDiff = detail.AvgExamDiff
				items[i].AvgScore = detail.AvgScore
				items[i].ResourceCount = detail.ResourceCount
				items[i].DownloadTotal = detail.DownloadTotal
				items[i].LikeTotal = detail.LikeTotal
			}
			if card, ok := resourceCollectionMap[items[i].ID]; ok {
				items[i].ResourceCount = int64(card.ResourceCount)
				items[i].DownloadTotal = int64(card.DownloadCount)
				items[i].LikeTotal = int64(card.LikeCount)
				items[i].FavoriteCount = int64(card.FavoriteCount)
				items[i].ResourcesPreview = card.ResourcesPreview
				items[i].ResourceCollectionPath = card.ResourceCollectionPath
				items[i].ResourceCollection = card
			}
		}
	}
	return nil
}
