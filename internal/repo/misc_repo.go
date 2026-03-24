package repo

import (
	"csu-star-backend/internal/model"
	"errors"
	"time"

	"gorm.io/gorm"
)

var ErrAlreadyCheckedIn = errors.New("already checked in")

type MeProfile struct {
	ID                int64     `json:"id"`
	Email             string    `json:"email"`
	EmailVerified     bool      `json:"email_verified"`
	Nickname          string    `json:"nickname"`
	AvatarURL         string    `json:"avatar_url"`
	Role              string    `json:"role"`
	Points            int       `json:"points"`
	FreeDownloadCount int       `json:"free_download_count"`
	CreatedAt         time.Time `json:"created_at"`
}

type DownloadHistoryItem struct {
	ID         int64     `json:"id"`
	ResourceID int64     `json:"resource_id"`
	Title      string    `json:"title"`
	CourseID   int64     `json:"course_id"`
	CourseName string    `json:"course_name"`
	PointsCost int       `json:"points_cost"`
	CreatedAt  time.Time `json:"created_at"`
}

type FavoriteItem struct {
	ID         int64     `json:"id"`
	TargetType string    `json:"target_type"`
	TargetID   int64     `json:"target_id"`
	Name       string    `json:"name"`
	CreatedAt  time.Time `json:"created_at"`
}

type PointRecordItem struct {
	ID        int64     `json:"id"`
	Type      string    `json:"type"`
	Delta     int       `json:"delta"`
	Balance   int       `json:"balance"`
	Reason    string    `json:"reason"`
	RelatedID int64     `json:"related_id"`
	CreatedAt time.Time `json:"created_at"`
}

type AnnouncementItem struct {
	ID          int64     `json:"id"`
	Title       string    `json:"title"`
	Content     string    `json:"content"`
	Type        string    `json:"type"`
	IsPinned    bool      `json:"is_pinned"`
	PublishedAt time.Time `json:"published_at"`
}

type SearchResultItem struct {
	Type string `json:"type"`
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

type NotificationItem struct {
	ID        int64     `json:"id"`
	Type      string    `json:"type"`
	Title     string    `json:"title"`
	Content   string    `json:"content"`
	RelatedID int64     `json:"related_id"`
	IsRead    bool      `json:"is_read"`
	IsGlobal  bool      `json:"is_global"`
	CreatedAt time.Time `json:"created_at"`
}

type MiscRepository interface {
	GetMe(userID int64) (*MeProfile, error)
	UpdateMe(userID int64, nickname, avatarURL string) error
	DailyCheckin(userID int64) (int, error)
	ListMyDownloads(userID int64, page, size int) ([]DownloadHistoryItem, int64, error)
	ListMyFavorites(userID int64, targetType string, page, size int) ([]FavoriteItem, int64, error)
	ListMyPoints(userID int64, page, size int) ([]PointRecordItem, int64, error)
	ListAnnouncements() ([]AnnouncementItem, error)
	CreateFeedback(feedback *model.Feedbacks) error
	CreateReport(report *model.Reports) error
	CreateCorrection(correction *model.Corrections) error
	Search(q, searchType string, page, size int) ([]SearchResultItem, int64, error)
	UpsertSearchHistory(userID int64, keyword string) error
	ListSearchHistory(userID int64) ([]model.SearchHistories, error)
	ClearSearchHistory(userID int64) error
	ListHotKeywords(period string) ([]model.HotKeywords, error)
	ListNotifications(userID int64, isRead *bool, page, size int) ([]NotificationItem, int64, error)
	CountUnreadNotifications(userID int64) (int64, error)
	MarkNotificationRead(userID, notificationID int64) error
	MarkAllNotificationsRead(userID int64) error
	CreateNotification(notification *model.Notifications) error
}

type miscRepository struct {
	db *gorm.DB
}

func NewMiscRepository(db *gorm.DB) MiscRepository {
	return &miscRepository{db: db}
}

func (r *miscRepository) GetMe(userID int64) (*MeProfile, error) {
	var item MeProfile
	err := r.db.Table("users").Select("id, email, email_verified, nickname, avatar_url, role, points, free_download_count, created_at").Where("id = ?", userID).Scan(&item).Error
	if err != nil {
		return nil, err
	}
	if item.ID == 0 {
		return nil, gorm.ErrRecordNotFound
	}
	return &item, nil
}

func (r *miscRepository) UpdateMe(userID int64, nickname, avatarURL string) error {
	updates := map[string]interface{}{}
	if nickname != "" {
		updates["nickname"] = nickname
	}
	if avatarURL != "" {
		updates["avatar_url"] = avatarURL
	}
	if len(updates) == 0 {
		return nil
	}
	return r.db.Model(&model.Users{}).Where("id = ?", userID).Updates(updates).Error
}

func (r *miscRepository) DailyCheckin(userID int64) (int, error) {
	var balance int
	err := r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec("SELECT pg_advisory_xact_lock(?)", userID).Error; err != nil {
			return err
		}

		var count int64
		if err := tx.Table("points_records").
			Where("user_id = ? AND type = ? AND created_at >= CURRENT_DATE", userID, model.PointsTypeCheckin).
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

		return tx.Create(&model.Notifications{
			UserID:    userID,
			Type:      model.NotificationPointsChanged,
			Title:     "签到成功",
			Content:   "你已完成今日签到，获得 1 积分。",
			RelatedID: 0,
			IsRead:    false,
			IsGlobal:  false,
		}).Error
	})
	return balance, err
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
		Select("download_records.id, download_records.resource_id, resources.title, resources.course_id, courses.name AS course_name, download_records.points_cost, download_records.created_at").
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

