package service

import (
	"csu-star-backend/internal/model"
	"csu-star-backend/internal/repo"
	"csu-star-backend/logger"
	"csu-star-backend/pkg/utils"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/url"
	"strings"
	"time"

	"go.uber.org/zap"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

var (
	ErrAdminTargetNotFound = errors.New("admin target not found")
	ErrAdminInvalidAction  = errors.New("admin invalid action")
	ErrAdminInvalidPayload = errors.New("admin invalid payload")
	ErrAdminConflict       = errors.New("admin conflict")
)

type AdminService struct {
	db           *gorm.DB
	adminRepo    repo.AdminRepository
	courseRepo   repo.CourseRepository
	teacherRepo  repo.TeacherRepository
	commentRepo  repo.CommentRepository
	socialRepo   repo.SocialRepository
	resourceRepo repo.ResourceRepository
}

func NewAdminService(
	db *gorm.DB,
	ar repo.AdminRepository,
	cr repo.CourseRepository,
	tr repo.TeacherRepository,
	cor repo.CommentRepository,
	sr repo.SocialRepository,
	rr repo.ResourceRepository,
) *AdminService {
	return &AdminService{
		db:           db,
		adminRepo:    ar,
		courseRepo:   cr,
		teacherRepo:  tr,
		commentRepo:  cor,
		socialRepo:   sr,
		resourceRepo: rr,
	}
}

func (s *AdminService) GetStatistics() (*repo.AdminStatistics, error) {
	return s.adminRepo.GetStatistics()
}

func (s *AdminService) ListReports(status string, page, size int) ([]repo.AdminReportItem, int64, error) {
	fillPagination(&page, &size)
	return s.adminRepo.ListReports(status, page, size)
}

func (s *AdminService) HandleReport(reportID, operatorID int64, action, remark string, ip net.IP) error {
	return s.withWriteTx(func(adminRepo repo.AdminRepository, _ repo.CourseRepository, tx *gorm.DB) error {
		report, err := adminRepo.GetReportByID(reportID)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrAdminTargetNotFound
		}
		if err != nil {
			return err
		}
		if report.Status != model.ReportStatusPending {
			return ErrAdminConflict
		}

		var nextStatus model.ReportStatus
		var auditAction model.AuditAction
		switch action {
		case "resolve":
			nextStatus = model.ReportStatusResolved
			auditAction = model.AuditActionApprove
		case "dismiss":
			nextStatus = model.ReportStatusDismissed
			auditAction = model.AuditActionReject
		default:
			return ErrAdminInvalidAction
		}

		now := time.Now()
		oldValues := jsonMap("status", report.Status, "process_note", report.ProcessNote)
		var reportOwnerID int64
		if action == "resolve" {
			reportOwnerID, err = s.resolveReportTargetOwnerID(report, tx)
			if err != nil {
				return err
			}
			if err := s.applyResolvedReportEffects(report, tx); err != nil {
				return err
			}
		}
		report.Status = nextStatus
		report.ProcessorID = &operatorID
		report.ProcessAt = &now
		report.ProcessNote = strings.TrimSpace(remark)

		if err := adminRepo.UpdateReport(report); err != nil {
			return err
		}

		if err := adminRepo.CreateAuditLog(&model.AuditLogs{
			OperatorID: operatorID,
			Action:     auditAction,
			TargetType: "report",
			TargetID:   report.ID,
			OldValues:  oldValues,
			NewValues:  jsonMap("status", report.Status, "process_note", report.ProcessNote),
			Reason:     report.ProcessNote,
			IpAddress:  ip,
		}); err != nil {
			return err
		}

		if err := adminRepo.CreateNotification(buildReportHandledNotification(report)); err != nil {
			return err
		}

		if action == "resolve" && reportOwnerID > 0 && reportOwnerID != report.UserID {
			notification := buildReportedContentHandledNotification(report, reportOwnerID, report.ProcessNote)
			if notification != nil {
				return adminRepo.CreateNotification(notification)
			}
		}
		return nil
	})
}

func (s *AdminService) applyResolvedReportEffects(report *model.Reports, tx *gorm.DB) error {
	if report == nil {
		return nil
	}

	switch report.TargetType {
	case model.ReportTargetTypeResource:
		return s.deleteResourceForReport(report.TargetID, tx)
	case model.ReportTargetTypeTeacherEvaluation:
		return s.deleteTeacherEvaluationForReport(report.TargetID, tx)
	case model.ReportTargetTypeCourseEvaluation:
		return s.deleteCourseEvaluationForReport(report.TargetID, tx)
	case model.ReportTargetTypeTeacherReply:
		return s.deleteTeacherEvaluationReplyForReport(report.TargetID, tx)
	case model.ReportTargetTypeCourseReply:
		return s.deleteCourseEvaluationReplyForReport(report.TargetID, tx)
	case model.ReportTargetTypeComment:
		return s.deleteCommentForReport(report.TargetID, tx)
	default:
		return nil
	}
}

func (s *AdminService) resolveReportTargetOwnerID(report *model.Reports, tx *gorm.DB) (int64, error) {
	if report == nil || s.socialRepo == nil {
		return 0, nil
	}

	socialRepo := s.socialRepoWithTx(tx)
	switch report.TargetType {
	case model.ReportTargetTypeResource:
		return socialRepo.GetResourceOwnerID(report.TargetID)
	case model.ReportTargetTypeTeacherEvaluation:
		return socialRepo.GetLikeNotificationRecipient(model.LikeTargetTypeTeacherEvaluation, report.TargetID)
	case model.ReportTargetTypeCourseEvaluation:
		return socialRepo.GetLikeNotificationRecipient(model.LikeTargetTypeCourseEvaluation, report.TargetID)
	case model.ReportTargetTypeTeacherReply:
		return socialRepo.GetLikeNotificationRecipient(model.LikeTargetTypeTeacherReply, report.TargetID)
	case model.ReportTargetTypeCourseReply:
		return socialRepo.GetLikeNotificationRecipient(model.LikeTargetTypeCourseReply, report.TargetID)
	case model.ReportTargetTypeComment:
		return socialRepo.GetLikeNotificationRecipient(model.LikeTargetTypeComment, report.TargetID)
	default:
		return 0, nil
	}
}

