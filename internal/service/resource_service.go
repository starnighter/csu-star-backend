package service

import (
	"csu-star-backend/config"
	"csu-star-backend/internal/constant"
	"csu-star-backend/internal/model"
	"csu-star-backend/internal/repo"
	"csu-star-backend/internal/task"
	"csu-star-backend/logger"
	"csu-star-backend/pkg/utils"
	"encoding/json"
	"errors"
	"net"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

var (
	ErrResourceNotFound               = errors.New("resource not found")
	ErrResourceForbidden              = errors.New("resource forbidden")
	ErrInsufficientPoints             = errors.New("insufficient points")
	ErrNoFreeDownloadChance           = errors.New("no free download chance")
	ErrInvalidResourceFile            = errors.New("invalid resource file")
	ErrResourceCourseNotFound         = errors.New("resource course not found")
	ErrResourceAlreadyDeleted         = errors.New("resource already deleted")
	ErrResourceUploadTooLarge         = errors.New("resource upload too large")
	ErrResourceUploadSessionNotFound  = errors.New("resource upload session not found")
	ErrResourceUploadSessionForbidden = errors.New("resource upload session forbidden")
	ErrResourceUploadIncomplete       = errors.New("resource upload incomplete")
	ErrResourceRateLimited            = errors.New("resource rate limited")
	ErrResourceUserBanned             = errors.New("resource user banned")
	ErrResourceEmailNotVerified       = errors.New("resource email not verified")
)

type ResourceService struct {
	db           *gorm.DB
	resourceRepo repo.ResourceRepository
	courseRepo   repo.CourseRepository
	socialRepo   repo.SocialRepository
	securitySvc  *SecurityService
}

type ResourceUploadURLItem struct {
	FileID string `json:"file_id"`
	URL    string `json:"url"`
	Method string `json:"method"`
}

type ResourceUploadResponse struct {
	UploadSessionID string                  `json:"upload_session_id"`
	UploadURLs      []ResourceUploadURLItem `json:"upload_urls"`
}

type ResourceFinalizeResponse struct {
	ResourceID int64 `json:"resource_id,string"`
}

const (
	maxResourceUploadSizeBytes = int64(300 * 1024 * 1024)
	resourceUploadSessionTTL   = 2 * time.Hour
)

type ResourceUploadSession struct {
	ID           string                 `json:"id"`
	UserID       int64                  `json:"user_id"`
	Title        string                 `json:"title"`
	Description  string                 `json:"description"`
	CourseID     int64                  `json:"course_id"`
	ResourceType string                 `json:"resource_type"`
	Files        []UploadedResourceFile `json:"files"`
}

func NewResourceService(db *gorm.DB, rr repo.ResourceRepository, cr repo.CourseRepository, sr repo.SocialRepository) *ResourceService {
	return &ResourceService{db: db, resourceRepo: rr, courseRepo: cr, socialRepo: sr}
}

func (s *ResourceService) SetSecurityService(securitySvc *SecurityService) {
	s.securitySvc = securitySvc
}

func (s *ResourceService) resourceRateLimitEnabled() bool {
	if config.GlobalConfig == nil {
		return true
	}
	return config.GlobalConfig.Security.ResourceRateLimitEnabled
}

func (s *ResourceService) ListResources(query repo.ResourceListQuery) ([]repo.ResourceListItem, int64, error) {
	fillPagination(&query.Page, &query.Size)
	if query.Sort == "" {
		query.Sort = "created_at"
	}
	return s.resourceRepo.FindResources(query)
}

func (s *ResourceService) GetResourceDetail(resourceID, userID int64, userRole string) (*repo.ResourceDetail, error) {
	detail, err := s.resourceRepo.FindResourceDetail(resourceID, userID, userRole)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrResourceNotFound
	}
	if err != nil {
		return nil, err
	}
	if detail.Status == string(model.ResourceStatusApproved) {
		if err := s.withWriteTx(func(resourceRepo repo.ResourceRepository, courseRepo repo.CourseRepository) error {
			if err := resourceRepo.IncrementResourceViewCount(resourceID); err != nil {
				return err
			}
			return courseRepo.AdjustCourseAggregates(detail.CourseID, 0, 0, 1, 0, 0, 0, 0)
		}); err != nil {
			return nil, err
		}
		detail.ViewCount++
	}
	if userID > 0 && s.socialRepo != nil {
		likedMap, err := s.socialRepo.ListLikedTargetIDs(userID, model.LikeTargetTypeResource, []int64{detail.ID})
		if err != nil {
			return nil, err
		}
		favoritedMap, err := s.socialRepo.ListFavoritedTargetIDs(userID, model.FavoriteTargetTypeResource, []int64{detail.ID})
		if err != nil {
			return nil, err
		}
		detail.IsLiked = likedMap[detail.ID]
		detail.IsFavorited = favoritedMap[detail.ID]
	}
	return detail, nil
}

