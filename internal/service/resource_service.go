package service

import (
	"csu-star-backend/internal/constant"
	"csu-star-backend/internal/model"
	"csu-star-backend/internal/repo"
	"csu-star-backend/internal/task"
	"csu-star-backend/pkg/utils"
	"errors"
	"mime/multipart"
	"net"
	"strconv"
	"strings"
	"time"

	"gorm.io/datatypes"
	"gorm.io/gorm"
)

const defaultResourceDownloadCost = 1

var (
	ErrResourceNotFound       = errors.New("resource not found")
	ErrResourceForbidden      = errors.New("resource forbidden")
	ErrInsufficientPoints     = errors.New("insufficient points")
	ErrNoFreeDownloadChance   = errors.New("no free download chance")
	ErrInvalidResourceFile    = errors.New("invalid resource file")
	ErrResourceCourseNotFound = errors.New("resource course not found")
)

type ResourceService struct {
	resourceRepo repo.ResourceRepository
	courseRepo   repo.CourseRepository
	socialRepo   repo.SocialRepository
}

func NewResourceService(rr repo.ResourceRepository, cr repo.CourseRepository, sr repo.SocialRepository) *ResourceService {
	return &ResourceService{resourceRepo: rr, courseRepo: cr, socialRepo: sr}
}

func (s *ResourceService) ListResources(query repo.ResourceListQuery) ([]repo.ResourceListItem, int64, error) {
	fillPagination(&query.Page, &query.Size)
	if query.Sort == "" {
		query.Sort = "created_at"
	}
	return s.resourceRepo.FindResources(query)
}

func (s *ResourceService) GetResourceDetail(resourceID, userID int64) (*repo.ResourceDetail, error) {
	detail, err := s.resourceRepo.FindResourceDetail(resourceID, userID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrResourceNotFound
	}
	if err != nil {
		return nil, err
	}
	if userID > 0 && s.socialRepo != nil {
		liked, err := s.socialRepo.HasLike(userID, model.LikeTargetTypeResource, detail.ID)
		if err != nil {
			return nil, err
		}
		favorited, err := s.socialRepo.HasFavorite(userID, model.FavoriteTargetTypeResource, detail.ID)
		if err != nil {
			return nil, err
		}
		detail.IsLiked = liked
		detail.IsFavorited = favorited
	}
	return detail, nil
}

func (s *ResourceService) CreateResource(userID int64, title, description string, courseID int64, resourceType, semester string, files []UploadedResourceFile) (*model.Resources, []model.ResourceFiles, error) {
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
		Semester:    semester,
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

func (s *ResourceService) SubmitResource(userID, resourceID int64) error {
	err := s.resourceRepo.SubmitResource(resourceID, userID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return ErrResourceNotFound
	}
	return err
}

func (s *ResourceService) DownloadResource(userID, resourceID, fileID int64, ip net.IP) (*repo.ResourceDownloadResult, error) {
	file, _, err := s.resourceRepo.FindDownloadFile(resourceID, fileID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrResourceNotFound
	}
	if err != nil {
		return nil, err
	}

	dedupKey := "download:dedup:" + strconv.FormatInt(userID, 10) + ":" + strconv.FormatInt(resourceID, 10)
	alreadyDeduped := false
	if v, err := utils.RDB.Exists(utils.Ctx, dedupKey).Result(); err == nil && v > 0 {
		alreadyDeduped = true
	}

	result, err := s.resourceRepo.DownloadResource(userID, resourceID, ip, defaultResourceDownloadCost, alreadyDeduped)
	if err != nil {
		if errors.Is(err, repo.ErrRepoInsufficientPoints) {
			return nil, ErrInsufficientPoints
		}
		if errors.Is(err, repo.ErrRepoNoFreeDownloadChance) {
			return nil, ErrNoFreeDownloadChance
		}
		return nil, err
	}

	url, err := utils.TencentCosDownloadTemporarily(file.FileKey)
	if err != nil {
		return nil, err
	}

	if !alreadyDeduped {
		_ = utils.RDB.Set(utils.Ctx, dedupKey, time.Now().Unix(), 7*24*time.Hour).Err()
	}

	result.DownloadURL = url
	result.ExpiresIn = 7200
	return result, nil
}

func (s *ResourceService) ListMyUploads(userID int64, page, size int) ([]repo.MyUploadItem, int64, error) {
	fillPagination(&page, &size)
	return s.resourceRepo.ListMyUploads(userID, page, size)
}

func (s *ResourceService) ListResourceRankings(query repo.ResourceRankingQuery) ([]repo.ResourceRankingItem, int64, error) {
	fillPagination(&query.Page, &query.Size)
	if query.RankType == "" {
		query.RankType = "hot_score"
	}

	cacheKey := task.ResourceRankingCacheKey(query.RankType)
	ids, scores, total, err := task.ReadRankingIDs(cacheKey, query.Page, query.Size, query.IsIncreased)
	if err == nil && total > 0 {
		items, err := s.resourceRepo.FindResourceRankingItemsByIDs(ids)
		if err == nil {
			itemMap := make(map[int64]repo.ResourceRankingItem, len(items))
			for _, item := range items {
				itemMap[item.ID] = item
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
	Filename string
	FileKey  string
	FileURL  string
	FileSize int64
	FileHash string
	MimeType string
}

func UploadResourceFiles(fileHeaders []*multipart.FileHeader) ([]UploadedResourceFile, error) {
	files := make([]UploadedResourceFile, 0, len(fileHeaders))
	for _, fileHeader := range fileHeaders {
		info, err := utils.TencentCosUpload(fileHeader, constant.TencentCosResourcesKeyPrefix)
		if err != nil {
			return nil, err
		}
		files = append(files, UploadedResourceFile{
			Filename: info.Filename,
			FileKey:  info.FileKey,
			FileURL:  info.FileUrl,
			FileSize: info.FileSize,
			FileHash: info.FileHash,
			MimeType: info.MimeType,
		})
	}
	return files, nil
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
