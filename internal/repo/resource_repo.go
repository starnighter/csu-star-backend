package repo

import (
	"csu-star-backend/internal/model"
	"errors"
	"net"
	"strconv"
	"time"

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

type MyResourcesQuery struct {
	Page int
	Size int
}

type ResourceRankingQuery struct {
	RankType    string
	Page        int
	Size        int
	IsIncreased bool
}

type ResourceListItem struct {
	ID            int64     `json:"id,string"`
	Title         string    `json:"title"`
	Description   string    `json:"description"`
	UploaderID    int64     `json:"uploader_id"`
	UploaderName  string    `json:"uploader_name"`
	CourseID      int64     `json:"course_id,string"`
	CourseName    string    `json:"course_name"`
	Type          string    `json:"resource_type"`
	DownloadCount int       `json:"download_count"`
	ViewCount     int       `json:"view_count"`
	LikeCount     int       `json:"like_count"`
	CommentCount  int       `json:"comment_count"`
	CreatedAt     time.Time `json:"created_at"`
}

type ResourceDetail struct {
	ID                           int64              `json:"id,string"`
	Title                        string             `json:"title"`
	Description                  string             `json:"description"`
	UploaderID                   int64              `json:"uploader_id,string"`
	UploaderName                 string             `json:"-"`
	CourseID                     int64              `json:"course_id,string"`
	CourseName                   string             `json:"-"`
	Course                       CourseBrief        `json:"course" gorm:"-"`
	Type                         string             `json:"resource_type"`
	DownloadCount                int                `json:"downloads"`
	ViewCount                    int                `json:"views"`
	LikeCount                    int                `json:"likes"`
	FavoriteCount                int                `json:"favorite_count"`
	CommentCount                 int                `json:"comment_count"`
	HotScore                     int                `json:"hot_score"`
	Status                       string             `json:"status"`
	CreatedAt                    time.Time          `json:"created_at"`
	UpdatedAt                    time.Time          `json:"-"`
	CourseResourceCollectionPath string             `json:"course_resource_collection_path,omitempty"`
	CourseEvaluationAnchor       string             `json:"course_evaluation_anchor,omitempty"`
	Files                        []ResourceFileItem `json:"files" gorm:"-"`
	Tags                         []string           `json:"tags" gorm:"-"`
	IsLiked                      bool               `json:"is_liked"`
	IsFavorited                  bool               `json:"is_favorited"`
}

type MyUploadItem struct {
	ID            int64       `json:"id,string"`
	Title         string      `json:"title"`
	UploaderID    int64       `json:"uploader_id,string"`
	CourseID      int64       `json:"course_id,string"`
	Course        CourseBrief `json:"course"`
	CourseCode    string      `json:"-"`
	CourseName    string      `json:"-"`
	Type          string      `json:"resource_type"`
	DownloadCount int         `json:"downloads"`
	ViewCount     int         `json:"views"`
	LikeCount     int         `json:"likes"`
	HotScore      int         `json:"hot_score"`
	Status        string      `json:"status"`
	CreatedAt     time.Time   `json:"created_at"`
	UpdatedAt     time.Time   `json:"-"`
}

type ResourceFileItem struct {
	ID        string `json:"id"`
	Filename  string `json:"filename"`
	Mime      string `json:"mime,omitempty"`
	SizeBytes int64  `json:"size_bytes"`
	SortOrder int    `json:"sort_order"`
}

type DownloadFile struct {
	ID         int64  `json:"id,string"`
	ResourceID int64  `json:"resource_id,string"`
	FileKey    string `json:"file_key"`
	Filename   string `json:"filename"`
}

type ResourceDeletePayload struct {
	ResourceID    int64
	CourseID      int64
	UploaderID    int64
	Status        model.ResourceStatus
	DownloadCount int
	ViewCount     int
	LikeCount     int
	CommentCount  int
	FileKeys      []string
}

type ResourceDownloadResult struct {
	DownloadURL       string `json:"url"`
	ExpiresIn         int    `json:"expires_in"`
	RemainingPoints   int    `json:"remaining_points"`
	FreeDownloadCount int    `json:"free_download_count"`
}

type ResourceRankingItem struct {
	ID                     int64   `json:"id,string"`
	CourseID               int64   `json:"course_id,string"`
	CourseName             string  `json:"course_name"`
	CourseType             string  `json:"course_type"`
	ResourceCount          int     `json:"resource_count"`
	DownloadCount          int     `json:"download_total"`
	ViewTotal              int     `json:"view_total"`
	LikeCount              int     `json:"like_total"`
	FavoriteCount          int     `json:"favorite_count"`
	Score                  float64 `json:"score"`
	Rank                   int64   `json:"rank"`
	ResourceCollectionPath string  `json:"detail_path,omitempty"`
}

var (
	ErrRepoInsufficientPoints   = errors.New("repo insufficient points")
	ErrRepoNoFreeDownloadChance = errors.New("repo no free download chance")
)

type ResourceRepository interface {
	FindResources(query ResourceListQuery) ([]ResourceListItem, int64, error)
	FindResourceDetail(id, viewerUserID int64, viewerRole string) (*ResourceDetail, error)
	CreateResource(resource *model.Resources, files []model.ResourceFiles) error
	GetResourceCourseID(resourceID int64) (int64, error)
	GetResourceDeletePayload(resourceID int64) (*ResourceDeletePayload, error)
	GetResourceByID(resourceID int64) (*model.Resources, error)
	IncrementResourceViewCount(resourceID int64) error
	UpdateResource(resourceID int64, title, description string, courseID int64, resourceType model.ResourceType) error
	SoftDeleteResource(resourceID int64) error
	FindDownloadFile(resourceID, fileID int64) (*DownloadFile, *model.Resources, error)
	DownloadResource(userID, resourceID int64, ip net.IP, pointsCost int) (*ResourceDownloadResult, error)
	ListMyUploads(userID int64, page, size int) ([]MyUploadItem, int64, error)
	ListMyResources(query MyResourcesQuery, userID int64) ([]MyUploadItem, int64, error)
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

func (r *resourceRepository) WithTx(tx *gorm.DB) ResourceRepository {
	return &resourceRepository{db: tx}
}

func (r *resourceRepository) FindResources(query ResourceListQuery) ([]ResourceListItem, int64, error) {
	var items []ResourceListItem
	var total int64

	base := r.db.Table("resources").
		Joins("JOIN users ON users.id = resources.uploader_id").
		Joins("JOIN courses ON courses.id = resources.course_id AND courses.status = ?", model.CourseStatusActive).
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

func (r *resourceRepository) FindResourceDetail(id, viewerUserID int64, viewerRole string) (*ResourceDetail, error) {
	var detail ResourceDetail
	base := r.db.Table("resources").
		Joins("JOIN users ON users.id = resources.uploader_id").
		Joins("JOIN courses ON courses.id = resources.course_id AND courses.status = ?", model.CourseStatusActive).
		Where("resources.id = ?", id)

	if viewerUserID > 0 {
		if viewerRole == string(model.UserRoleAdmin) || viewerRole == string(model.UserRoleAuditor) {
			base = base.Where("resources.status IN ?", []model.ResourceStatus{
				model.ResourceStatusApproved,
				model.ResourceStatusPending,
				model.ResourceStatusDeleted,
			})
		} else {
			base = base.Where(`
				resources.status = ?
				OR (resources.uploader_id = ? AND resources.status IN ?)
			`, model.ResourceStatusApproved, viewerUserID, []model.ResourceStatus{
				model.ResourceStatusPending,
				model.ResourceStatusDeleted,
			})
		}
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
		resources.download_count,
		resources.view_count,
		resources.like_count,
		COALESCE((
			SELECT COUNT(*)
			FROM favorites
			WHERE favorites.target_type = 'resource' AND favorites.target_id = resources.id
		), 0) AS favorite_count,
		resources.comment_count,
		(resources.download_count + resources.like_count + resources.comment_count) AS hot_score,
		resources.status,
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
	detail.Files = make([]ResourceFileItem, 0, len(files))
	for i, file := range files {
		detail.Files = append(detail.Files, ResourceFileItem{
			ID:        strconv.FormatInt(file.ID, 10),
			Filename:  file.Filename,
			Mime:      file.MimeType,
			SizeBytes: file.FileSize,
			SortOrder: i,
		})
	}
	var tags []string
	if err := r.db.Table("resource_tags").
		Joins("JOIN tags ON tags.id = resource_tags.tag_id").
		Where("resource_tags.resource_id = ?", id).
		Order("tags.name ASC").
		Pluck("tags.name", &tags).Error; err != nil {
		return nil, err
	}
	detail.Tags = tags
	detail.Course = CourseBrief{
		ID:                     detail.CourseID,
		Name:                   detail.CourseName,
		DetailPath:             CourseDetailPath(detail.CourseID),
		ResourceCollectionPath: CourseResourceCollectionPath(detail.CourseID),
	}
	detail.CourseResourceCollectionPath = CourseResourceCollectionPath(detail.CourseID)
	detail.CourseEvaluationAnchor = CourseEvaluationAnchorPath(detail.CourseID)

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

func (r *resourceRepository) GetResourceCourseID(resourceID int64) (int64, error) {
	type result struct {
		CourseID int64
	}

	var item result
	if err := r.db.Table("resources").
		Select("course_id").
		Where("id = ?", resourceID).
		Scan(&item).Error; err != nil {
		return 0, err
	}
	if item.CourseID == 0 {
		return 0, gorm.ErrRecordNotFound
	}
	return item.CourseID, nil
}

func (r *resourceRepository) GetResourceDeletePayload(resourceID int64) (*ResourceDeletePayload, error) {
	type resourceRow struct {
		ID            int64
		CourseID      int64
		UploaderID    int64
		Status        model.ResourceStatus
		DownloadCount int
		ViewCount     int
		LikeCount     int
		CommentCount  int
	}

	var resource resourceRow
	if err := r.db.Table("resources").
		Select("id, course_id, uploader_id, status, download_count, view_count, like_count, comment_count").
		Where("id = ?", resourceID).
		Scan(&resource).Error; err != nil {
		return nil, err
	}
	if resource.ID == 0 {
		return nil, gorm.ErrRecordNotFound
	}

	var fileKeys []string
	if err := r.db.Table("resource_files").
		Where("resource_id = ?", resourceID).
		Pluck("file_key", &fileKeys).Error; err != nil {
		return nil, err
	}

	return &ResourceDeletePayload{
		ResourceID:    resource.ID,
		CourseID:      resource.CourseID,
		UploaderID:    resource.UploaderID,
		Status:        resource.Status,
		DownloadCount: resource.DownloadCount,
		ViewCount:     resource.ViewCount,
		LikeCount:     resource.LikeCount,
		CommentCount:  resource.CommentCount,
		FileKeys:      fileKeys,
	}, nil
}

func (r *resourceRepository) GetResourceByID(resourceID int64) (*model.Resources, error) {
	var resource model.Resources
	if err := r.db.First(&resource, resourceID).Error; err != nil {
		return nil, err
	}
	return &resource, nil
}

func (r *resourceRepository) IncrementResourceViewCount(resourceID int64) error {
	return r.db.Model(&model.Resources{}).
		Where("id = ? AND status = ?", resourceID, model.ResourceStatusApproved).
		UpdateColumn("view_count", gorm.Expr("view_count + 1")).Error
}

func (r *resourceRepository) UpdateResource(resourceID int64, title, description string, courseID int64, resourceType model.ResourceType) error {
	result := r.db.Model(&model.Resources{}).
		Where("id = ? AND status <> ?", resourceID, model.ResourceStatusDeleted).
		Updates(map[string]interface{}{
			"title":       title,
			"description": description,
			"course_id":   courseID,
			"type":        resourceType,
			"updated_at":  time.Now(),
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func (r *resourceRepository) SoftDeleteResource(resourceID int64) error {
	result := r.db.Model(&model.Resources{}).
		Where("id = ? AND status <> ?", resourceID, model.ResourceStatusDeleted).
		Updates(map[string]interface{}{
			"status":     model.ResourceStatusDeleted,
			"updated_at": time.Now(),
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func (r *resourceRepository) FindDownloadFile(resourceID, fileID int64) (*DownloadFile, *model.Resources, error) {
	var resource model.Resources
	err := r.db.Table("resources").
		Joins("JOIN courses ON courses.id = resources.course_id AND courses.status = ?", model.CourseStatusActive).
		Where("resources.id = ? AND resources.status = ?", resourceID, model.ResourceStatusApproved).
		First(&resource).Error
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

func (r *resourceRepository) DownloadResource(userID, resourceID int64, ip net.IP, pointsCost int) (*ResourceDownloadResult, error) {
	result := &ResourceDownloadResult{}

	err := r.db.Transaction(func(tx *gorm.DB) error {
		var user model.Users
		if err := tx.Clauses().Where("id = ?", userID).First(&user).Error; err != nil {
			return err
		}

		cost := 0
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
				Category:  model.NotificationCategoryPoints,
				Result:    model.NotificationResultInform,
				Title:     "积分已扣减",
				Content:   "你下载资源消耗了 1 积分。",
				RelatedID: resourceID,
				IsRead:    true,
				IsGlobal:  false,
			}).Error; err != nil {
				return err
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
	return r.ListMyResources(MyResourcesQuery{Page: page, Size: size}, userID)
}

func (r *resourceRepository) ListMyResources(query MyResourcesQuery, userID int64) ([]MyUploadItem, int64, error) {
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
			resources.uploader_id,
			resources.course_id,
			courses.name AS course_name,
			resources.type,
			resources.download_count,
			resources.view_count,
			resources.like_count,
			(resources.download_count + resources.like_count + resources.comment_count) AS hot_score,
			resources.status,
			resources.created_at,
			resources.updated_at`).
		Order("resources.created_at DESC").
		Offset((query.Page - 1) * query.Size).
		Limit(query.Size).
		Scan(&items).Error
	if err != nil {
		return nil, 0, err
	}
	for i := range items {
		items[i].Course = CourseBrief{
			ID:                     items[i].CourseID,
			Code:                   items[i].CourseCode,
			Name:                   items[i].CourseName,
			DetailPath:             CourseDetailPath(items[i].CourseID),
			ResourceCollectionPath: CourseResourceCollectionPath(items[i].CourseID),
		}
	}

	return items, total, nil
}

func (r *resourceRepository) ResourceExists(id int64) (bool, error) {
	var count int64
	if err := r.db.Table("resources").
		Joins("JOIN courses ON courses.id = resources.course_id AND courses.status = ?", model.CourseStatusActive).
		Where("resources.id = ? AND resources.status = ?", id, model.ResourceStatusApproved).Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r *resourceRepository) FindResourceRankings(query ResourceRankingQuery) ([]ResourceRankingItem, int64, error) {
	var items []ResourceRankingItem
	var total int64

	base := r.db.Table("courses").Where("courses.status = ?", model.CourseStatusActive)
	if query.IsIncreased {
		base = base.Where(resourceRankingExpr(query.RankType) + " > 0")
	}

	if err := base.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	orderDirection := "DESC"
	if query.IsIncreased {
		orderDirection = "ASC"
	}

	err := base.Select(`
			courses.id,
			courses.id AS course_id,
			courses.name AS course_name,
			courses.course_type,
			COALESCE(courses.resource_count, 0) AS resource_count,
			COALESCE(courses.download_total, 0) AS download_count,
			COALESCE(courses.view_total, 0) AS view_total,
			COALESCE(courses.like_total, 0) AS like_count,
			` + courseResourceFavoriteTotalExpr("courses") + ` AS favorite_count,
			` + resourceRankingExpr(query.RankType) + ` AS score`).
		Order(resourceRankingExpr(query.RankType) + " " + orderDirection).
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
		items[i].ResourceCollectionPath = CourseResourceCollectionPath(items[i].CourseID)
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
	err := r.db.Table("courses").
		Where("courses.id IN ? AND courses.status = ?", ids, model.CourseStatusActive).
		Select(`
			courses.id,
			courses.id AS course_id,
			courses.name AS course_name,
			courses.course_type,
			COALESCE(courses.resource_count, 0) AS resource_count,
			COALESCE(courses.download_total, 0) AS download_count,
			COALESCE(courses.view_total, 0) AS view_total,
			COALESCE(courses.like_total, 0) AS like_count,
			` + courseResourceFavoriteTotalExpr("courses") + ` AS favorite_count`).
		Scan(&items).Error
	if err != nil {
		return nil, err
	}
	for i := range items {
		items[i].ResourceCollectionPath = CourseResourceCollectionPath(items[i].CourseID)
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
		return "COALESCE(courses.download_total, 0)"
	case "likes":
		return "COALESCE(courses.like_total, 0)"
	case "views":
		return "COALESCE(courses.view_total, 0)"
	case "resource_count":
		return "COALESCE(courses.resource_count, 0)"
	case "favorite_count":
		return courseResourceFavoriteTotalExpr("courses")
	case "comprehensive":
		return `(
			COALESCE(courses.resource_count, 0) * 12 +
			COALESCE(courses.download_total, 0) * 8 +
			` + courseResourceFavoriteTotalExpr("courses") + ` * 5 +
			COALESCE(courses.like_total, 0)
		)`
	default:
		return `(
			COALESCE(courses.resource_count, 0) * 12 +
			COALESCE(courses.download_total, 0) * 8 +
			` + courseResourceFavoriteTotalExpr("courses") + ` * 5 +
			COALESCE(courses.like_total, 0)
		)`
	}
}
