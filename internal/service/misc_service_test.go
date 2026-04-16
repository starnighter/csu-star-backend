package service

import (
	"csu-star-backend/internal/model"
	"csu-star-backend/internal/repo"
	"errors"
	"testing"
	"time"

	"gorm.io/gorm"
)

type miscRepositoryStub struct {
	events                   []repo.ContributionEvent
	feedback                 *model.Feedbacks
	report                   *model.Reports
	correction               *model.Corrections
	teacherSupplementRequest *model.TeacherSupplementRequests
	courseSupplementRequest  *model.CourseSupplementRequests
	teacherSupplementView    *repo.TeacherSupplementRequestItem
	courseSupplementView     *repo.CourseSupplementRequestItem
}

func (m *miscRepositoryStub) GetMe(userID int64) (*repo.MeProfile, error) {
	return nil, nil
}

func (m *miscRepositoryStub) UpdateMe(userID int64, nickname, avatarURL string, departmentID *int16, grade *int) error {
	return nil
}

func (m *miscRepositoryStub) DailyCheckin(userID int64) (int, error) {
	return 0, nil
}

func (m *miscRepositoryStub) ListMyDownloads(userID int64, page, size int) ([]repo.DownloadHistoryItem, int64, error) {
	return nil, 0, nil
}

func (m *miscRepositoryStub) ListMyFavorites(userID int64, targetType string, page, size int) ([]repo.FavoriteItem, int64, error) {
	return nil, 0, nil
}

func (m *miscRepositoryStub) ListMyPoints(userID int64, page, size int) ([]repo.PointRecordItem, int64, error) {
	return nil, 0, nil
}

func (m *miscRepositoryStub) ListMyContributionEvents(userID int64, start, end time.Time) ([]repo.ContributionEvent, error) {
	return m.events, nil
}

func (m *miscRepositoryStub) ListAnnouncements() ([]repo.AnnouncementItem, error) {
	return nil, nil
}

func (m *miscRepositoryStub) GetShowcaseStats() (*repo.ShowcaseStats, error) {
	return nil, nil
}

func (m *miscRepositoryStub) CreateFeedback(feedback *model.Feedbacks) error {
	m.feedback = feedback
	return nil
}

func (m *miscRepositoryStub) CreateReport(report *model.Reports) error {
	m.report = report
	return nil
}

func (m *miscRepositoryStub) CreateCorrection(correction *model.Corrections) error {
	m.correction = correction
	return nil
}

func (m *miscRepositoryStub) Search(q, searchType string, page, size int, relevanceFirst bool) ([]repo.SearchResultItem, int64, error) {
	return nil, 0, nil
}

func (m *miscRepositoryStub) ListNotifications(userID int64, isRead *bool, page, size int) ([]repo.NotificationItem, int64, error) {
	return nil, 0, nil
}

func (m *miscRepositoryStub) ListHomeNotificationSummary(userID int64) (*repo.HomeNotificationSummary, error) {
	return &repo.HomeNotificationSummary{}, nil
}

func (m *miscRepositoryStub) CountUnreadNotifications(userID int64) (int64, error) {
	return 0, nil
}

func (m *miscRepositoryStub) MarkNotificationRead(userID, notificationID int64) error {
	return nil
}

func (m *miscRepositoryStub) MarkAllNotificationsRead(userID int64) error {
	return nil
}

func (m *miscRepositoryStub) CreateNotification(notification *model.Notifications) error {
	return nil
}

func (m *miscRepositoryStub) PurgeExpiredNotifications(now time.Time) error {
	return nil
}

func (m *miscRepositoryStub) CreateTeacherSupplementRequest(request *model.TeacherSupplementRequests) error {
	m.teacherSupplementRequest = request
	if request.ID == 0 {
		request.ID = 1
	}
	return nil
}

