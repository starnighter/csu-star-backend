package repo

import (
	"csu-star-backend/internal/model"
	"errors"
	"net"
	"time"

	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type ResourceListQuery struct {
	Q            string
	CourseID     int64
	ResourceType string
	Sort         string
	Page         int
	Size         int
}

type ResourceRankingQuery struct {
	RankType    string
	Page        int
	Size        int
	IsIncreased bool
}

type ResourceListItem struct {
	ID            int64     `json:"id"`
	Title         string    `json:"title"`
	Description   string    `json:"description"`
	UploaderID    int64     `json:"uploader_id"`
	UploaderName  string    `json:"uploader_name"`
	CourseID      int64     `json:"course_id"`
	CourseName    string    `json:"course_name"`
	Type          string    `json:"type"`
	Semester      string    `json:"semester"`
	Status        string    `json:"status"`
	DownloadCount int       `json:"download_count"`
	ViewCount     int       `json:"view_count"`
	LikeCount     int       `json:"like_count"`
	CommentCount  int       `json:"comment_count"`
	CreatedAt     time.Time `json:"created_at"`
}

type ResourceDetail struct {
	ID            int64                 `json:"id"`
	Title         string                `json:"title"`
	Description   string                `json:"description"`
	UploaderID    int64                 `json:"uploader_id"`
	UploaderName  string                `json:"uploader_name"`
	CourseID      int64                 `json:"course_id"`
	CourseName    string                `json:"course_name"`
	Type          string                `json:"type"`
	Semester      string                `json:"semester"`
	Status        string                `json:"status"`
	DownloadCount int                   `json:"download_count"`
	ViewCount     int                   `json:"view_count"`
	LikeCount     int                   `json:"like_count"`
	CommentCount  int                   `json:"comment_count"`
	ReviewReason  string                `json:"review_reason"`
	Metadata      datatypes.JSON        `json:"metadata"`
	CreatedAt     time.Time             `json:"created_at"`
	UpdatedAt     time.Time             `json:"updated_at"`
	Files         []model.ResourceFiles `json:"files"`
	IsLiked       bool                  `json:"is_liked"`
	IsFavorited   bool                  `json:"is_favorited"`
}

type MyUploadItem struct {
	ID            int64     `json:"id"`
	Title         string    `json:"title"`
	CourseID      int64     `json:"course_id"`
	CourseName    string    `json:"course_name"`
	Type          string    `json:"type"`
	Semester      string    `json:"semester"`
	Status        string    `json:"status"`
	ReviewReason  string    `json:"review_reason"`
	DownloadCount int       `json:"download_count"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

type DownloadFile struct {
	ID         int64  `json:"id"`
	ResourceID int64  `json:"resource_id"`
	FileKey    string `json:"file_key"`
	Filename   string `json:"filename"`
}

type ResourceDownloadResult struct {
	DownloadURL       string `json:"url"`
	ExpiresIn         int    `json:"expires_in"`
	RemainingPoints   int    `json:"remaining_points"`
	FreeDownloadCount int    `json:"free_download_count"`
}

type ResourceRankingItem struct {
	ID            int64   `json:"id"`
	Title         string  `json:"title"`
	CourseID      int64   `json:"course_id"`
	CourseName    string  `json:"course_name"`
	UploaderID    int64   `json:"uploader_id"`
	UploaderName  string  `json:"uploader_name"`
	Type          string  `json:"type"`
	DownloadCount int     `json:"download_count"`
	ViewCount     int     `json:"view_count"`
	LikeCount     int     `json:"like_count"`
	CommentCount  int     `json:"comment_count"`
	Score         float64 `json:"score"`
	Rank          int64   `json:"rank"`
}

var (
	ErrRepoInsufficientPoints   = errors.New("repo insufficient points")
	ErrRepoNoFreeDownloadChance = errors.New("repo no free download chance")
)

type ResourceRepository interface {
	FindResources(query ResourceListQuery) ([]ResourceListItem, int64, error)
	FindResourceDetail(id int64, includePendingForUserID int64) (*ResourceDetail, error)
	CreateResource(resource *model.Resources, files []model.ResourceFiles) error
	SubmitResource(resourceID, uploaderID int64) error
	FindDownloadFile(resourceID, fileID int64) (*DownloadFile, *model.Resources, error)
	DownloadResource(userID, resourceID int64, ip net.IP, pointsCost int, alreadyDeduped bool) (*ResourceDownloadResult, error)
	ListMyUploads(userID int64, page, size int) ([]MyUploadItem, int64, error)
	FindResourceRankings(query ResourceRankingQuery) ([]ResourceRankingItem, int64, error)
	FindResourceRankingItemsByIDs(ids []int64) ([]ResourceRankingItem, error)
	ResourceExists(id int64) (bool, error)
	RecalculateResourceCommentCount(resourceID int64) error
}

type resourceRepository struct {
	db *gorm.DB
}

func NewResourceRepository(db *gorm.DB) ResourceRepository {
	return &resourceRepository{db: db}
}

func (r *resourceRepository) FindResources(query ResourceListQuery) ([]ResourceListItem, int64, error) {
	var items []ResourceListItem
	var total int64

	base := r.db.Table("resources").
		Joins("JOIN users ON users.id = resources.uploader_id").
		Joins("JOIN courses ON courses.id = resources.course_id").
		Where("resources.status = ?", model.ResourceStatusApproved)

	if query.Q != "" {
		base = base.Where("resources.title ILIKE ?", "%"+query.Q+"%")
	}
	if query.CourseID > 0 {
		base = base.Where("resources.course_id = ?", query.CourseID)
	}
	if query.ResourceType != "" {
		base = base.Where("resources.type = ?", query.ResourceType)
	}

	if err := base.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	err := base.Select(`
		resources.id,
		resources.title,
		resources.description,
		resources.uploader_id,
		users.nickname AS uploader_name,
		resources.course_id,
		courses.name AS course_name,
		resources.type,
		resources.semester,
		resources.status,
		resources.download_count,
		resources.view_count,
		resources.like_count,
		resources.comment_count,
		resources.created_at`).
		Order(resourceSortExpr(query.Sort)).
		Order("resources.id DESC").
		Offset((query.Page - 1) * query.Size).
		Limit(query.Size).
		Scan(&items).Error
	if err != nil {
		return nil, 0, err
	}

	return items, total, nil
}

func (r *resourceRepository) FindResourceDetail(id int64, includePendingForUserID int64) (*ResourceDetail, error) {
	var detail ResourceDetail
	base := r.db.Table("resources").
		Joins("JOIN users ON users.id = resources.uploader_id").
		Joins("JOIN courses ON courses.id = resources.course_id").
		Where("resources.id = ?", id)

	if includePendingForUserID > 0 {
		base = base.Where("(resources.status = ? OR resources.uploader_id = ?)", model.ResourceStatusApproved, includePendingForUserID)
	} else {
		base = base.Where("resources.status = ?", model.ResourceStatusApproved)
	}

	err := base.Select(`
		resources.id,
		resources.title,
		resources.description,
		resources.uploader_id,
		users.nickname AS uploader_name,
		resources.course_id,
		courses.name AS course_name,
		resources.type,
		resources.semester,
		resources.status,
		resources.download_count,
		resources.view_count,
		resources.like_count,
		resources.comment_count,
		resources.review_reason,
		resources.metadata,
		resources.created_at,
		resources.updated_at,
		FALSE AS is_liked,
		FALSE AS is_favorited`).
		Scan(&detail).Error
	if err != nil {
		return nil, err
	}
	if detail.ID == 0 {
		return nil, gorm.ErrRecordNotFound
	}

	var files []model.ResourceFiles
	if err := r.db.Where("resource_id = ?", id).Order("id ASC").Find(&files).Error; err != nil {
		return nil, err
	}
	detail.Files = files

	_ = r.db.Model(&model.Resources{}).
		Where("id = ?", id).
		UpdateColumn("view_count", gorm.Expr("view_count + 1")).Error
	detail.ViewCount++

	return &detail, nil
}

func (r *resourceRepository) CreateResource(resource *model.Resources, files []model.ResourceFiles) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(resource).Error; err != nil {
			return err
		}
		for i := range files {
			files[i].ResourceID = resource.ID
			if err := tx.Create(&files[i]).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (r *resourceRepository) SubmitResource(resourceID, uploaderID int64) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		result := tx.Model(&model.Resources{}).
			Where("id = ? AND uploader_id = ? AND status = ?", resourceID, uploaderID, model.ResourceStatusDraft).
			Updates(map[string]interface{}{
				"status":     model.ResourceStatusPending,
				"updated_at": time.Now(),
			})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return gorm.ErrRecordNotFound
		}
		return nil
	})
}

func (r *resourceRepository) FindDownloadFile(resourceID, fileID int64) (*DownloadFile, *model.Resources, error) {
	var resource model.Resources
	err := r.db.Where("id = ? AND status = ?", resourceID, model.ResourceStatusApproved).First(&resource).Error
	if err != nil {
		return nil, nil, err
	}

	var file DownloadFile
	base := r.db.Table("resource_files").Where("resource_id = ?", resourceID)
	if fileID > 0 {
		base = base.Where("id = ?", fileID)
	}
	err = base.Select("id, resource_id, file_key, filename").Order("id ASC").Limit(1).Scan(&file).Error
	if err != nil {
		return nil, nil, err
	}
	if file.ID == 0 {
		return nil, nil, gorm.ErrRecordNotFound
	}

	return &file, &resource, nil
}

func (r *resourceRepository) DownloadResource(userID, resourceID int64, ip net.IP, pointsCost int, alreadyDeduped bool) (*ResourceDownloadResult, error) {
	result := &ResourceDownloadResult{}

	err := r.db.Transaction(func(tx *gorm.DB) error {
		var user model.Users
		if err := tx.Clauses().Where("id = ?", userID).First(&user).Error; err != nil {
			return err
		}

		cost := 0
		if !alreadyDeduped {
			if !user.EmailVerified {
				if user.FreeDownloadCount <= 0 {
					return ErrRepoNoFreeDownloadChance
				}
				if err := tx.Model(&model.Users{}).Where("id = ?", userID).
					Update("free_download_count", gorm.Expr("free_download_count - 1")).Error; err != nil {
					return err
				}
				user.FreeDownloadCount--
			} else {
				cost = pointsCost
				if user.Points < cost {
					return ErrRepoInsufficientPoints
				}
				if err := tx.Model(&model.Users{}).Where("id = ?", userID).
					Update("points", gorm.Expr("points - ?", cost)).Error; err != nil {
					return err
				}
				user.Points -= cost

				record := model.PointsRecords{
					UserID:    userID,
					Type:      model.PointsTypeDownload,
					Delta:     -cost,
					Balance:   user.Points,
					Reason:    "下载资源消耗积分",
					RelatedID: resourceID,
				}
				if err := tx.Create(&record).Error; err != nil {
					return err
				}

				if err := tx.Create(&model.Notifications{
					UserID:    userID,
					Type:      model.NotificationPointsChanged,
					Title:     "积分已扣减",
					Content:   "你下载资源消耗了 1 积分。",
					RelatedID: resourceID,
					IsRead:    false,
					IsGlobal:  false,
				}).Error; err != nil {
					return err
				}
			}
		}

		downloadRecord := model.DownloadRecords{
			UserID:     userID,
			ResourceID: resourceID,
			PointsCost: cost,
			IpAddress:  ip,
		}
		if err := tx.Create(&downloadRecord).Error; err != nil {
			return err
		}

		if err := tx.Model(&model.Resources{}).Where("id = ?", resourceID).
			Update("download_count", gorm.Expr("download_count + 1")).Error; err != nil {
			return err
		}

		result.RemainingPoints = user.Points
		result.FreeDownloadCount = user.FreeDownloadCount
		return nil
	})
	if err != nil {
		return nil, err
	}

	return result, nil
}

func (r *resourceRepository) ListMyUploads(userID int64, page, size int) ([]MyUploadItem, int64, error) {
	var items []MyUploadItem
	var total int64

	base := r.db.Table("resources").Where("uploader_id = ?", userID)
	if err := base.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	err := base.Joins("JOIN courses ON courses.id = resources.course_id").
		Select(`
			resources.id,
			resources.title,
			resources.course_id,
			courses.name AS course_name,
			resources.type,
			resources.semester,
			resources.status,
			resources.review_reason,
			resources.download_count,
			resources.created_at,
			resources.updated_at`).
		Order("resources.created_at DESC").
		Offset((page - 1) * size).
		Limit(size).
		Scan(&items).Error
	if err != nil {
		return nil, 0, err
	}

	return items, total, nil
}

func (r *resourceRepository) ResourceExists(id int64) (bool, error) {
	var count int64
	if err := r.db.Table("resources").Where("id = ?", id).Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r *resourceRepository) FindResourceRankings(query ResourceRankingQuery) ([]ResourceRankingItem, int64, error) {
	var items []ResourceRankingItem
	var total int64

	base := r.db.Table("resources").
		Joins("JOIN courses ON courses.id = resources.course_id").
		Joins("JOIN users ON users.id = resources.uploader_id").
		Where("resources.status = ?", model.ResourceStatusApproved)

	if err := base.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	orderDirection := "DESC"
	if query.IsIncreased {
		orderDirection = "ASC"
	}

	err := base.Select(`
		resources.id,
		resources.title,
		resources.course_id,
		courses.name AS course_name,
		resources.uploader_id,
		users.nickname AS uploader_name,
		resources.type,
		resources.download_count,
		resources.view_count,
		resources.like_count,
		resources.comment_count,
		` + resourceRankingExpr(query.RankType) + ` AS score`).
		Order(resourceRankingExpr(query.RankType) + " " + orderDirection).
		Order("resources.id DESC").
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

func (r *resourceRepository) RecalculateResourceCommentCount(resourceID int64) error {
	return r.db.Model(&model.Resources{}).
		Where("id = ?", resourceID).
		Update("comment_count", gorm.Expr(`(
			SELECT COUNT(*)
			FROM comments
			WHERE comments.target_type = ? AND comments.target_id = ? AND comments.status = ?
		)`, model.CommentTargetTypeResource, resourceID, model.CommentStatusActive)).Error
}

func (r *resourceRepository) FindResourceRankingItemsByIDs(ids []int64) ([]ResourceRankingItem, error) {
	if len(ids) == 0 {
		return []ResourceRankingItem{}, nil
	}

	var items []ResourceRankingItem
	err := r.db.Table("resources").
		Joins("JOIN courses ON courses.id = resources.course_id").
		Joins("JOIN users ON users.id = resources.uploader_id").
		Where("resources.status = ?", model.ResourceStatusApproved).
		Where("resources.id IN ?", ids).
		Select(`
			resources.id,
			resources.title,
			resources.course_id,
			courses.name AS course_name,
			resources.uploader_id,
			users.nickname AS uploader_name,
			resources.type,
			resources.download_count,
			resources.view_count,
			resources.like_count,
			resources.comment_count`).
		Scan(&items).Error
	if err != nil {
		return nil, err
	}
	return items, nil
}

func resourceSortExpr(sort string) string {
	switch sort {
	case "hot_score":
		return "(resources.download_count + resources.like_count + resources.comment_count) DESC"
	case "downloads":
		return "resources.download_count DESC"
	default:
		return "resources.created_at DESC"
	}
}

func resourceRankingExpr(rankType string) string {
	switch rankType {
	case "downloads":
		return "COALESCE(resources.download_count, 0)"
	case "likes":
		return "COALESCE(resources.like_count, 0)"
	case "comments":
		return "COALESCE(resources.comment_count, 0)"
	case "views":
		return "COALESCE(resources.view_count, 0)"
	default:
		return "COALESCE(resources.download_count + resources.like_count + resources.comment_count, 0)"
	}
}