func (s *AdminService) deleteResourceForReport(resourceID int64, tx *gorm.DB) error {
	resourceRepo := s.resourceRepoWithTx(tx)
	_, err := resourceRepo.GetResourceByID(resourceID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil
	}
	if err != nil {
		return err
	}
	return resourceRepo.SoftDeleteResource(resourceID)
}

func (s *AdminService) deleteTeacherEvaluationForReport(evaluationID int64, tx *gorm.DB) error {
	teacherRepo := s.teacherRepoWithTx(tx)
	courseRepo := s.courseRepoWithTx(tx)

	evaluation, err := teacherRepo.GetTeacherEvaluationByID(evaluationID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil
	}
	if err != nil {
		return err
	}

	if tx == nil {
		return nil
	}

	if evaluation.MirrorEvaluationID != nil {
		if err := tx.Delete(&model.CourseEvaluations{}, *evaluation.MirrorEvaluationID).Error; err != nil {
			return err
		}
	}
	if err := tx.Delete(&model.TeacherEvaluations{}, evaluationID).Error; err != nil {
		return err
	}
	if err := teacherRepo.RecalculateTeacherStats(evaluation.TeacherID); err != nil {
		return err
	}
	if evaluation.CourseID != nil {
		return courseRepo.RecalculateCourseStats(*evaluation.CourseID)
	}
	return nil
}

func (s *AdminService) deleteCourseEvaluationForReport(evaluationID int64, tx *gorm.DB) error {
	courseRepo := s.courseRepoWithTx(tx)
	teacherRepo := s.teacherRepoWithTx(tx)

	evaluation, err := courseRepo.GetCourseEvaluationByID(evaluationID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil
	}
	if err != nil {
		return err
	}

	if tx == nil {
		return nil
	}

	if evaluation.MirrorEvaluationID != nil {
		if err := tx.Delete(&model.TeacherEvaluations{}, *evaluation.MirrorEvaluationID).Error; err != nil {
			return err
		}
	}
	if err := tx.Delete(&model.CourseEvaluations{}, evaluationID).Error; err != nil {
		return err
	}
	if err := courseRepo.RecalculateCourseStats(evaluation.CourseID); err != nil {
		return err
	}
	if evaluation.TeacherID != nil {
		return teacherRepo.RecalculateTeacherStats(*evaluation.TeacherID)
	}
	return nil
}

func (s *AdminService) deleteTeacherEvaluationReplyForReport(replyID int64, tx *gorm.DB) error {
	teacherRepo := s.teacherRepoWithTx(tx)
	_, err := teacherRepo.GetTeacherEvaluationReplyByID(replyID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil
	}
	if err != nil {
		return err
	}
	return teacherRepo.DeleteTeacherEvaluationReply(replyID)
}

func (s *AdminService) deleteCourseEvaluationReplyForReport(replyID int64, tx *gorm.DB) error {
	courseRepo := s.courseRepoWithTx(tx)
	_, err := courseRepo.GetCourseEvaluationReplyByID(replyID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil
	}
	if err != nil {
		return err
	}
	return courseRepo.DeleteCourseEvaluationReply(replyID)
}

func (s *AdminService) deleteCommentForReport(commentID int64, tx *gorm.DB) error {
	commentRepo := s.commentRepoWithTx(tx)
	comment, err := commentRepo.GetCommentByID(commentID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil
	}
	if err != nil {
		return err
	}
	return commentRepo.SoftDeleteCommentWithEffects(
		commentID,
		comment.TargetID,
		comment.TargetType == model.CommentTargetTypeResource,
	)
}

func (s *AdminService) ListCorrections(status string, page, size int) ([]repo.AdminCorrectionItem, int64, error) {
	fillPagination(&page, &size)
	return s.adminRepo.ListCorrections(status, page, size)
}

func (s *AdminService) HandleCorrection(correctionID, operatorID int64, action, remark string, ip net.IP) error {
	return s.withWriteTx(func(adminRepo repo.AdminRepository, _ repo.CourseRepository, tx *gorm.DB) error {
		correction, err := adminRepo.GetCorrectionByID(correctionID)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrAdminTargetNotFound
		}
		if err != nil {
			return err
		}
		if correction.Status != model.CorrectionStatusPending {
			return ErrAdminConflict
		}

		oldValues := jsonMap("status", correction.Status, "suggested_value", correction.SuggestedValue)
		now := time.Now()
		switch action {
		case "approve":
			if err := s.applyCorrection(adminRepo, tx, correction); err != nil {
				return err
			}
			correction.Status = model.CorrectionStatusAccepted
		case "reject":
			correction.Status = model.CorrectionStatusRejected
		default:
			return ErrAdminInvalidAction
		}
		correction.ProcessorID = &operatorID
		correction.ProcessAt = &now
		correction.ProcessNote = strings.TrimSpace(remark)
		if err := adminRepo.UpdateCorrection(correction); err != nil {
			return err
		}

		auditAction := model.AuditActionReject
		if correction.Status == model.CorrectionStatusAccepted {
			auditAction = model.AuditActionApprove
		}
		if err := adminRepo.CreateAuditLog(&model.AuditLogs{
			OperatorID: operatorID,
			Action:     auditAction,
			TargetType: "correction",
			TargetID:   correction.ID,
			OldValues:  oldValues,
			NewValues:  jsonMap("status", correction.Status, "process_note", correction.ProcessNote),
			Reason:     correction.ProcessNote,
			IpAddress:  ip,
		}); err != nil {
			return err
		}

		return adminRepo.CreateNotification(buildCorrectionHandledNotification(correction))
	})
}

func (s *AdminService) ListFeedbacks(status string, page, size int) ([]repo.AdminFeedbackItem, int64, error) {
	fillPagination(&page, &size)
	return s.adminRepo.ListFeedbacks(status, page, size)
}