func (m *miscRepositoryStub) GetTeacherSupplementRequestByID(id int64) (*repo.TeacherSupplementRequestItem, error) {
	if m.teacherSupplementView != nil {
		return m.teacherSupplementView, nil
	}
	if m.teacherSupplementRequest == nil || m.teacherSupplementRequest.ID != id {
		return nil, gorm.ErrRecordNotFound
	}
	return &repo.TeacherSupplementRequestItem{
		ID:           m.teacherSupplementRequest.ID,
		UserID:       m.teacherSupplementRequest.UserID,
		Status:       string(m.teacherSupplementRequest.Status),
		Contact:      m.teacherSupplementRequest.Contact,
		TeacherName:  m.teacherSupplementRequest.TeacherName,
		DepartmentID: m.teacherSupplementRequest.DepartmentID,
		Remark:       m.teacherSupplementRequest.Remark,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}, nil
}

func (m *miscRepositoryStub) ListTeacherSupplementRequests(query repo.SupplementRequestListQuery) ([]repo.TeacherSupplementRequestItem, int64, error) {
	if m.teacherSupplementView == nil {
		return nil, 0, nil
	}
	return []repo.TeacherSupplementRequestItem{*m.teacherSupplementView}, 1, nil
}

func (m *miscRepositoryStub) UpdateTeacherSupplementRequest(id int64, updates map[string]interface{}) error {
	return nil
}

func (m *miscRepositoryStub) CreateCourseSupplementRequest(request *model.CourseSupplementRequests) error {
	m.courseSupplementRequest = request
	if request.ID == 0 {
		request.ID = 2
	}
	return nil
}

func (m *miscRepositoryStub) GetCourseSupplementRequestByID(id int64) (*repo.CourseSupplementRequestItem, error) {
	if m.courseSupplementView != nil {
		return m.courseSupplementView, nil
	}
	if m.courseSupplementRequest == nil || m.courseSupplementRequest.ID != id {
		return nil, gorm.ErrRecordNotFound
	}
	return &repo.CourseSupplementRequestItem{
		ID:         m.courseSupplementRequest.ID,
		UserID:     m.courseSupplementRequest.UserID,
		Status:     string(m.courseSupplementRequest.Status),
		Contact:    m.courseSupplementRequest.Contact,
		CourseName: m.courseSupplementRequest.CourseName,
		CourseType: string(m.courseSupplementRequest.CourseType),
		Remark:     m.courseSupplementRequest.Remark,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}, nil
}

func (m *miscRepositoryStub) ListCourseSupplementRequests(query repo.SupplementRequestListQuery) ([]repo.CourseSupplementRequestItem, int64, error) {
	if m.courseSupplementView == nil {
		return nil, 0, nil
	}
	return []repo.CourseSupplementRequestItem{*m.courseSupplementView}, 1, nil
}

func (m *miscRepositoryStub) UpdateCourseSupplementRequest(id int64, updates map[string]interface{}) error {
	return nil
}

type socialRepositoryStub struct {
	resourceExists          bool
	teacherEvaluationExists bool
	courseEvaluationExists  bool
	teacherReplyExists      bool
	courseReplyExists       bool
	commentExists           bool
	teacherExists           bool
	courseExists            bool
}

type invitationRepositoryStub struct {
	invitation *model.Invitations
	usedCount  int64
}

func (i *invitationRepositoryStub) CreateInvitation(invitation *model.Invitations) error {
	i.invitation = invitation
	return nil
}

func (i *invitationRepositoryStub) GetOrCreateActiveInvitation(inviterID int64) (*model.Invitations, error) {
	return i.invitation, nil
}

func (i *invitationRepositoryStub) CountUsedInvitations(inviterID int64) (int64, error) {
	return i.usedCount, nil
}

func (i *invitationRepositoryStub) FindInviterIDByCode(code string) (int64, error) {
	return 0, nil
}

func (i *invitationRepositoryStub) ConsumeInvitation(code string, inviteeID int64) (int64, error) {
	return 0, nil
}

