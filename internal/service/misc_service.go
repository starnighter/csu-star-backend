package service

import (
	"csu-star-backend/internal/model"
	"csu-star-backend/internal/repo"
	"encoding/json"
	"errors"
	"time"

	"gorm.io/gorm"
)

var ErrMeNotFound = errors.New("me not found")
var ErrAlreadyCheckedIn = errors.New("already checked in")

const contributionWeeks = 26

var contributionLocation = loadContributionLocation()

type MiscService struct {
	db             *gorm.DB
	miscRepo       repo.MiscRepository
	socialRepo     repo.SocialRepository
	invitationRepo repo.InvitationRepository
}

func NewMiscService(db *gorm.DB, mr repo.MiscRepository, sr repo.SocialRepository, ir repo.InvitationRepository) *MiscService {
	return &MiscService{db: db, miscRepo: mr, socialRepo: sr, invitationRepo: ir}
}

func (s *MiscService) GetMe(userID int64) (*repo.MeProfile, error) {
	item, err := s.miscRepo.GetMe(userID)
	if err != nil {
		return nil, err
	}
	item.Role = externalUserRole(item.Role)
	return item, nil
}

func (s *MiscService) GetMyInviteCode(userID int64) (*repo.InviteCodeInfo, error) {
	invitation, err := s.invitationRepo.GetOrCreateActiveInvitation(userID)
	if err != nil {
		return nil, err
	}

	usedCount, err := s.invitationRepo.CountUsedInvitations(userID)
	if err != nil {
		return nil, err
	}

	return &repo.InviteCodeInfo{
		InviteCode: invitation.Code,
		UsedCount:  usedCount,
	}, nil
}

func (s *MiscService) UpdateMe(userID int64, nickname, avatarURL string, departmentID *int16, grade *int) (*repo.MeProfile, error) {
	if err := s.miscRepo.UpdateMe(userID, nickname, avatarURL, departmentID, grade); err != nil {
		return nil, err
	}
	return s.GetMe(userID)
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

func (s *MiscService) GetMyContributionSummary(userID int64) (*repo.ContributionSummary, error) {
	today := startOfContributionDay(time.Now().In(contributionLocation))
	currentWeekStart := today.AddDate(0, 0, -int(today.Weekday()))
	start := currentWeekStart.AddDate(0, 0, -(contributionWeeks-1)*7)
	end := currentWeekStart.AddDate(0, 0, 7)

	events, err := s.miscRepo.ListMyContributionEvents(userID, start, end)
	if err != nil {
		return nil, err
	}

	type daySummary struct {
		Score   int
		Actions []repo.ContributionAction
	}

	byDate := make(map[string]*daySummary, len(events))
	for _, event := range events {
		key := contributionDateKey(event.OccurredAt)
		item := byDate[key]
		if item == nil {
			item = &daySummary{}
			byDate[key] = item
		}
		item.Score += event.Score
		item.Actions = append(item.Actions, repo.ContributionAction{
			Type:  event.EventType,
			Label: event.Label,
			Score: event.Score,
		})
	}

	weeks := make([][]repo.ContributionCell, 0, contributionWeeks)
	totalScore := 0
	activeDays := 0
	maxDayScore := 0

	for weekIndex := 0; weekIndex < contributionWeeks; weekIndex += 1 {
		cells := make([]repo.ContributionCell, 0, 7)
		for dayIndex := 0; dayIndex < 7; dayIndex += 1 {
			date := start.AddDate(0, 0, weekIndex*7+dayIndex)
			dateKey := contributionDateKey(date)
			day := byDate[dateKey]
			isFuture := date.After(today)
			score := 0
			actions := make([]repo.ContributionAction, 0)
			if !isFuture && day != nil {
				score = day.Score
				actions = append(actions, day.Actions...)
			}
			if !isFuture && score > 0 {
				totalScore += score
				activeDays += 1
				if score > maxDayScore {
					maxDayScore = score
				}
			}

			cells = append(cells, repo.ContributionCell{
				Date:     dateKey,
				Score:    score,
				Level:    contributionLevel(score),
				IsFuture: isFuture,
				Actions:  actions,
			})
		}
		weeks = append(weeks, cells)
	}

	currentStreak := 0
	for cursor := today; ; cursor = cursor.AddDate(0, 0, -1) {
		day := byDate[contributionDateKey(cursor)]
		if day == nil || day.Score <= 0 {
			break
		}
		currentStreak += 1
	}

	return &repo.ContributionSummary{
		Weeks:         weeks,
		TotalScore:    totalScore,
		ActiveDays:    activeDays,
		CurrentStreak: currentStreak,
		MaxDayScore:   maxDayScore,
	}, nil
}

func (s *MiscService) GetUserContributionProfile(userID int64) (*repo.UserContributionProfile, error) {
	return s.miscRepo.GetUserContributionProfile(userID)
}

func (s *MiscService) ListAnnouncements() ([]repo.AnnouncementItem, error) {
	return s.miscRepo.ListAnnouncements()
}

func (s *MiscService) GetShowcaseStats() (*repo.ShowcaseStats, error) {
	return s.miscRepo.GetShowcaseStats()
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
	return items, total, nil
}

func (s *MiscService) ListNotifications(userID int64, isRead *bool, page, size int) ([]repo.NotificationItem, int64, error) {
	fillPagination(&page, &size)
	return s.miscRepo.ListNotifications(userID, isRead, page, size)
}

func (s *MiscService) ListHomeNotificationSummary(userID int64) (*repo.HomeNotificationSummary, error) {
	return s.miscRepo.ListHomeNotificationSummary(userID)
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
	case model.ReportTargetTypeCourse:
		return s.socialRepo.CourseExists(targetID)
	case model.ReportTargetTypeTeacherEvaluation:
		return s.socialRepo.TeacherEvaluationExists(targetID)
	case model.ReportTargetTypeCourseEvaluation:
		return s.socialRepo.CourseEvaluationExists(targetID)
	case model.ReportTargetTypeTeacherReply:
		return s.socialRepo.TeacherEvaluationReplyExists(targetID)
	case model.ReportTargetTypeCourseReply:
		return s.socialRepo.CourseEvaluationReplyExists(targetID)
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

func loadContributionLocation() *time.Location {
	location, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		return time.FixedZone("Asia/Shanghai", 8*60*60)
	}
	return location
}

func startOfContributionDay(t time.Time) time.Time {
	localTime := t.In(contributionLocation)
	year, month, day := localTime.Date()
	return time.Date(year, month, day, 0, 0, 0, 0, contributionLocation)
}

func contributionDateKey(t time.Time) string {
	return t.In(contributionLocation).Format("2006-01-02")
}

func contributionLevel(score int) int {
	switch {
	case score <= 0:
		return 0
	case score < 3:
		return 1
	case score < 6:
		return 2
	case score < 9:
		return 3
	default:
		return 4
	}
}