func (s *AdminService) ReplyFeedback(feedbackID, operatorID int64, reply, status string, ip net.IP) error {
	reply = strings.TrimSpace(reply)
	if reply == "" {
		return ErrAdminInvalidPayload
	}

	return s.withWriteTx(func(adminRepo repo.AdminRepository, _ repo.CourseRepository, _ *gorm.DB) error {
		item, err := adminRepo.GetFeedbackByID(feedbackID)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrAdminTargetNotFound
		}
		if err != nil {
			return err
		}

		oldValues := jsonMap("status", item.Status, "reply", item.Reply)
		item.Reply = reply
		item.RepliedBy = &operatorID
		repliedAt := time.Now()
		item.RepliedAt = &repliedAt
		if status != "" {
			item.Status = model.FeedbackStatus(status)
		} else if item.Status == model.FeedbackStatusPending {
			item.Status = model.FeedbackStatusProcessing
		}
		if err := adminRepo.UpdateFeedback(item); err != nil {
			return err
		}
		if err := adminRepo.CreateAuditLog(&model.AuditLogs{
			OperatorID: operatorID,
			Action:     model.AuditActionApprove,
			TargetType: "feedback",
			TargetID:   item.ID,
			OldValues:  oldValues,
			NewValues:  jsonMap("status", item.Status, "reply", item.Reply),
			Reason:     reply,
			IpAddress:  ip,
		}); err != nil {
			return err
		}
		return adminRepo.CreateNotification(buildFeedbackReplyNotification(item))
	})
}

func (s *AdminService) ListUsers(status, role, keyword string, page, size int) ([]repo.AdminUserItem, int64, error) {
	fillPagination(&page, &size)
	return s.adminRepo.ListUsers(status, role, keyword, page, size)
}

func (s *AdminService) ListUserViolations(userID, page, size int64) ([]repo.AdminUserViolationItem, int64, error) {
	parsedPage := int(page)
	parsedSize := int(size)
	fillPagination(&parsedPage, &parsedSize)
	return s.adminRepo.ListUserViolations(userID, parsedPage, parsedSize)
}

func (s *AdminService) BanUser(userID, operatorID int64, reason string, ip net.IP) error {
	return s.withWriteTx(func(adminRepo repo.AdminRepository, _ repo.CourseRepository, _ *gorm.DB) error {
		item, err := adminRepo.GetUserByID(userID)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrAdminTargetNotFound
		}
		if err != nil {
			return err
		}
		if item.Status == model.UserStatusBanned {
			return ErrAdminConflict
		}
		oldValues := jsonMap("status", item.Status)
		item.Status = model.UserStatusBanned
		item.BanUntil = nil
		item.BanReason = strings.TrimSpace(reason)
		item.BanSource = model.UserBanSourceAdmin
		if err := adminRepo.UpdateUser(item); err != nil {
			return err
		}
		return adminRepo.CreateAuditLog(&model.AuditLogs{
			OperatorID: operatorID,
			Action:     model.AuditActionBan,
			TargetType: "user",
			TargetID:   item.ID,
			OldValues:  oldValues,
			NewValues:  jsonMap("status", item.Status, "ban_until", item.BanUntil, "ban_reason", item.BanReason, "ban_source", item.BanSource),
			Reason:     reason,
			IpAddress:  ip,
		})
	})
}

func (s *AdminService) UnbanUser(userID, operatorID int64, ip net.IP) error {
	return s.withWriteTx(func(adminRepo repo.AdminRepository, _ repo.CourseRepository, _ *gorm.DB) error {
		item, err := adminRepo.GetUserByID(userID)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrAdminTargetNotFound
		}
		if err != nil {
			return err
		}
		if item.Status == model.UserStatusActive {
			return ErrAdminConflict
		}
		oldValues := jsonMap("status", item.Status)
		item.Status = model.UserStatusActive
		item.BanUntil = nil
		item.BanReason = ""
		item.BanSource = ""
		if err := adminRepo.UpdateUser(item); err != nil {
			return err
		}
		return adminRepo.CreateAuditLog(&model.AuditLogs{
			OperatorID: operatorID,
			Action:     model.AuditActionUnban,
			TargetType: "user",
			TargetID:   item.ID,
			OldValues:  oldValues,
			NewValues:  jsonMap("status", item.Status, "ban_until", item.BanUntil, "ban_reason", item.BanReason, "ban_source", item.BanSource),
			IpAddress:  ip,
		})
	})
}

func (s *AdminService) AdjustUserPoints(userID, operatorID int64, delta int, reason string, ip net.IP) (int, error) {
	if delta == 0 || strings.TrimSpace(reason) == "" {
		return 0, ErrAdminInvalidPayload
	}
	return withWriteTxValue(s, func(adminRepo repo.AdminRepository, _ repo.CourseRepository, _ *gorm.DB) (int, error) {
		item, err := adminRepo.GetUserByID(userID)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return 0, ErrAdminTargetNotFound
		}
		if err != nil {
			return 0, err
		}
		oldValues := jsonMap("points", item.Points)
		balance, err := adminRepo.AdjustUserPoints(userID, delta, reason)
		if err != nil {
			return 0, err
		}
		if err := adminRepo.CreateAuditLog(&model.AuditLogs{
			OperatorID: operatorID,
			Action:     model.AuditActionManualAdjustPoints,
			TargetType: "user",
			TargetID:   userID,
			OldValues:  oldValues,
			NewValues:  jsonMap("points", balance, "delta", delta),
			Reason:     reason,
			IpAddress:  ip,
		}); err != nil {
			return 0, err
		}
		if err := adminRepo.CreateNotification(buildPointsChangedNotification(userID, reason)); err != nil {
			return 0, err
		}
		return balance, nil
	})
}

func (s *AdminService) SendUserNotification(userID, operatorID int64, title, content string, result model.NotificationResult, ip net.IP) error {
	title = strings.TrimSpace(title)
	content = strings.TrimSpace(content)
	if title == "" || content == "" {
		return ErrAdminInvalidPayload
	}

	return s.withWriteTx(func(adminRepo repo.AdminRepository, _ repo.CourseRepository, _ *gorm.DB) error {
		user, err := adminRepo.GetUserByID(userID)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrAdminTargetNotFound
		}
		if err != nil {
			return err
		}
		if err := adminRepo.CreateNotification(buildAdminDirectNotification(user.ID, title, content, result)); err != nil {
			return err
		}
		return adminRepo.CreateAuditLog(&model.AuditLogs{
			OperatorID: operatorID,
			Action:     model.AuditActionApprove,
			TargetType: "notification",
			TargetID:   user.ID,
			NewValues: mustJSON(map[string]interface{}{
				"user_id":  user.ID,
				"title":    title,
				"content":  content,
				"category": model.NotificationCategoryAdminMessage,
				"result":   result,
			}),
			Reason:    "send direct notification",
			IpAddress: ip,
		})
	})
}