func (s *socialRepositoryStub) CreateLike(like *model.Likes) error {
	return nil
}

func (s *socialRepositoryStub) DeleteLike(userID int64, targetType model.LikeTargetType, targetID int64) error {
	return nil
}

func (s *socialRepositoryStub) CreateLikeWithEffects(like *model.Likes, recipientID int64, notification *model.Notifications) error {
	return nil
}

func (s *socialRepositoryStub) DeleteLikeWithEffects(userID int64, targetType model.LikeTargetType, targetID int64) error {
	return nil
}

func (s *socialRepositoryStub) CreateFavorite(favorite *model.Favorites) error {
	return nil
}

func (s *socialRepositoryStub) DeleteFavorite(userID int64, targetType model.FavoriteTargetType, targetID int64) error {
	return nil
}

func (s *socialRepositoryStub) HasLike(userID int64, targetType model.LikeTargetType, targetID int64) (bool, error) {
	return false, nil
}

func (s *socialRepositoryStub) HasFavorite(userID int64, targetType model.FavoriteTargetType, targetID int64) (bool, error) {
	return false, nil
}

func (s *socialRepositoryStub) ListLikedTargetIDs(userID int64, targetType model.LikeTargetType, targetIDs []int64) (map[int64]bool, error) {
	return map[int64]bool{}, nil
}

func (s *socialRepositoryStub) ListFavoritedTargetIDs(userID int64, targetType model.FavoriteTargetType, targetIDs []int64) (map[int64]bool, error) {
	return map[int64]bool{}, nil
}

func (s *socialRepositoryStub) ResourceExists(resourceID int64) (bool, error) {
	return s.resourceExists, nil
}

func (s *socialRepositoryStub) TeacherEvaluationExists(evaluationID int64) (bool, error) {
	return s.teacherEvaluationExists, nil
}

func (s *socialRepositoryStub) CourseEvaluationExists(evaluationID int64) (bool, error) {
	return s.courseEvaluationExists, nil
}

func (s *socialRepositoryStub) TeacherEvaluationReplyExists(replyID int64) (bool, error) {
	return s.teacherReplyExists, nil
}

func (s *socialRepositoryStub) CourseEvaluationReplyExists(replyID int64) (bool, error) {
	return s.courseReplyExists, nil
}

func (s *socialRepositoryStub) CommentExists(commentID int64) (bool, error) {
	return s.commentExists, nil
}

func (s *socialRepositoryStub) UpdateResourceLikeCount(resourceID int64, delta int) error {
	return nil
}

func (s *socialRepositoryStub) UpdateCommentLikeCount(commentID int64, delta int) error {
	return nil
}

func (s *socialRepositoryStub) CreateNotification(notification *model.Notifications) error {
	return nil
}

func (s *socialRepositoryStub) GetLikeNotificationRecipient(targetType model.LikeTargetType, targetID int64) (int64, error) {
	return 0, nil
}

func (s *socialRepositoryStub) GetResourceOwnerID(resourceID int64) (int64, error) {
	return 0, nil
}

func (s *socialRepositoryStub) TeacherExists(teacherID int64) (bool, error) {
	return s.teacherExists, nil
}

func (s *socialRepositoryStub) CourseExists(courseID int64) (bool, error) {
	return s.courseExists, nil
}

func TestCreateReportLeavesProcessorFieldsNil(t *testing.T) {
	miscRepo := &miscRepositoryStub{}
	service := NewMiscService(nil, miscRepo, &socialRepositoryStub{
		teacherEvaluationExists: true,
	}, &invitationRepositoryStub{})

	err := service.CreateReport(1, string(model.ReportTargetTypeTeacherEvaluation), 5, "other", "评价举报")
	if err != nil {
		t.Fatalf("CreateReport() error = %v", err)
	}
	if miscRepo.report == nil {
		t.Fatalf("expected report to be created")
	}
	if miscRepo.report.ProcessorID != nil {
		t.Fatalf("expected processor_id to be nil, got %v", *miscRepo.report.ProcessorID)
	}
	if miscRepo.report.ProcessAt != nil {
		t.Fatalf("expected process_at to be nil, got %v", miscRepo.report.ProcessAt)
	}
}

