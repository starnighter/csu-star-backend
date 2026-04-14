package service

import (
	"csu-star-backend/internal/model"
	"encoding/json"
	"strings"

	"gorm.io/datatypes"
)

type interactionRoute struct {
	targetPage   string
	targetID     int64
	evaluationID int64
	commentID    int64
	replyID      int64
}

func buildAnnouncementNotification(item model.Announcements, actorUserID int64) *model.Notifications {
	return &model.Notifications{
		UserID:    actorUserID,
		Type:      model.NotificationTypeSystem,
		Category:  model.NotificationCategoryAnnouncement,
		Result:    model.NotificationResultInform,
		Title:     strings.TrimSpace(item.Title),
		Content:   strings.TrimSpace(item.Content),
		RelatedID: item.ID,
		IsGlobal:  true,
		Metadata:  mustJSON(map[string]interface{}{"announcement_type": item.Type}),
	}
}

func buildReportHandledNotification(report *model.Reports) *model.Notifications {
	result := model.NotificationResultApproved
	title := "举报处理通过"
	content := "你提交的举报已处理并确认有效。"
	if report.Status == model.ReportStatusDismissed {
		result = model.NotificationResultRejected
		title = "举报处理未通过"
		content = "你提交的举报已处理，但未被采纳。"
	}
	if note := strings.TrimSpace(report.ProcessNote); note != "" {
		content += " 备注：" + note
	}
	return &model.Notifications{
		UserID:    report.UserID,
		Type:      model.NotificationTypeSystem,
		Category:  model.NotificationCategoryReport,
		Result:    result,
		Title:     title,
		Content:   content,
		RelatedID: report.ID,
	}
}

func buildReportedContentHandledNotification(report *model.Reports, ownerID int64, processNote string) *model.Notifications {
	if report == nil || ownerID <= 0 {
		return nil
	}

	title := "你的内容因举报已被处理"
	content := "你发布的" + reportTargetLabel(report.TargetType) + "因举报核查已被处理。"
	if note := strings.TrimSpace(processNote); note != "" {
		content += " 备注：" + note
	}

	return &model.Notifications{
		UserID:    ownerID,
		Type:      model.NotificationTypeSystem,
		Category:  model.NotificationCategoryReport,
		Result:    model.NotificationResultRejected,
		Title:     title,
		Content:   content,
		RelatedID: report.ID,
		Metadata: mustJSON(map[string]interface{}{
			"source_type": string(report.TargetType),
			"source_id":   report.TargetID,
			"report_id":   report.ID,
		}),
	}
}

func buildCorrectionHandledNotification(correction *model.Corrections) *model.Notifications {
	result := model.NotificationResultApproved
	title := "纠错处理通过"
	content := "你提交的纠错已处理并采纳。"
	if correction.Status == model.CorrectionStatusRejected {
		result = model.NotificationResultRejected
		title = "纠错处理未通过"
		content = "你提交的纠错已处理，但未被采纳。"
	}
	if note := strings.TrimSpace(correction.ProcessNote); note != "" {
		content += " 备注：" + note
	}
	return &model.Notifications{
		UserID:    correction.UserID,
		Type:      model.NotificationTypeSystem,
		Category:  model.NotificationCategoryCorrection,
		Result:    result,
		Title:     title,
		Content:   content,
		RelatedID: correction.ID,
	}
}

func buildFeedbackReplyNotification(item *model.Feedbacks) *model.Notifications {
	result := model.NotificationResultInform
	title := "反馈有新回复"
	content := strings.TrimSpace(item.Reply)
	switch item.Status {
	case model.FeedbackStatusResolved:
		result = model.NotificationResultApproved
		title = "反馈处理完成"
	case model.FeedbackStatusClosed:
		result = model.NotificationResultRejected
		title = "反馈已关闭"
	}
	if content == "" {
		content = "你的反馈有新的处理结果。"
	}
	return &model.Notifications{
		UserID:    item.UserID,
		Type:      model.NotificationTypeSystem,
		Category:  model.NotificationCategoryFeedback,
		Result:    result,
		Title:     title,
		Content:   content,
		RelatedID: item.ID,
	}
}