func (s *AdminService) ListAnnouncements(page, size int) ([]repo.AdminAnnouncementItem, int64, error) {
	fillPagination(&page, &size)
	return s.adminRepo.ListAnnouncements(page, size)
}

func (s *AdminService) CreateAnnouncement(operatorID int64, input model.Announcements, ip net.IP) (*model.Announcements, error) {
	if strings.TrimSpace(input.Title) == "" || strings.TrimSpace(input.Content) == "" {
		return nil, ErrAdminInvalidPayload
	}
	if input.Type == "" {
		input.Type = model.AnnouncementTypeNotice
	}
	if input.PublishedAt.IsZero() {
		input.PublishedAt = time.Now()
	}
	return withWriteTxValue(s, func(adminRepo repo.AdminRepository, _ repo.CourseRepository, _ *gorm.DB) (*model.Announcements, error) {
		if err := adminRepo.CreateAnnouncement(&input); err != nil {
			return nil, err
		}
		if input.IsPublished {
			if err := adminRepo.CreateNotification(buildAnnouncementNotification(input)); err != nil {
				return nil, err
			}
		}
		if err := adminRepo.CreateAuditLog(&model.AuditLogs{
			OperatorID: operatorID,
			Action:     model.AuditActionApprove,
			TargetType: "announcement",
			TargetID:   input.ID,
			NewValues:  mustJSON(input),
			Reason:     "create announcement",
			IpAddress:  ip,
		}); err != nil {
			return nil, err
		}
		return &input, nil
	})
}

func (s *AdminService) UpdateAnnouncement(operatorID, id int64, updates map[string]interface{}, ip net.IP) (*model.Announcements, error) {
	if len(updates) == 0 {
		return nil, ErrAdminInvalidPayload
	}
	return withWriteTxValue(s, func(adminRepo repo.AdminRepository, _ repo.CourseRepository, _ *gorm.DB) (*model.Announcements, error) {
		item, err := adminRepo.GetAnnouncementByID(id)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrAdminTargetNotFound
		}
		if err != nil {
			return nil, err
		}
		oldValues := mustJSON(item)
		wasPublished := item.IsPublished
		oldTitle := item.Title
		oldContent := item.Content
		applyAnnouncementUpdates(item, updates)
		if item.IsPublished && item.PublishedAt.IsZero() {
			item.PublishedAt = time.Now()
		}
		if err := adminRepo.UpdateAnnouncement(item); err != nil {
			return nil, err
		}
		if shouldBroadcastAnnouncementUpdate(wasPublished, oldTitle, oldContent, item) {
			if err := adminRepo.CreateNotification(buildAnnouncementNotification(*item)); err != nil {
				return nil, err
			}
		}
		if err := adminRepo.CreateAuditLog(&model.AuditLogs{
			OperatorID: operatorID,
			Action:     model.AuditActionApprove,
			TargetType: "announcement",
			TargetID:   item.ID,
			OldValues:  oldValues,
			NewValues:  mustJSON(item),
			Reason:     "update announcement",
			IpAddress:  ip,
		}); err != nil {
			return nil, err
		}
		return item, nil
	})
}

func (s *AdminService) DeleteAnnouncement(operatorID, id int64, ip net.IP) error {
	return s.withWriteTx(func(adminRepo repo.AdminRepository, _ repo.CourseRepository, _ *gorm.DB) error {
		item, err := adminRepo.GetAnnouncementByID(id)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrAdminTargetNotFound
		}
		if err != nil {
			return err
		}
		if err := adminRepo.DeleteAnnouncement(id); err != nil {
			return err
		}
		return adminRepo.CreateAuditLog(&model.AuditLogs{
			OperatorID: operatorID,
			Action:     model.AuditActionDelete,
			TargetType: "announcement",
			TargetID:   id,
			OldValues:  mustJSON(item),
			Reason:     "delete announcement",
			IpAddress:  ip,
		})
	})
}

func (s *AdminService) ListCourses(status, courseType, keyword string, page, size int) ([]repo.AdminCourseItem, int64, error) {
	fillPagination(&page, &size)
	return s.adminRepo.ListCourses(status, courseType, keyword, page, size)
}

func (s *AdminService) CreateCourse(operatorID int64, course *model.Courses, ip net.IP) (*model.Courses, error) {
	if strings.TrimSpace(course.Name) == "" || course.CourseType == "" {
		return nil, ErrAdminInvalidPayload
	}
	course.Status = model.CourseStatusActive
	return withWriteTxValue(s, func(adminRepo repo.AdminRepository, _ repo.CourseRepository, _ *gorm.DB) (*model.Courses, error) {
		if err := adminRepo.CreateCourse(course); err != nil {
			return nil, err
		}
		if err := adminRepo.CreateAuditLog(&model.AuditLogs{
			OperatorID: operatorID,
			Action:     model.AuditActionApprove,
			TargetType: "course",
			TargetID:   course.ID,
			NewValues:  mustJSON(course),
			Reason:     "create course",
			IpAddress:  ip,
		}); err != nil {
			return nil, err
		}
		return course, nil
	})
}