func (s *ResourceService) CreateResource(userID int64, title, description string, courseID int64, resourceType string, files []UploadedResourceFile) (*model.Resources, []model.ResourceFiles, error) {
	exists, err := s.courseRepo.CourseExists(courseID)
	if err != nil {
		return nil, nil, err
	}
	if !exists {
		return nil, nil, ErrResourceCourseNotFound
	}
	if len(files) == 0 {
		return nil, nil, ErrInvalidResourceFile
	}

	resource := &model.Resources{
		Title:       title,
		Description: description,
		UploaderID:  userID,
		CourseID:    courseID,
		Type:        model.ResourceType(resourceType),
		Status:      model.ResourceStatusDraft,
		Metadata:    datatypes.JSON([]byte(`{}`)),
	}

	resourceFiles := make([]model.ResourceFiles, 0, len(files))
	for _, file := range files {
		resourceFiles = append(resourceFiles, model.ResourceFiles{
			Filename: file.Filename,
			FileKey:  file.FileKey,
			FileUrl:  file.FileURL,
			FileSize: file.FileSize,
			FileHash: file.FileHash,
			MimeType: file.MimeType,
		})
	}

	if err := s.resourceRepo.CreateResource(resource, resourceFiles); err != nil {
		return nil, nil, err
	}
	return resource, resourceFiles, nil
}

func (s *ResourceService) PrepareResourceUpload(userID int64, title, description string, courseID int64, resourceType string, files []UploadedResourceFile) (*ResourceUploadResponse, error) {
	if err := s.enforceUploadRateLimit(userID, files); err != nil {
		return nil, err
	}

	if s.db != nil {
		var user model.Users
		if err := s.db.Select("email_verified").Where("id = ?", userID).First(&user).Error; err != nil {
			return nil, err
		}
		if !user.EmailVerified {
			return nil, ErrResourceEmailNotVerified
		}
	}

	exists, err := s.courseRepo.CourseExists(courseID)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, ErrResourceCourseNotFound
	}
	if len(files) == 0 {
		return nil, ErrInvalidResourceFile
	}
	totalSize := int64(0)
	uploadURLs := make([]ResourceUploadURLItem, 0, len(files))
	for _, file := range files {
		totalSize += file.FileSize
		if totalSize > maxResourceUploadSizeBytes {
			return nil, ErrResourceUploadTooLarge
		}

		uploadURL, err := utils.TencentCosUploadTemporarily(file.FileKey)
		if err != nil {
			return nil, err
		}
		uploadURLs = append(uploadURLs, ResourceUploadURLItem{
			FileID: file.FileID,
			URL:    uploadURL,
			Method: "PUT",
		})
	}

	session := ResourceUploadSession{
		ID:           uuid.NewString(),
		UserID:       userID,
		Title:        title,
		Description:  description,
		CourseID:     courseID,
		ResourceType: resourceType,
		Files:        files,
	}
	if err := s.saveUploadSession(session); err != nil {
		return nil, err
	}

	return &ResourceUploadResponse{
		UploadSessionID: session.ID,
		UploadURLs:      uploadURLs,
	}, nil
}