func buildTeacherSupplementReviewNotification(userID int64, request *model.TeacherSupplementRequests, approved bool, content string) *model.Notifications {
	result := model.NotificationResultRejected
	title := "教师补录申请未通过"
	if approved {
		result = model.NotificationResultApproved
		title = "教师补录申请已通过"
	}
	return &model.Notifications{
		UserID:    userID,
		Type:      model.NotificationTypeSystem,
		Category:  model.NotificationCategorySupplement,
		Result:    result,
		Title:     title,
		Content:   strings.TrimSpace(content),
		RelatedID: request.ID,
	}
}

func buildCourseSupplementReviewNotification(userID int64, request *model.CourseSupplementRequests, approved bool, content string) *model.Notifications {
	result := model.NotificationResultRejected
	title := "课程补录申请未通过"
	if approved {
		result = model.NotificationResultApproved
		title = "课程补录申请已通过"
	}
	return &model.Notifications{
		UserID:    userID,
		Type:      model.NotificationTypeSystem,
		Category:  model.NotificationCategorySupplement,
		Result:    result,
		Title:     title,
		Content:   strings.TrimSpace(content),
		RelatedID: request.ID,
	}
}

func buildAdminDirectNotification(userID int64, title, content string, result model.NotificationResult) *model.Notifications {
	if result == "" {
		result = model.NotificationResultInform
	}
	return &model.Notifications{
		UserID:    userID,
		Type:      model.NotificationTypeSystem,
		Category:  model.NotificationCategoryAdminMessage,
		Result:    result,
		Title:     strings.TrimSpace(title),
		Content:   strings.TrimSpace(content),
		RelatedID: 0,
	}
}

func buildResourceDeletedNotification(userID, resourceID int64, resourceTitle, reason string) *model.Notifications {
	if userID <= 0 {
		return nil
	}

	title := "你上传的资源已被删除"
	content := "你上传的资源"
	if trimmedTitle := strings.TrimSpace(resourceTitle); trimmedTitle != "" {
		content += "《" + trimmedTitle + "》"
	}
	content += "因审核处理已被删除。"
	if trimmedReason := strings.TrimSpace(reason); trimmedReason != "" {
		content += " 原因：" + trimmedReason
	}

	return &model.Notifications{
		UserID:    userID,
		Type:      model.NotificationTypeSystem,
		Category:  model.NotificationCategoryAdminMessage,
		Result:    model.NotificationResultRejected,
		Title:     title,
		Content:   content,
		RelatedID: resourceID,
		Metadata: mustJSON(map[string]interface{}{
			"source_type": "resource",
			"source_id":   resourceID,
		}),
	}
}

func buildPointsChangedNotification(userID int64, reason string) *model.Notifications {
	return &model.Notifications{
		UserID:    userID,
		Type:      model.NotificationTypeSystem,
		Category:  model.NotificationCategoryPoints,
		Result:    model.NotificationResultInform,
		Title:     "积分已调整",
		Content:   strings.TrimSpace(reason),
		RelatedID: 0,
	}
}

func mustJSON(value interface{}) datatypes.JSON {
	data, err := json.Marshal(value)
	if err != nil {
		return datatypes.JSON([]byte("{}"))
	}
	return datatypes.JSON(data)
}

func buildReadPointsNotification(userID int64, title, content string, relatedID int64) *model.Notifications {
	return &model.Notifications{
		UserID:    userID,
		Type:      model.NotificationTypeSystem,
		Category:  model.NotificationCategoryPoints,
		Result:    model.NotificationResultInform,
		Title:     strings.TrimSpace(title),
		Content:   strings.TrimSpace(content),
		RelatedID: relatedID,
		IsRead:    true,
	}
}

func buildResourceFavoriteNotification(recipientID, resourceID int64) *model.Notifications {
	if recipientID <= 0 || resourceID <= 0 {
		return nil
	}

	return &model.Notifications{
		UserID:    recipientID,
		Type:      model.NotificationTypeLiked,
		Category:  model.NotificationCategoryInteraction,
		Result:    model.NotificationResultInform,
		Title:     "收到新的收藏",
		Content:   "你的资源被收藏了。",
		RelatedID: resourceID,
		IsRead:    false,
		IsGlobal:  false,
		Metadata:  buildInteractionMetadata("resource", resourceID, buildResourceInteractionRoute(resourceID)),
	}
}