func (s *AdminService) UpdateCourse(operatorID, courseID int64, updates map[string]interface{}, ip net.IP) (*model.Courses, error) {
	if len(updates) == 0 {
		return nil, ErrAdminInvalidPayload
	}
	return withWriteTxValue(s, func(adminRepo repo.AdminRepository, _ repo.CourseRepository, _ *gorm.DB) (*model.Courses, error) {
		item, err := adminRepo.GetCourseByID(courseID)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrAdminTargetNotFound
		}
		if err != nil {
			return nil, err
		}
		updates["updated_at"] = time.Now()
		oldValues := mustJSON(item)
		if err := adminRepo.UpdateCourseFields(courseID, updates); err != nil {
			return nil, err
		}
		item, err = adminRepo.GetCourseByID(courseID)
		if err != nil {
			return nil, err
		}
		if err := adminRepo.CreateAuditLog(&model.AuditLogs{
			OperatorID: operatorID,
			Action:     model.AuditActionApprove,
			TargetType: "course",
			TargetID:   courseID,
			OldValues:  oldValues,
			NewValues:  mustJSON(item),
			Reason:     "update course",
			IpAddress:  ip,
		}); err != nil {
			return nil, err
		}
		return item, nil
	})
}

func (s *AdminService) DeleteCourse(operatorID, courseID int64, ip net.IP) error {
	return s.withWriteTx(func(adminRepo repo.AdminRepository, _ repo.CourseRepository, _ *gorm.DB) error {
		item, err := adminRepo.GetCourseByID(courseID)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrAdminTargetNotFound
		}
		if err != nil {
			return err
		}
		if item.Status == model.CourseStatusDeleted {
			return ErrAdminConflict
		}
		oldValues := mustJSON(item)
		if err := adminRepo.UpdateCourseFields(courseID, map[string]interface{}{
			"status":     model.CourseStatusDeleted,
			"updated_at": time.Now(),
		}); err != nil {
			return err
		}
		return adminRepo.CreateAuditLog(&model.AuditLogs{
			OperatorID: operatorID,
			Action:     model.AuditActionDelete,
			TargetType: "course",
			TargetID:   courseID,
			OldValues:  oldValues,
			NewValues:  jsonMap("status", model.CourseStatusDeleted),
			Reason:     "delete course",
			IpAddress:  ip,
		})
	})
}

func (s *AdminService) ListCourseRelations(courseID int64) ([]repo.AdminCourseRelationItem, error) {
	if exists, err := s.courseRepo.CourseExists(courseID); err != nil {
		return nil, err
	} else if !exists {
		return nil, ErrAdminTargetNotFound
	}
	return s.adminRepo.ListCourseRelations(courseID)
}

func (s *AdminService) AddCourseRelation(operatorID, courseID, teacherID int64, ip net.IP) (*model.CourseTeachers, error) {
	if exists, err := s.courseRepo.CourseExists(courseID); err != nil {
		return nil, err
	} else if !exists {
		return nil, ErrAdminTargetNotFound
	}
	if exists, err := s.teacherRepo.TeacherExists(teacherID); err != nil {
		return nil, err
	} else if !exists {
		return nil, ErrAdminTargetNotFound
	}

	return withWriteTxValue(s, func(adminRepo repo.AdminRepository, _ repo.CourseRepository, tx *gorm.DB) (*model.CourseTeachers, error) {
		teacherRepo := s.teacherRepoWithTx(tx)
		existing, err := teacherRepo.GetCourseTeacherRelation(courseID, teacherID)
		switch {
		case err == nil:
			if existing.Status == model.CourseTeacherRelationStatusActive {
				return nil, ErrAdminConflict
			}
			oldValues := mustJSON(existing)
			existing.Status = model.CourseTeacherRelationStatusActive
			existing.CanceledAt = nil
			if err := teacherRepo.UpdateCourseTeacherRelation(existing); err != nil {
				return nil, err
			}
			if err := adminRepo.CreateAuditLog(&model.AuditLogs{
				OperatorID: operatorID,
				Action:     model.AuditActionApprove,
				TargetType: "course_teacher_relation",
				TargetID:   existing.ID,
				OldValues:  oldValues,
				NewValues:  mustJSON(existing),
				Reason:     "restore course teacher relation",
				IpAddress:  ip,
			}); err != nil {
				return nil, err
			}
			if err := s.courseRepoWithTx(tx).RecalculateCourseStats(courseID); err != nil {
				return nil, err
			}
			if err := teacherRepo.RecalculateTeacherStats(teacherID); err != nil {
				return nil, err
			}
			return existing, nil
		case !errors.Is(err, gorm.ErrRecordNotFound):
			return nil, err
		}

		relation := &model.CourseTeachers{
			CourseID:  courseID,
			TeacherID: teacherID,
			Status:    model.CourseTeacherRelationStatusActive,
		}
		if err := teacherRepo.CreateCourseTeacherRelation(relation); err != nil {
			if isDuplicateKeyErr(err) {
				return nil, ErrAdminConflict
			}
			return nil, err
		}
		if err := adminRepo.CreateAuditLog(&model.AuditLogs{
			OperatorID: operatorID,
			Action:     model.AuditActionApprove,
			TargetType: "course_teacher_relation",
			TargetID:   relation.ID,
			NewValues:  mustJSON(relation),
			Reason:     "create course teacher relation",
			IpAddress:  ip,
		}); err != nil {
			return nil, err
		}
		if err := s.courseRepoWithTx(tx).RecalculateCourseStats(courseID); err != nil {
			return nil, err
		}
		if err := teacherRepo.RecalculateTeacherStats(teacherID); err != nil {
			return nil, err
		}
		return relation, nil
	})
}

func (s *AdminService) RemoveCourseRelation(operatorID, courseID, teacherID int64, ip net.IP) error {
	if exists, err := s.courseRepo.CourseExists(courseID); err != nil {
		return err
	} else if !exists {
		return ErrAdminTargetNotFound
	}
	if exists, err := s.teacherRepo.TeacherExists(teacherID); err != nil {
		return err
	} else if !exists {
		return ErrAdminTargetNotFound
	}

	return s.withWriteTx(func(adminRepo repo.AdminRepository, _ repo.CourseRepository, tx *gorm.DB) error {
		relation, err := s.teacherRepoWithTx(tx).GetCourseTeacherRelation(courseID, teacherID)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrAdminTargetNotFound
		}
		if err != nil {
			return err
		}
		if relation.Status != model.CourseTeacherRelationStatusActive {
			return ErrAdminConflict
		}
		oldValues := mustJSON(relation)
		now := time.Now()
		relation.Status = model.CourseTeacherRelationStatusCanceled
		relation.CanceledAt = &now
		if err := s.teacherRepoWithTx(tx).UpdateCourseTeacherRelation(relation); err != nil {
			return err
		}
		if err := adminRepo.CreateAuditLog(&model.AuditLogs{
			OperatorID: operatorID,
			Action:     model.AuditActionDelete,
			TargetType: "course_teacher_relation",
			TargetID:   relation.ID,
			OldValues:  oldValues,
			NewValues:  mustJSON(relation),
			Reason:     "cancel course teacher relation",
			IpAddress:  ip,
		}); err != nil {
			return err
		}
		if err := s.courseRepoWithTx(tx).RecalculateCourseStats(courseID); err != nil {
			return err
		}
		return s.teacherRepoWithTx(tx).RecalculateTeacherStats(teacherID)
	})
}