func (s *ResourceService) FinalizeResourceUpload(userID int64, sessionID string) (*ResourceFinalizeResponse, error) {
	if err := s.enforceSingleWindowLimit(userID, "resource_finalize", 20, time.Hour); err != nil {
		return nil, err
	}

	session, err := s.loadUploadSession(sessionID)
	if err != nil {
		return nil, err
	}
	if session.UserID != userID {
		if s.securitySvc != nil {
			_, _, _ = s.securitySvc.CountAbuseAndMaybeBan(
				utils.Ctx,
				utils.RDB,
				userID,
				"resource_finalize",
				BuildTriggerKey("resource_finalize", "foreign_session"),
				"频繁尝试操作他人的上传会话",
				3,
				6*time.Hour,
				BuildViolationEvidence("upload_session_id", sessionID),
			)
		}
		return nil, ErrResourceUploadSessionForbidden
	}

	for _, file := range session.Files {
		exists, err := utils.TencentCosObjectExists(file.FileKey)
		if err != nil {
			return nil, err
		}
		if !exists {
			_ = s.cleanupUploadSessionObjects(session)
			_ = s.deleteUploadSession(session.ID)
			if s.securitySvc != nil {
				_, _, _ = s.securitySvc.CountAbuseAndMaybeBan(
					utils.Ctx,
					utils.RDB,
					userID,
					"resource_finalize",
					BuildTriggerKey("resource_finalize", "incomplete"),
					"频繁提交未完成的上传资源",
					3,
					6*time.Hour,
					BuildViolationEvidence("upload_session_id", session.ID, "file_key", file.FileKey),
				)
			}
			return nil, ErrResourceUploadIncomplete
		}
	}

	resource := &model.Resources{
		Title:       session.Title,
		Description: session.Description,
		UploaderID:  session.UserID,
		CourseID:    session.CourseID,
		Type:        model.ResourceType(session.ResourceType),
		Status:      model.ResourceStatusApproved,
		Metadata:    datatypes.JSON([]byte(`{}`)),
	}

	resourceFiles := make([]model.ResourceFiles, 0, len(session.Files))
	for _, file := range session.Files {
		resourceFiles = append(resourceFiles, model.ResourceFiles{
			Filename: file.Filename,
			FileKey:  file.FileKey,
			FileUrl:  file.FileURL,
			FileSize: file.FileSize,
			FileHash: file.FileHash,
			MimeType: file.MimeType,
		})
	}

	if err := s.db.Transaction(func(tx *gorm.DB) error {
		resourceRepo := s.resourceRepoWithTx(tx)
		courseRepo := s.courseRepoWithTx(tx)
		if err := resourceRepo.CreateResource(resource, resourceFiles); err != nil {
			return err
		}
		if err := courseRepo.AdjustCourseAggregates(session.CourseID, 1, 0, 0, 0, 0, 0, 0); err != nil {
			return err
		}
		return rewardUserPointsTx(
			tx,
			userID,
			resource.ID,
			resourceUploadRewardPoints,
			model.PointsTypeUpload,
			"上传资源获得积分",
			"上传成功",
			"你上传资源获得了 2 积分。",
		)
	}); err != nil {
		_ = s.cleanupUploadSessionObjects(session)
		_ = s.deleteUploadSession(session.ID)
		return nil, err
	}

	if err := s.deleteUploadSession(session.ID); err != nil {
		logger.Log.Warn("删除资源上传会话失败", zap.String("session_id", session.ID), zap.Error(err))
	}

	return &ResourceFinalizeResponse{
		ResourceID: resource.ID,
	}, nil
}

func (s *ResourceService) AbortResourceUpload(userID int64, sessionID string) error {
	if err := s.enforceSingleWindowLimit(userID, "resource_abort", 20, time.Hour); err != nil {
		return err
	}
	session, err := s.loadUploadSession(sessionID)
	if err != nil {
		return err
	}
	if session.UserID != userID {
		return ErrResourceUploadSessionForbidden
	}
	if err := s.cleanupUploadSessionObjects(session); err != nil {
		return err
	}
	return s.deleteUploadSession(sessionID)
}