func (r *miscRepository) ListAnnouncements() ([]AnnouncementItem, error) {
	var items []AnnouncementItem
	err := r.db.Table("announcements").
		Where("is_published = ? AND (expires_at IS NULL OR expires_at > ?)", true, time.Now()).
		Select("id, title, content, type, is_pinned, published_at").
		Order("is_pinned DESC").Order("published_at DESC").Scan(&items).Error
	return items, err
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

func (r *miscRepository) Search(q, searchType string, page, size int) ([]SearchResultItem, int64, error) {
	var items []SearchResultItem
	var total int64
	like := "%" + q + "%"
	offset := (page - 1) * size

	if searchType == "" || searchType == "all" {
		sql := `
			SELECT * FROM (
				SELECT 'resource' AS type, id, title AS name, created_at FROM resources WHERE status = 'approved' AND title ILIKE ?
				UNION ALL
				SELECT 'course' AS type, id, name, created_at FROM courses WHERE name ILIKE ?
				UNION ALL
				SELECT 'teacher' AS type, id, name, created_at FROM teachers WHERE name ILIKE ?
			) AS search_results
			ORDER BY created_at DESC
			LIMIT ? OFFSET ?`
		countSQL := `
			SELECT COUNT(*) FROM (
				SELECT id FROM resources WHERE status = 'approved' AND title ILIKE ?
				UNION ALL
				SELECT id FROM courses WHERE name ILIKE ?
				UNION ALL
				SELECT id FROM teachers WHERE name ILIKE ?
			) AS search_results`
		if err := r.db.Raw(countSQL, like, like, like).Scan(&total).Error; err != nil {
			return nil, 0, err
		}
		err := r.db.Raw(sql, like, like, like, size, offset).Scan(&items).Error
		return items, total, err
	}

	table := map[string]string{"resource": "resources", "course": "courses", "teacher": "teachers"}[searchType]
	nameField := map[string]string{"resource": "title", "course": "name", "teacher": "name"}[searchType]
	base := r.db.Table(table)
	if searchType == "resource" {
		base = base.Where("status = ?", model.ResourceStatusApproved)
	}
	if err := base.Where(nameField+" ILIKE ?", like).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	err := base.Select("? AS type, id, "+nameField+" AS name", searchType).Where(nameField+" ILIKE ?", like).
		Order("created_at DESC").Offset(offset).Limit(size).Scan(&items).Error
	return items, total, err
}

func (r *miscRepository) UpsertSearchHistory(userID int64, keyword string) error {
	return r.db.Exec(`
		INSERT INTO search_histories (user_id, keyword, created_at)
		VALUES (?, ?, CURRENT_TIMESTAMP)
		ON CONFLICT (user_id, keyword)
		DO UPDATE SET created_at = CURRENT_TIMESTAMP
	`, userID, keyword).Error
}

func (r *miscRepository) ListSearchHistory(userID int64) ([]model.SearchHistories, error) {
	var items []model.SearchHistories
	err := r.db.Where("user_id = ?", userID).Order("created_at DESC").Limit(20).Find(&items).Error
	return items, err
}

func (r *miscRepository) ClearSearchHistory(userID int64) error {
	return r.db.Where("user_id = ?", userID).Delete(&model.SearchHistories{}).Error
}

func (r *miscRepository) ListHotKeywords(period string) ([]model.HotKeywords, error) {
	var items []model.HotKeywords
	err := r.db.Where("period = ?", period).Order("count DESC").Limit(20).Find(&items).Error
	return items, err
}

func (r *miscRepository) ListNotifications(userID int64, isRead *bool, page, size int) ([]NotificationItem, int64, error) {
	var items []NotificationItem
	var total int64
	base := r.db.Table("notifications").Where("user_id = ? OR is_global = ?", userID, true)
	if isRead != nil {
		base = base.Where("is_read = ?", *isRead)
	}
	if err := base.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	err := base.Select("id, type, title, content, related_id, is_read, is_global, created_at").Order("created_at DESC").Offset((page - 1) * size).Limit(size).Scan(&items).Error
	return items, total, err
}

func (r *miscRepository) CountUnreadNotifications(userID int64) (int64, error) {
	var count int64
	err := r.db.Table("notifications").Where("(user_id = ? OR is_global = ?) AND is_read = ?", userID, true, false).Count(&count).Error
	return count, err
}

func (r *miscRepository) MarkNotificationRead(userID, notificationID int64) error {
	return r.db.Model(&model.Notifications{}).Where("id = ? AND (user_id = ? OR is_global = ?)", notificationID, userID, true).Update("is_read", true).Error
}

func (r *miscRepository) MarkAllNotificationsRead(userID int64) error {
	return r.db.Model(&model.Notifications{}).Where("user_id = ? OR is_global = ?", userID, true).Update("is_read", true).Error
}

func (r *miscRepository) CreateNotification(notification *model.Notifications) error {
	return r.db.Create(notification).Error
}