func (s *AdminService) ListTeachers(status, keyword string, departmentID *int16, page, size int) ([]repo.AdminTeacherItem, int64, error) {
	fillPagination(&page, &size)
	return s.adminRepo.ListTeachers(status, keyword, departmentID, page, size)
}

func (s *AdminService) CreateTeacher(operatorID int64, teacher *model.Teachers, bio, tutorType, homepageURL string, ip net.IP) (*model.Teachers, error) {
	if strings.TrimSpace(teacher.Name) == "" || teacher.DepartmentID <= 0 || !s.departmentExists(s.db, teacher.DepartmentID) {
		return nil, ErrAdminInvalidPayload
	}
	metadata, err := buildTeacherMetadata(teacher.Metadata, teacherMetadataInput{
		Bio:         &bio,
		TutorType:   &tutorType,
		HomepageURL: &homepageURL,
	})
	if err != nil {
		return nil, ErrAdminInvalidPayload
	}
	teacher.Status = "active"
	teacher.Metadata = metadata
	return withWriteTxValue(s, func(adminRepo repo.AdminRepository, _ repo.CourseRepository, tx *gorm.DB) (*model.Teachers, error) {
		if !s.departmentExists(tx, teacher.DepartmentID) {
			return nil, ErrAdminInvalidPayload
		}
		if err := adminRepo.CreateTeacher(teacher); err != nil {
			return nil, err
		}
		if err := adminRepo.CreateAuditLog(&model.AuditLogs{
			OperatorID: operatorID,
			Action:     model.AuditActionApprove,
			TargetType: "teacher",
			TargetID:   teacher.ID,
			NewValues:  mustJSON(teacher),
			Reason:     "create teacher",
			IpAddress:  ip,
		}); err != nil {
			return nil, err
		}
		return teacher, nil
	})
}

func (s *AdminService) UpdateTeacher(operatorID, teacherID int64, updates map[string]interface{}, bio, tutorType, homepageURL *string, ip net.IP) (*model.Teachers, error) {
	if len(updates) == 0 && bio == nil && tutorType == nil && homepageURL == nil {
		return nil, ErrAdminInvalidPayload
	}
	return withWriteTxValue(s, func(adminRepo repo.AdminRepository, _ repo.CourseRepository, tx *gorm.DB) (*model.Teachers, error) {
		item, err := adminRepo.GetTeacherByID(teacherID)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrAdminTargetNotFound
		}
		if err != nil {
			return nil, err
		}
		if departmentID, ok := updates["department_id"].(int16); ok && !s.departmentExists(tx, departmentID) {
			return nil, ErrAdminInvalidPayload
		}
		if bio != nil || tutorType != nil || homepageURL != nil {
			metadata, err := buildTeacherMetadata(item.Metadata, teacherMetadataInput{
				Bio:         bio,
				TutorType:   tutorType,
				HomepageURL: homepageURL,
			})
			if err != nil {
				return nil, ErrAdminInvalidPayload
			}
			updates["metadata"] = metadata
		}
		updates["updated_at"] = time.Now()
		oldValues := mustJSON(item)
		if err := adminRepo.UpdateTeacherFields(teacherID, updates); err != nil {
			return nil, err
		}
		item, err = adminRepo.GetTeacherByID(teacherID)
		if err != nil {
			return nil, err
		}
		if err := adminRepo.CreateAuditLog(&model.AuditLogs{
			OperatorID: operatorID,
			Action:     model.AuditActionApprove,
			TargetType: "teacher",
			TargetID:   teacherID,
			OldValues:  oldValues,
			NewValues:  mustJSON(item),
			Reason:     "update teacher",
			IpAddress:  ip,
		}); err != nil {
			return nil, err
		}
		return item, nil
	})
}

func (s *AdminService) DeleteTeacher(operatorID, teacherID int64, ip net.IP) error {
	return s.withWriteTx(func(adminRepo repo.AdminRepository, _ repo.CourseRepository, _ *gorm.DB) error {
		item, err := adminRepo.GetTeacherByID(teacherID)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrAdminTargetNotFound
		}
		if err != nil {
			return err
		}
		if item.Status == "deleted" {
			return ErrAdminConflict
		}
		oldValues := mustJSON(item)
		if err := adminRepo.UpdateTeacherFields(teacherID, map[string]interface{}{
			"status":     "deleted",
			"updated_at": time.Now(),
		}); err != nil {
			return err
		}
		return adminRepo.CreateAuditLog(&model.AuditLogs{
			OperatorID: operatorID,
			Action:     model.AuditActionDelete,
			TargetType: "teacher",
			TargetID:   teacherID,
			OldValues:  oldValues,
			NewValues:  jsonMap("status", "deleted"),
			Reason:     "delete teacher",
			IpAddress:  ip,
		})
	})
}

func (s *AdminService) ListTeacherRelations(teacherID int64) ([]repo.AdminTeacherRelationItem, error) {
	if exists, err := s.teacherRepo.TeacherExists(teacherID); err != nil {
		return nil, err
	} else if !exists {
		return nil, ErrAdminTargetNotFound
	}
	return s.adminRepo.ListTeacherRelations(teacherID)
}

func (s *AdminService) ListResources(status, keyword, resourceType string, courseID int64, page, size int) ([]repo.AdminResourceItem, int64, error) {
	fillPagination(&page, &size)
	return s.adminRepo.ListResources(status, keyword, resourceType, courseID, page, size)
}