func buildInteractionMetadata(sourceType string, sourceID int64, route *interactionRoute) datatypes.JSON {
	payload := map[string]interface{}{}
	if strings.TrimSpace(sourceType) == "" && sourceID <= 0 {
		if route == nil {
			return datatypes.JSON([]byte("{}"))
		}
	} else {
		payload["source_type"] = strings.TrimSpace(sourceType)
		payload["source_id"] = sourceID
	}
	if route != nil {
		if route.targetPage != "" {
			payload["target_page"] = route.targetPage
		}
		if route.targetID > 0 {
			payload["target_id"] = route.targetID
		}
		if route.evaluationID > 0 {
			payload["evaluation_id"] = route.evaluationID
		}
		if route.commentID > 0 {
			payload["comment_id"] = route.commentID
		}
		if route.replyID > 0 {
			payload["reply_id"] = route.replyID
		}
	}
	if len(payload) == 0 {
		return datatypes.JSON([]byte("{}"))
	}
	raw, _ := json.Marshal(payload)
	return datatypes.JSON(raw)
}

func buildEvaluationReplyNotification(
	recipientID int64,
	targetType model.LikeTargetType,
	targetID int64,
	route *interactionRoute,
) *model.Notifications {
	if recipientID <= 0 {
		return nil
	}

	return &model.Notifications{
		UserID:    recipientID,
		Type:      model.NotificationTypeCommented,
		Category:  model.NotificationCategoryInteraction,
		Result:    model.NotificationResultInform,
		Title:     "收到新的回复",
		Content:   buildEvaluationReplyNotificationContent(targetType),
		RelatedID: targetID,
		IsRead:    false,
		IsGlobal:  false,
		Metadata:  buildInteractionMetadata(string(targetType), targetID, route),
	}
}

func buildResourceInteractionRoute(resourceID int64) *interactionRoute {
	if resourceID <= 0 {
		return nil
	}
	return &interactionRoute{
		targetPage: "resource",
		targetID:   resourceID,
	}
}

func buildTeacherEvaluationInteractionRoute(teacherID, evaluationID, replyID int64) *interactionRoute {
	if teacherID <= 0 {
		return nil
	}
	return &interactionRoute{
		targetPage:   "teacher",
		targetID:     teacherID,
		evaluationID: evaluationID,
		replyID:      replyID,
	}
}

func buildCourseEvaluationInteractionRoute(courseID, evaluationID, replyID int64) *interactionRoute {
	if courseID <= 0 {
		return nil
	}
	return &interactionRoute{
		targetPage:   "course",
		targetID:     courseID,
		evaluationID: evaluationID,
		replyID:      replyID,
	}
}

func buildEvaluationReplyNotificationContent(targetType model.LikeTargetType) string {
	switch targetType {
	case model.LikeTargetTypeTeacherEvaluation:
		return "你的教师评价收到了新的回复。"
	case model.LikeTargetTypeCourseEvaluation:
		return "你的课程评价收到了新的回复。"
	case model.LikeTargetTypeTeacherReply:
		return "你的教师评价回复收到了新的回复。"
	case model.LikeTargetTypeCourseReply:
		return "你的课程评价回复收到了新的回复。"
	default:
		return "你的内容收到了新的回复。"
	}
}

func reportTargetLabel(targetType model.ReportTargetType) string {
	switch targetType {
	case model.ReportTargetTypeResource:
		return "资源"
	case model.ReportTargetTypeCourse:
		return "课程"
	case model.ReportTargetTypeTeacherEvaluation, model.ReportTargetTypeCourseEvaluation:
		return "评价"
	case model.ReportTargetTypeTeacherReply, model.ReportTargetTypeCourseReply:
		return "回复"
	case model.ReportTargetTypeComment:
		return "评论"
	default:
		return "内容"
	}
}