func (s *ResourceService) DownloadResource(userID, resourceID, fileID int64, ip net.IP) (*repo.ResourceDownloadResult, error) {
	if err := s.enforceSingleWindowLimit(userID, "resource_download", 60, time.Hour); err != nil {
		return nil, err
	}
	if ip != nil && s.securitySvc != nil {
		key := BuildRateLimitKey("resource_download", "ip", ip.String())
		decision, err := s.securitySvc.RateLimit(utils.Ctx, utils.RDB, key, 120, time.Hour)
		if err != nil {
			return nil, err
		}
		if !decision.Allowed {
			return nil, ErrResourceRateLimited
		}
	}

	file, resource, err := s.resourceRepo.FindDownloadFile(resourceID, fileID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrResourceNotFound
	}
	if err != nil {
		return nil, err
	}

	var result *repo.ResourceDownloadResult
	err = s.withWriteTx(func(resourceRepo repo.ResourceRepository, courseRepo repo.CourseRepository) error {
		var txErr error
		result, txErr = resourceRepo.DownloadResource(userID, resourceID, ip, 1)
		if txErr != nil {
			if errors.Is(txErr, repo.ErrRepoInsufficientPoints) {
				return ErrInsufficientPoints
			}
			if errors.Is(txErr, repo.ErrRepoNoFreeDownloadChance) {
				return ErrNoFreeDownloadChance
			}
			return txErr
		}
		return courseRepo.AdjustCourseAggregates(resource.CourseID, 0, 1, 0, 0, 0, 0, 0)
	})
	if err != nil {
		return nil, err
	}

	url, err := utils.TencentCosDownloadTemporarily(file.FileKey, file.Filename)
	if err != nil {
		return nil, err
	}
	result.DownloadURL = url
	result.ExpiresIn = 7200
	return result, nil
}

func (s *ResourceService) ListMyUploads(userID int64, page, size int) ([]repo.MyUploadItem, int64, error) {
	fillPagination(&page, &size)
	return s.resourceRepo.ListMyUploads(userID, page, size)
}

func (s *ResourceService) ListMyResources(userID int64, page, size int) ([]repo.MyUploadItem, int64, error) {
	fillPagination(&page, &size)
	return s.resourceRepo.ListMyResources(repo.MyResourcesQuery{
		Page: page,
		Size: size,
	}, userID)
}

func (s *ResourceService) DeleteResource(userID int64, userRole string, resourceID int64) error {
	payload, err := s.resourceRepo.GetResourceDeletePayload(resourceID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return ErrResourceNotFound
	}
	if err != nil {
		return err
	}
	if payload.Status == model.ResourceStatusDeleted {
		return ErrResourceAlreadyDeleted
	}
	if payload.UploaderID != userID && !isPrivilegedRole(userRole) {
		return ErrResourceForbidden
	}

	if err := s.withWriteTx(func(resourceRepo repo.ResourceRepository, courseRepo repo.CourseRepository) error {
		if err := resourceRepo.SoftDeleteResource(resourceID); err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrResourceAlreadyDeleted
			}
			return err
		}
		return courseRepo.RecalculateCourseStats(payload.CourseID)
	}); err != nil {
		return err
	}

	for _, fileKey := range payload.FileKeys {
		if err := utils.TencentCosDeleteObject(fileKey); err != nil {
			logger.Log.Error("资源文件物理删除失败，资源记录已软删除，等待后续补偿清理",
				zap.Int64("resource_id", resourceID),
				zap.String("file_key", fileKey),
				zap.Error(err))
			break
		}
	}

	return nil
}

func (s *ResourceService) UpdateResource(userID int64, userRole string, resourceID int64, title, description string, courseID int64, resourceType string) (*repo.ResourceDetail, error) {
	resource, err := s.resourceRepo.GetResourceByID(resourceID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrResourceNotFound
	}
	if err != nil {
		return nil, err
	}
	if resource.Status == model.ResourceStatusDeleted {
		return nil, ErrResourceAlreadyDeleted
	}
	if resource.UploaderID != userID && !isPrivilegedRole(userRole) {
		return nil, ErrResourceForbidden
	}
	exists, err := s.courseRepo.CourseExists(courseID)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, ErrResourceCourseNotFound
	}
	oldCourseID := resource.CourseID
	if err := s.withWriteTx(func(resourceRepo repo.ResourceRepository, courseRepo repo.CourseRepository) error {
		if err := resourceRepo.UpdateResource(resourceID, title, description, courseID, model.ResourceType(resourceType)); err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrResourceNotFound
			}
			return err
		}
		if oldCourseID == courseID {
			return nil
		}
		if err := courseRepo.RecalculateCourseStats(oldCourseID); err != nil {
			return err
		}
		return courseRepo.RecalculateCourseStats(courseID)
	}); err != nil {
		return nil, err
	}
	return s.GetResourceDetail(resourceID, userID, userRole)
}