func (s *AdminService) DeleteResource(operatorID, resourceID int64, reason string, ip net.IP) error {
	reason = strings.TrimSpace(reason)
	if reason == "" {
		return ErrAdminInvalidPayload
	}

	payload, err := s.resourceRepo.GetResourceDeletePayload(resourceID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return ErrAdminTargetNotFound
	}
	if err != nil {
		return err
	}

	err = s.withWriteTx(func(adminRepo repo.AdminRepository, courseRepo repo.CourseRepository, _ *gorm.DB) error {
		item, err := adminRepo.GetResourceByID(resourceID)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrAdminTargetNotFound
		}
		if err != nil {
			return err
		}
		if item.Status == model.ResourceStatusDeleted {
			return ErrAdminConflict
		}
		oldValues := mustJSON(item)
		if err := adminRepo.UpdateResourceStatus(resourceID, model.ResourceStatusDeleted); err != nil {
			return err
		}
		if err := courseRepo.RecalculateCourseStats(item.CourseID); err != nil {
			return err
		}
		if err := adminRepo.CreateAuditLog(&model.AuditLogs{
			OperatorID: operatorID,
			Action:     model.AuditActionDelete,
			TargetType: "resource",
			TargetID:   resourceID,
			OldValues:  oldValues,
			NewValues:  jsonMap("status", model.ResourceStatusDeleted),
			Reason:     reason,
			IpAddress:  ip,
		}); err != nil {
			return err
		}

		notification := buildResourceDeletedNotification(item.UploaderID, item.ID, item.Title, reason)
		if notification != nil {
			return adminRepo.CreateNotification(notification)
		}

		return nil
	})
	if err != nil {
		return err
	}

	for _, fileKey := range payload.FileKeys {
		if err := utils.TencentCosDeleteObject(fileKey); err != nil {
			logger.Log.Error("管理员删除资源后清理 COS 文件失败，资源记录已软删除，等待后续补偿清理",
				zap.Int64("resource_id", resourceID),
				zap.String("file_key", fileKey),
				zap.Error(err))
			break
		}
	}

	return nil
}

func (s *AdminService) ListAuditLogs(action string, operatorID int64, targetType string, page, size int) ([]repo.AdminAuditLogItem, int64, error) {
	fillPagination(&page, &size)
	return s.adminRepo.ListAuditLogs(action, operatorID, targetType, page, size)
}

func (s *AdminService) applyCorrection(adminRepo repo.AdminRepository, db *gorm.DB, correction *model.Corrections) error {
	switch correction.TargetType {
	case model.CorrectionTargetTypeCourse:
		return s.applyCourseCorrection(adminRepo, correction)
	case model.CorrectionTargetTypeTeacher:
		return s.applyTeacherCorrection(adminRepo, db, correction)
	default:
		return ErrAdminInvalidPayload
	}
}

func (s *AdminService) applyCourseCorrection(adminRepo repo.AdminRepository, correction *model.Corrections) error {
	item, err := adminRepo.GetCourseByID(correction.TargetID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return ErrAdminTargetNotFound
	}
	if err != nil {
		return err
	}
	updates := map[string]interface{}{"updated_at": time.Now()}
	switch strings.TrimSpace(correction.Field) {
	case "name":
		updates["name"] = correction.SuggestedValue
	case "description":
		updates["description"] = correction.SuggestedValue
	case "course_type":
		normalized := normalizeAdminCourseType(correction.SuggestedValue)
		if normalized == "" {
			return ErrAdminInvalidPayload
		}
		updates["course_type"] = normalized
	default:
		return ErrAdminInvalidPayload
	}
	_ = item
	return adminRepo.UpdateCourseFields(correction.TargetID, updates)
}

func (s *AdminService) applyTeacherCorrection(adminRepo repo.AdminRepository, db *gorm.DB, correction *model.Corrections) error {
	item, err := adminRepo.GetTeacherByID(correction.TargetID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return ErrAdminTargetNotFound
	}
	if err != nil {
		return err
	}
	updates := map[string]interface{}{"updated_at": time.Now()}
	switch strings.TrimSpace(correction.Field) {
	case "name":
		updates["name"] = correction.SuggestedValue
	case "title":
		updates["title"] = correction.SuggestedValue
	case "avatar_url":
		updates["avatar_url"] = correction.SuggestedValue
	case "department_id":
		departmentID, ok := parseInt16String(correction.SuggestedValue)
		if !ok || !s.departmentExists(db, departmentID) {
			return ErrAdminInvalidPayload
		}
		updates["department_id"] = departmentID
	case "bio":
		metadata, err := buildTeacherMetadata(item.Metadata, teacherMetadataInput{Bio: &correction.SuggestedValue})
		if err != nil {
			return ErrAdminInvalidPayload
		}
		updates["metadata"] = metadata
	case "tutor_type":
		metadata, err := buildTeacherMetadata(item.Metadata, teacherMetadataInput{TutorType: &correction.SuggestedValue})
		if err != nil {
			return ErrAdminInvalidPayload
		}
		updates["metadata"] = metadata
	case "homepage_url":
		metadata, err := buildTeacherMetadata(item.Metadata, teacherMetadataInput{HomepageURL: &correction.SuggestedValue})
		if err != nil {
			return ErrAdminInvalidPayload
		}
		updates["metadata"] = metadata
	default:
		return ErrAdminInvalidPayload
	}
	return adminRepo.UpdateTeacherFields(correction.TargetID, updates)
}

func (s *AdminService) departmentExists(db *gorm.DB, id int16) bool {
	if db == nil {
		return true
	}
	var count int64
	if err := db.Model(&model.Departments{}).Where("id = ?", id).Count(&count).Error; err != nil {
		return false
	}
	return count > 0
}

func applyAnnouncementUpdates(item *model.Announcements, updates map[string]interface{}) {
	if value, ok := updates["title"].(string); ok {
		item.Title = value
	}
	if value, ok := updates["content"].(string); ok {
		item.Content = value
	}
	if value, ok := updates["type"].(model.AnnouncementType); ok {
		item.Type = value
	}
	if value, ok := updates["is_pinned"].(bool); ok {
		item.IsPinned = value
	}
	if value, ok := updates["is_published"].(bool); ok {
		item.IsPublished = value
	}
	if value, ok := updates["published_at"].(time.Time); ok {
		item.PublishedAt = value
	}
	if value, ok := updates["expires_at"].(time.Time); ok {
		item.ExpiresAt = value
	}
}

