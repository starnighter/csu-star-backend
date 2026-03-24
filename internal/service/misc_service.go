package service

import (
	"csu-star-backend/internal/model"
	"csu-star-backend/internal/repo"
	"csu-star-backend/pkg/utils"
	"encoding/json"
	"errors"
)

var ErrMeNotFound = errors.New("me not found")
var ErrAlreadyCheckedIn = errors.New("already checked in")

type MiscService struct {
	miscRepo   repo.MiscRepository
	socialRepo repo.SocialRepository
}

func NewMiscService(mr repo.MiscRepository, sr repo.SocialRepository) *MiscService {
	return &MiscService{miscRepo: mr, socialRepo: sr}
}

func (s *MiscService) GetMe(userID int64) (*repo.MeProfile, error) {
	item, err := s.miscRepo.GetMe(userID)
	if err != nil {
		return nil, err
	}
	return item, nil
}

func (s *MiscService) UpdateMe(userID int64, nickname, avatarURL string) error {
	return s.miscRepo.UpdateMe(userID, nickname, avatarURL)
}

func (s *MiscService) DailyCheckin(userID int64) (int, error) {
	balance, err := s.miscRepo.DailyCheckin(userID)
	if errors.Is(err, repo.ErrAlreadyCheckedIn) {
		return balance, ErrAlreadyCheckedIn
	}
	return balance, err
}

func (s *MiscService) ListMyDownloads(userID int64, page, size int) ([]repo.DownloadHistoryItem, int64, error) {
	fillPagination(&page, &size)
	return s.miscRepo.ListMyDownloads(userID, page, size)
}

func (s *MiscService) ListMyFavorites(userID int64, targetType string, page, size int) ([]repo.FavoriteItem, int64, error) {
	fillPagination(&page, &size)
	return s.miscRepo.ListMyFavorites(userID, targetType, page, size)
}

func (s *MiscService) ListMyPoints(userID int64, page, size int) ([]repo.PointRecordItem, int64, error) {
	fillPagination(&page, &size)
	return s.miscRepo.ListMyPoints(userID, page, size)
}

func (s *MiscService) ListAnnouncements() ([]repo.AnnouncementItem, error) {
	return s.miscRepo.ListAnnouncements()
}

func (s *MiscService) CreateFeedback(userID int64, feedbackType, title, content string, files []string) error {
	raw, _ := json.Marshal(files)
	return s.miscRepo.CreateFeedback(&model.Feedbacks{
		UserID:      userID,
		Type:        model.FeedbackType(feedbackType),
		Title:       title,
		Content:     content,
		Attachments: raw,
		Status:      model.FeedbackStatusPending,
	})
}

func (s *MiscService) CreateReport(userID int64, targetType string, targetID int64, reason, description string) error {
	ok, err := s.reportTargetExists(model.ReportTargetType(targetType), targetID)
	if err != nil {
		return err
	}
	if !ok {
		return ErrSocialTargetNotFound
	}
	return s.miscRepo.CreateReport(&model.Reports{
		UserID:      userID,
		TargetType:  model.ReportTargetType(targetType),
		TargetID:    targetID,
		Reason:      reason,
		Description: description,
		Status:      model.ReportStatusPending,
	})
}

func (s *MiscService) CreateCorrection(userID int64, targetType string, targetID int64, field, suggestedValue string) error {
	ok, err := s.correctionTargetExists(model.CorrectionTargetType(targetType), targetID)
	if err != nil {
		return err
	}
	if !ok {
		return ErrSocialTargetNotFound
	}
	return s.miscRepo.CreateCorrection(&model.Corrections{
		UserID:         userID,
		TargetType:     model.CorrectionTargetType(targetType),
		TargetID:       targetID,
		Field:          field,
		SuggestedValue: suggestedValue,
		Status:         model.CorrectionStatusPending,
	})
}

func (s *MiscService) Search(userID int64, q, searchType string, page, size int) ([]repo.SearchResultItem, int64, error) {
	fillPagination(&page, &size)
	if searchType == "" {
		searchType = "all"
	}
	items, total, err := s.miscRepo.Search(q, searchType, page, size)
	if err != nil {
		return nil, 0, err
	}
	if userID > 0 {
		_ = s.miscRepo.UpsertSearchHistory(userID, q)
		for _, period := range []string{"day", "week", "month"} {
			_ = utils.RDB.ZIncrBy(utils.Ctx, "search:hot:"+period, 1, q).Err()
		}
	}
	return items, total, nil
}

func (s *MiscService) ListSearchHistory(userID int64) ([]model.SearchHistories, error) {
	return s.miscRepo.ListSearchHistory(userID)
}

func (s *MiscService) ClearSearchHistory(userID int64) error {
	return s.miscRepo.ClearSearchHistory(userID)
}

func (s *MiscService) ListHotKeywords(period string) ([]model.HotKeywords, error) {
	if period == "" {
		period = "day"
	}
	return s.miscRepo.ListHotKeywords(period)
}

func (s *MiscService) ListNotifications(userID int64, isRead *bool, page, size int) ([]repo.NotificationItem, int64, error) {
	fillPagination(&page, &size)
	return s.miscRepo.ListNotifications(userID, isRead, page, size)
}

func (s *MiscService) CountUnreadNotifications(userID int64) (int64, error) {
	return s.miscRepo.CountUnreadNotifications(userID)
}

func (s *MiscService) MarkNotificationRead(userID, notificationID int64) error {
	return s.miscRepo.MarkNotificationRead(userID, notificationID)
}

func (s *MiscService) MarkAllNotificationsRead(userID int64) error {
	return s.miscRepo.MarkAllNotificationsRead(userID)
}

func (s *MiscService) reportTargetExists(targetType model.ReportTargetType, targetID int64) (bool, error) {
	switch targetType {
	case model.ReportTargetTypeResource:
		return s.socialRepo.ResourceExists(targetID)
	case model.ReportTargetTypeTeacherEvaluation:
		return s.socialRepo.TeacherEvaluationExists(targetID)
	case model.ReportTargetTypeCourseEvaluation:
		return s.socialRepo.CourseEvaluationExists(targetID)
	case model.ReportTargetTypeComment:
		return s.socialRepo.CommentExists(targetID)
	default:
		return false, nil
	}
}

func (s *MiscService) correctionTargetExists(targetType model.CorrectionTargetType, targetID int64) (bool, error) {
	switch targetType {
	case model.CorrectionTargetTypeTeacher:
		return s.socialRepo.TeacherExists(targetID)
	case model.CorrectionTargetTypeCourse:
		return s.socialRepo.CourseExists(targetID)
	default:
		return false, nil
	}
}