func (s *ResourceService) ListResourceRankings(query repo.ResourceRankingQuery) ([]repo.ResourceRankingItem, int64, error) {
	fillPagination(&query.Page, &query.Size)
	switch query.RankType {
	case "", "comprehensive":
		query.RankType = "comprehensive"
	case "downloads", "views", "likes", "favorite_count", "resource_count":
	default:
		query.RankType = "comprehensive"
	}

	cacheKey := task.ResourceRankingCacheKey(query.RankType)
	ids, scores, total, err := task.ReadRankingIDs(cacheKey, query.Page, query.Size, query.IsIncreased)
	if err == nil && total > 0 {
		items, err := s.resourceRepo.FindResourceRankingItemsByIDs(ids)
		if err == nil {
			itemMap := make(map[int64]repo.ResourceRankingItem, len(items))
			for _, item := range items {
				itemMap[item.CourseID] = item
			}
			ordered := make([]repo.ResourceRankingItem, 0, len(ids))
			startRank := int64((query.Page-1)*query.Size + 1)
			for i, id := range ids {
				item, ok := itemMap[id]
				if !ok {
					continue
				}
				item.Score = scores[i]
				item.Rank = startRank + int64(i)
				ordered = append(ordered, item)
			}
			return ordered, total, nil
		}
	}

	return s.resourceRepo.FindResourceRankings(query)
}

type UploadedResourceFile struct {
	FileID   string
	Filename string
	FileKey  string
	FileURL  string
	FileSize int64
	FileHash string
	MimeType string
}

func NewUploadedResourceFile(filename string, sizeBytes int64, mimeType string) UploadedResourceFile {
	fileID := uuid.NewString()
	fileKey := constant.TencentCosPendingResourcesKeyPrefix + fileID + filepath.Ext(filename)
	return UploadedResourceFile{
		FileID:   fileID,
		Filename: filename,
		FileKey:  fileKey,
		FileURL:  utils.TencentCosObjectURL(fileKey),
		FileSize: sizeBytes,
		MimeType: mimeType,
	}
}

func resourceUploadSessionKey(sessionID string) string {
	return constant.ResourceUploadSessionPrefix + sessionID
}

func (s *ResourceService) saveUploadSession(session ResourceUploadSession) error {
	payload, err := json.Marshal(session)
	if err != nil {
		return err
	}
	return utils.RDB.Set(
		utils.Ctx,
		resourceUploadSessionKey(session.ID),
		payload,
		resourceUploadSessionTTL,
	).Err()
}

func (s *ResourceService) loadUploadSession(sessionID string) (*ResourceUploadSession, error) {
	raw, err := utils.RDB.Get(utils.Ctx, resourceUploadSessionKey(sessionID)).Result()
	if err != nil {
		return nil, ErrResourceUploadSessionNotFound
	}
	var session ResourceUploadSession
	if err := json.Unmarshal([]byte(raw), &session); err != nil {
		return nil, err
	}
	return &session, nil
}

func (s *ResourceService) deleteUploadSession(sessionID string) error {
	return utils.RDB.Del(utils.Ctx, resourceUploadSessionKey(sessionID)).Err()
}

func (s *ResourceService) cleanupUploadSessionObjects(session *ResourceUploadSession) error {
	for _, file := range session.Files {
		if err := utils.TencentCosDeleteObject(file.FileKey); err != nil {
			return err
		}
	}
	return nil
}

func (s *ResourceService) enforceSingleWindowLimit(userID int64, scope string, limit int64, window time.Duration) error {
	if !s.resourceRateLimitEnabled() {
		return nil
	}
	if s.securitySvc == nil || userID <= 0 {
		return nil
	}
	key := BuildRateLimitKey(scope, "user", strconv.FormatInt(userID, 10))
	decision, err := s.securitySvc.RateLimit(utils.Ctx, utils.RDB, key, limit, window)
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil
		}
		return err
	}
	if decision.Allowed {
		return nil
	}
	return ErrResourceRateLimited
}