func TestCreateFeedbackLeavesReplyFieldsNil(t *testing.T) {
	miscRepo := &miscRepositoryStub{}
	service := NewMiscService(nil, miscRepo, &socialRepositoryStub{}, &invitationRepositoryStub{})

	err := service.CreateFeedback(1, string(model.FeedbackTypeSuggestion), "标题", "内容", nil)
	if err != nil {
		t.Fatalf("CreateFeedback() error = %v", err)
	}
	if miscRepo.feedback == nil {
		t.Fatalf("expected feedback to be created")
	}
	if miscRepo.feedback.RepliedBy != nil {
		t.Fatalf("expected replied_by to be nil, got %v", *miscRepo.feedback.RepliedBy)
	}
	if miscRepo.feedback.RepliedAt != nil {
		t.Fatalf("expected replied_at to be nil, got %v", miscRepo.feedback.RepliedAt)
	}
}

func TestCreateCorrectionLeavesProcessorFieldsNil(t *testing.T) {
	miscRepo := &miscRepositoryStub{}
	service := NewMiscService(nil, miscRepo, &socialRepositoryStub{
		teacherExists: true,
	}, &invitationRepositoryStub{})

	err := service.CreateCorrection(1, string(model.CorrectionTargetTypeTeacher), 5, "name", "新名字")
	if err != nil {
		t.Fatalf("CreateCorrection() error = %v", err)
	}
	if miscRepo.correction == nil {
		t.Fatalf("expected correction to be created")
	}
	if miscRepo.correction.ProcessorID != nil {
		t.Fatalf("expected processor_id to be nil, got %v", *miscRepo.correction.ProcessorID)
	}
	if miscRepo.correction.ProcessAt != nil {
		t.Fatalf("expected process_at to be nil, got %v", miscRepo.correction.ProcessAt)
	}
}