func shouldBroadcastAnnouncementUpdate(wasPublished bool, oldTitle, oldContent string, item *model.Announcements) bool {
	if item == nil || !item.IsPublished {
		return false
	}
	if !wasPublished {
		return true
	}
	return strings.TrimSpace(oldTitle) != strings.TrimSpace(item.Title) ||
		strings.TrimSpace(oldContent) != strings.TrimSpace(item.Content)
}

type teacherMetadataInput struct {
	Bio         *string
	TutorType   *string
	HomepageURL *string
}

func buildTeacherMetadata(raw datatypes.JSON, input teacherMetadataInput) (datatypes.JSON, error) {
	payload := map[string]interface{}{}
	if len(raw) > 0 {
		_ = json.Unmarshal(raw, &payload)
	}
	if input.Bio != nil {
		if value := strings.TrimSpace(*input.Bio); value == "" {
			delete(payload, "bio")
		} else {
			payload["bio"] = value
		}
	}
	if input.TutorType != nil {
		if value := strings.TrimSpace(*input.TutorType); value == "" {
			delete(payload, "tutor_type")
		} else {
			payload["tutor_type"] = value
		}
	}
	if input.HomepageURL != nil {
		value, err := normalizeTeacherHomepageURL(*input.HomepageURL)
		if err != nil {
			return nil, err
		}
		if value == "" {
			delete(payload, "homepage_url")
		} else {
			payload["homepage_url"] = value
		}
	}
	data, _ := json.Marshal(payload)
	return datatypes.JSON(data), nil
}

func normalizeTeacherHomepageURL(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", nil
	}
	parsed, err := url.Parse(value)
	if err != nil || parsed.Host == "" {
		return "", errors.New("invalid homepage url")
	}
	switch parsed.Scheme {
	case "http", "https":
		return value, nil
	default:
		return "", errors.New("invalid homepage url")
	}
}

func parseInt16String(value string) (int16, bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0, false
	}
	var parsed int
	_, err := fmt.Sscanf(value, "%d", &parsed)
	if err != nil || parsed <= 0 || parsed > 32767 {
		return 0, false
	}
	return int16(parsed), true
}

func normalizeAdminCourseType(value string) model.CourseType {
	switch strings.TrimSpace(value) {
	case "public", "公选课":
		return model.CourseTypePublic
	case "non_public", "非公选课":
		return model.CourseTypeNonPublic
	default:
		return ""
	}
}

func jsonMap(kv ...interface{}) datatypes.JSON {
	payload := map[string]interface{}{}
	for i := 0; i+1 < len(kv); i += 2 {
		key, ok := kv[i].(string)
		if !ok {
			continue
		}
		payload[key] = kv[i+1]
	}
	return mustJSON(payload)
}

func (s *AdminService) withWriteTx(fn func(repo.AdminRepository, repo.CourseRepository, *gorm.DB) error) error {
	if s.db == nil {
		return fn(s.adminRepo, s.courseRepo, nil)
	}
	return s.db.Transaction(func(tx *gorm.DB) error {
		return fn(s.adminRepoWithTx(tx), s.courseRepoWithTx(tx), tx)
	})
}

func (s *AdminService) adminRepoWithTx(tx *gorm.DB) repo.AdminRepository {
	withTx, ok := s.adminRepo.(interface {
		WithTx(*gorm.DB) repo.AdminRepository
	})
	if !ok {
		return s.adminRepo
	}
	return withTx.WithTx(tx)
}

func withWriteTxValue[T any](s *AdminService, fn func(repo.AdminRepository, repo.CourseRepository, *gorm.DB) (T, error)) (T, error) {
	var result T
	if s.db == nil {
		return fn(s.adminRepo, s.courseRepo, nil)
	}
	err := s.db.Transaction(func(tx *gorm.DB) error {
		value, err := fn(s.adminRepoWithTx(tx), s.courseRepoWithTx(tx), tx)
		if err != nil {
			return err
		}
		result = value
		return nil
	})
	return result, err
}

func (s *AdminService) courseRepoWithTx(tx *gorm.DB) repo.CourseRepository {
	if tx == nil {
		return s.courseRepo
	}
	withTx, ok := s.courseRepo.(interface {
		WithTx(*gorm.DB) repo.CourseRepository
	})
	if !ok {
		return s.courseRepo
	}
	return withTx.WithTx(tx)
}

func (s *AdminService) teacherRepoWithTx(tx *gorm.DB) repo.TeacherRepository {
	if tx == nil {
		return s.teacherRepo
	}
	withTx, ok := s.teacherRepo.(interface {
		WithTx(*gorm.DB) repo.TeacherRepository
	})
	if !ok {
		return s.teacherRepo
	}
	return withTx.WithTx(tx)
}

func (s *AdminService) commentRepoWithTx(tx *gorm.DB) repo.CommentRepository {
	if tx == nil {
		return s.commentRepo
	}
	withTx, ok := s.commentRepo.(interface {
		WithTx(*gorm.DB) repo.CommentRepository
	})
	if !ok {
		return s.commentRepo
	}
	return withTx.WithTx(tx)
}

func (s *AdminService) socialRepoWithTx(tx *gorm.DB) repo.SocialRepository {
	if tx == nil {
		return s.socialRepo
	}
	withTx, ok := s.socialRepo.(interface {
		WithTx(*gorm.DB) repo.SocialRepository
	})
	if !ok {
		return s.socialRepo
	}
	return withTx.WithTx(tx)
}

func (s *AdminService) resourceRepoWithTx(tx *gorm.DB) repo.ResourceRepository {
	if tx == nil {
		return s.resourceRepo
	}
	withTx, ok := s.resourceRepo.(interface {
		WithTx(*gorm.DB) repo.ResourceRepository
	})
	if !ok {
		return s.resourceRepo
	}
	return withTx.WithTx(tx)
}