func (s *ResourceService) enforceUploadRateLimit(userID int64, files []UploadedResourceFile) error {
	if !s.resourceRateLimitEnabled() {
		return nil
	}
	if s.securitySvc == nil || userID <= 0 {
		return nil
	}

	if _, banDecision, err := s.securitySvc.EnforceUserAccess(userID); err != nil {
		if errors.Is(err, ErrSecurityUserBanned) {
			_ = banDecision
			return ErrResourceUserBanned
		}
		return err
	}

	totalSize := int64(0)
	for _, file := range files {
		totalSize += file.FileSize
	}

	perHourKey := BuildRateLimitKey("resource_prepare", "user", strconv.FormatInt(userID, 10))
	decision, err := s.securitySvc.RateLimit(utils.Ctx, utils.RDB, perHourKey, 20, time.Hour)
	if err != nil {
		return err
	}
	if !decision.Allowed {
		_, _, _ = s.securitySvc.CountAbuseAndMaybeBan(
			utils.Ctx,
			utils.RDB,
			userID,
			"resource_prepare",
			BuildTriggerKey("resource_prepare", "too_many_requests"),
			"短时间内频繁创建上传会话",
			3,
			6*time.Hour,
			BuildViolationEvidence("files", len(files), "total_size", totalSize),
		)
		return ErrResourceRateLimited
	}

	dayBytesKey := BuildRateLimitKey("resource_upload_bytes", "user", strconv.FormatInt(userID, 10))
	byteDecision, err := s.securitySvc.RateLimit(utils.Ctx, utils.RDB, dayBytesKey, 1024*1024*1024, 24*time.Hour)
	if err != nil {
		return err
	}
	if totalSize > 0 {
		current := byteDecision.Current - 1 + totalSize
		if err := utils.RDB.Set(utils.Ctx, dayBytesKey, current, 24*time.Hour).Err(); err != nil {
			return err
		}
		if current > 1024*1024*1024 {
			_, _, _ = s.securitySvc.CountAbuseAndMaybeBan(
				utils.Ctx,
				utils.RDB,
				userID,
				"resource_prepare",
				BuildTriggerKey("resource_prepare", "daily_bytes"),
				"单日上传流量异常",
				2,
				24*time.Hour,
				BuildViolationEvidence("total_size", totalSize, "daily_total", current),
			)
			return ErrResourceRateLimited
		}
	}

	fileCountKey := BuildRateLimitKey("resource_upload_files", "user", strconv.FormatInt(userID, 10))
	countDecision, err := s.securitySvc.RateLimit(utils.Ctx, utils.RDB, fileCountKey, 100, 24*time.Hour)
	if err != nil {
		return err
	}
	currentCount := countDecision.Current - 1 + int64(len(files))
	if err := utils.RDB.Set(utils.Ctx, fileCountKey, currentCount, 24*time.Hour).Err(); err != nil {
		return err
	}
	if currentCount > 100 {
		_, _, _ = s.securitySvc.CountAbuseAndMaybeBan(
			utils.Ctx,
			utils.RDB,
			userID,
			"resource_prepare",
			BuildTriggerKey("resource_prepare", "daily_files"),
			"单日上传文件数异常",
			2,
			24*time.Hour,
			BuildViolationEvidence("files", len(files), "daily_total", currentCount),
		)
		return ErrResourceRateLimited
	}

	return nil
}

func ParseClientIP(rawIP string) net.IP {
	if rawIP == "" {
		return nil
	}
	if strings.Contains(rawIP, ":") {
		if host, _, err := net.SplitHostPort(rawIP); err == nil {
			rawIP = host
		}
	}
	return net.ParseIP(rawIP)
}

func (s *ResourceService) withWriteTx(fn func(repo.ResourceRepository, repo.CourseRepository) error) error {
	if s.db == nil {
		return fn(s.resourceRepo, s.courseRepo)
	}
	return s.db.Transaction(func(tx *gorm.DB) error {
		return fn(s.resourceRepoWithTx(tx), s.courseRepoWithTx(tx))
	})
}

func (s *ResourceService) resourceRepoWithTx(tx *gorm.DB) repo.ResourceRepository {
	withTx, ok := s.resourceRepo.(interface {
		WithTx(*gorm.DB) repo.ResourceRepository
	})
	if !ok {
		return s.resourceRepo
	}
	return withTx.WithTx(tx)
}

func (s *ResourceService) courseRepoWithTx(tx *gorm.DB) repo.CourseRepository {
	withTx, ok := s.courseRepo.(interface {
		WithTx(*gorm.DB) repo.CourseRepository
	})
	if !ok {
		return s.courseRepo
	}
	return withTx.WithTx(tx)
}