func TestGetMyContributionSummary(t *testing.T) {
	today := startOfContributionDay(time.Now().In(contributionLocation))
	yesterday := today.AddDate(0, 0, -1)
	var futureDate time.Time
	hasFutureCell := int(today.Weekday()) < 6
	if hasFutureCell {
		futureDate = today.AddDate(0, 0, 1)
	}

	service := NewMiscService(nil, &miscRepositoryStub{
		events: []repo.ContributionEvent{
			{EventType: "resource_upload", Label: "资源上传", Score: 5, OccurredAt: today.Add(9 * time.Hour)},
			{EventType: "daily_checkin", Label: "每日签到", Score: 1, OccurredAt: today.Add(10 * time.Hour)},
			{EventType: "invite_reward", Label: "邀请奖励", Score: 3, OccurredAt: yesterday.Add(11 * time.Hour)},
		},
	}, &socialRepositoryStub{}, &invitationRepositoryStub{})
	if hasFutureCell {
		service.miscRepo = &miscRepositoryStub{
			events: []repo.ContributionEvent{
				{EventType: "resource_upload", Label: "资源上传", Score: 5, OccurredAt: today.Add(9 * time.Hour)},
				{EventType: "daily_checkin", Label: "每日签到", Score: 1, OccurredAt: today.Add(10 * time.Hour)},
				{EventType: "invite_reward", Label: "邀请奖励", Score: 3, OccurredAt: yesterday.Add(11 * time.Hour)},
				{EventType: "teacher_evaluation", Label: "发布教师评价", Score: 3, OccurredAt: futureDate.Add(8 * time.Hour)},
			},
		}
	}

	summary, err := service.GetMyContributionSummary(1)
	if err != nil {
		t.Fatalf("GetMyContributionSummary() error = %v", err)
	}

	if len(summary.Weeks) != contributionWeeks {
		t.Fatalf("expected %d weeks, got %d", contributionWeeks, len(summary.Weeks))
	}

	foundToday := false
	foundYesterday := false
	foundTomorrow := false
	for _, week := range summary.Weeks {
		if len(week) != 7 {
			t.Fatalf("expected 7 cells per week, got %d", len(week))
		}
		for _, cell := range week {
			switch cell.Date {
			case contributionDateKey(today):
				foundToday = true
				if cell.Score != 6 {
					t.Fatalf("expected today's score to be 6, got %d", cell.Score)
				}
				if cell.Level != 3 {
					t.Fatalf("expected today's level to be 3, got %d", cell.Level)
				}
				if len(cell.Actions) != 2 {
					t.Fatalf("expected 2 actions today, got %d", len(cell.Actions))
				}
			case contributionDateKey(yesterday):
				foundYesterday = true
				if cell.Score != 3 {
					t.Fatalf("expected yesterday's score to be 3, got %d", cell.Score)
				}
			case contributionDateKey(futureDate):
				foundTomorrow = true
				if !cell.IsFuture {
					t.Fatalf("expected tomorrow to be marked as future")
				}
				if cell.Score != 0 {
					t.Fatalf("expected future score to be 0, got %d", cell.Score)
				}
				if len(cell.Actions) != 0 {
					t.Fatalf("expected future actions to be hidden, got %d", len(cell.Actions))
				}
			}
		}
	}

	if !foundToday || !foundYesterday {
		t.Fatalf("expected to find today and yesterday cells, got today=%v yesterday=%v", foundToday, foundYesterday)
	}

	if hasFutureCell && !foundTomorrow {
		t.Fatalf("expected to find a future cell for %s", contributionDateKey(futureDate))
	}

	if summary.TotalScore != 9 {
		t.Fatalf("expected total score 9, got %d", summary.TotalScore)
	}

	if summary.ActiveDays != 2 {
		t.Fatalf("expected 2 active days, got %d", summary.ActiveDays)
	}

	if summary.CurrentStreak != 2 {
		t.Fatalf("expected current streak 2, got %d", summary.CurrentStreak)
	}

	if summary.MaxDayScore != 6 {
		t.Fatalf("expected max day score 6, got %d", summary.MaxDayScore)
	}
}

func TestCreateSupplementRequestTeacher(t *testing.T) {
	repoStub := &miscRepositoryStub{}
	service := NewMiscService(nil, repoStub, &socialRepositoryStub{}, &invitationRepositoryStub{})

	item, err := service.CreateSupplementRequest(
		1,
		"teacher",
		"test@example.com",
		"张老师",
		ptrInt16(1),
		"",
		"",
		"希望补录",
	)
	if err != nil {
		t.Fatalf("CreateSupplementRequest() error = %v", err)
	}

	if item == nil || repoStub.teacherSupplementRequest == nil {
		t.Fatalf("expected supplement request to be created")
	}

	if item.RequestType != "teacher" {
		t.Fatalf("expected request type teacher, got %s", item.RequestType)
	}

	if repoStub.teacherSupplementRequest.DepartmentID != 1 {
		t.Fatalf("expected department id to be preserved")
	}
}

func TestCreateSupplementRequestCourseRejectsInvalidCourseType(t *testing.T) {
	service := NewMiscService(nil, &miscRepositoryStub{}, &socialRepositoryStub{}, &invitationRepositoryStub{})

	_, err := service.CreateSupplementRequest(
		1,
		"course",
		"test@example.com",
		"",
		nil,
		"大学英语",
		"未知类型",
		"",
	)
	if err == nil {
		t.Fatalf("expected invalid payload error")
	}
	if !errors.Is(err, ErrSupplementRequestInvalidPayload) {
		t.Fatalf("expected ErrSupplementRequestInvalidPayload, got %v", err)
	}
}

func ptrInt16(value int16) *int16 {
	return &value
}

func TestCreateSupplementRequestCourse(t *testing.T) {
	repoStub := &miscRepositoryStub{}
	service := NewMiscService(nil, repoStub, &socialRepositoryStub{}, &invitationRepositoryStub{})

	item, err := service.CreateSupplementRequest(
		1,
		"course",
		"test@example.com",
		"",
		nil,
		"大学英语",
		"public",
		"",
	)
	if err != nil {
		t.Fatalf("CreateSupplementRequest() error = %v", err)
	}

	if item == nil || repoStub.courseSupplementRequest == nil {
		t.Fatalf("expected supplement request to be created")
	}

	if item.RequestType != "course" {
		t.Fatalf("expected request type course, got %s", item.RequestType)
	}
	if repoStub.courseSupplementRequest.CourseType != model.CourseTypePublic {
		t.Fatalf("expected course type public, got %s", repoStub.courseSupplementRequest.CourseType)
	}
}

func TestCreateSupplementRequestTeacherRejectsMissingDepartment(t *testing.T) {
	service := NewMiscService(nil, &miscRepositoryStub{}, &socialRepositoryStub{}, &invitationRepositoryStub{})

	_, err := service.CreateSupplementRequest(
		1,
		"teacher",
		"test@example.com",
		"张老师",
		nil,
		"",
		"",
		"",
	)
	if !errors.Is(err, ErrSupplementRequestInvalidPayload) {
		t.Fatalf("expected ErrSupplementRequestInvalidPayload, got %v", err)
	}
}

func TestListSupplementRequestsMergesBothKinds(t *testing.T) {
	now := time.Now()
	repoStub := &miscRepositoryStub{
		teacherSupplementView: &repo.TeacherSupplementRequestItem{
			ID:             1,
			UserID:         10,
			Status:         "pending",
			Contact:        "teacher@test.com",
			TeacherName:    "张老师",
			DepartmentID:   1,
			DepartmentName: "计算机学院",
			CreatedAt:      now,
			UpdatedAt:      now,
		},
		courseSupplementView: &repo.CourseSupplementRequestItem{
			ID:         2,
			UserID:     11,
			Status:     "pending",
			Contact:    "course@test.com",
			CourseName: "大学英语",
			CourseType: "public",
			CreatedAt:  now.Add(-time.Minute),
			UpdatedAt:  now.Add(-time.Minute),
		},
	}
	service := NewMiscService(nil, repoStub, &socialRepositoryStub{}, &invitationRepositoryStub{})

	items, total, err := service.ListSupplementRequests(repo.SupplementRequestListQuery{
		Page: 1,
		Size: 10,
	})
	if err != nil {
		t.Fatalf("ListSupplementRequests() error = %v", err)
	}
	if total != 2 {
		t.Fatalf("expected total 2, got %d", total)
	}
	if len(items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(items))
	}
	if items[0].RequestType != "teacher" || items[1].RequestType != "course" {
		t.Fatalf("expected teacher then course ordering, got %s then %s", items[0].RequestType, items[1].RequestType)
	}
}

func TestGetMyInviteCode(t *testing.T) {
	expiresAt := time.Date(9999, 12, 31, 23, 59, 59, 0, time.UTC)
	service := NewMiscService(
		nil,
		&miscRepositoryStub{},
		&socialRepositoryStub{},
		&invitationRepositoryStub{
			invitation: &model.Invitations{
				Code:      "AB12CD",
				ExpiresAt: &expiresAt,
			},
			usedCount: 2,
		},
	)

	info, err := service.GetMyInviteCode(1)
	if err != nil {
		t.Fatalf("GetMyInviteCode() error = %v", err)
	}

	if info.InviteCode != "AB12CD" {
		t.Fatalf("expected invite code AB12CD, got %s", info.InviteCode)
	}
	if info.UsedCount != 2 {
		t.Fatalf("expected used count 2, got %d", info.UsedCount)
	}
}
