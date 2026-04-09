package service

import (
	"csu-star-backend/internal/model"
	"encoding/json"
	"testing"
)

func TestBuildEvaluationReplyNotificationForEvaluation(t *testing.T) {
	item := buildEvaluationReplyNotification(
		5,
		model.LikeTargetTypeTeacherEvaluation,
		12,
		buildTeacherEvaluationInteractionRoute(21, 12, 34),
	)
	if item == nil {
		t.Fatalf("expected notification to be created")
	}
	if item.UserID != 5 {
		t.Fatalf("expected recipient 5, got %d", item.UserID)
	}
	if item.Type != model.NotificationTypeCommented {
		t.Fatalf("expected commented notification type, got %s", item.Type)
	}
	if item.Category != model.NotificationCategoryInteraction {
		t.Fatalf("expected interaction category, got %s", item.Category)
	}
	if item.Content != "你的教师评价收到了新的回复。" {
		t.Fatalf("unexpected content: %s", item.Content)
	}
	metadata := decodeNotificationMetadata(t, item.Metadata)
	if metadata["target_page"] != "teacher" {
		t.Fatalf("expected target_page teacher, got %v", metadata["target_page"])
	}
	if metadata["target_id"] != float64(21) {
		t.Fatalf("expected target_id 21, got %v", metadata["target_id"])
	}
	if metadata["evaluation_id"] != float64(12) {
		t.Fatalf("expected evaluation_id 12, got %v", metadata["evaluation_id"])
	}
	if metadata["reply_id"] != float64(34) {
		t.Fatalf("expected reply_id 34, got %v", metadata["reply_id"])
	}
}

func TestBuildEvaluationReplyNotificationForReply(t *testing.T) {
	item := buildEvaluationReplyNotification(
		7,
		model.LikeTargetTypeCourseReply,
		18,
		buildCourseEvaluationInteractionRoute(25, 16, 18),
	)
	if item == nil {
		t.Fatalf("expected notification to be created")
	}
	if item.Content != "你的课程评价回复收到了新的回复。" {
		t.Fatalf("unexpected content: %s", item.Content)
	}
	if item.RelatedID != 18 {
		t.Fatalf("expected related id 18, got %d", item.RelatedID)
	}
	metadata := decodeNotificationMetadata(t, item.Metadata)
	if metadata["target_page"] != "course" {
		t.Fatalf("expected target_page course, got %v", metadata["target_page"])
	}
	if metadata["target_id"] != float64(25) {
		t.Fatalf("expected target_id 25, got %v", metadata["target_id"])
	}
	if metadata["evaluation_id"] != float64(16) {
		t.Fatalf("expected evaluation_id 16, got %v", metadata["evaluation_id"])
	}
	if metadata["reply_id"] != float64(18) {
		t.Fatalf("expected reply_id 18, got %v", metadata["reply_id"])
	}
}

func TestBuildResourceFavoriteNotification(t *testing.T) {
	item := buildResourceFavoriteNotification(9, 42)
	if item == nil {
		t.Fatalf("expected favorite notification to be created")
	}
	if item.UserID != 9 {
		t.Fatalf("expected recipient 9, got %d", item.UserID)
	}
	if item.Type != model.NotificationTypeLiked {
		t.Fatalf("expected liked notification type, got %s", item.Type)
	}
	if item.Title != "收到新的收藏" {
		t.Fatalf("unexpected title: %s", item.Title)
	}
	if item.Content != "你的资源被收藏了。" {
		t.Fatalf("unexpected content: %s", item.Content)
	}
	if item.RelatedID != 42 {
		t.Fatalf("expected related id 42, got %d", item.RelatedID)
	}
	metadata := decodeNotificationMetadata(t, item.Metadata)
	if metadata["target_page"] != "resource" {
		t.Fatalf("expected target_page resource, got %v", metadata["target_page"])
	}
	if metadata["target_id"] != float64(42) {
		t.Fatalf("expected target_id 42, got %v", metadata["target_id"])
	}
}

func decodeNotificationMetadata(t *testing.T, metadata []byte) map[string]interface{} {
	t.Helper()
	var payload map[string]interface{}
	if err := json.Unmarshal(metadata, &payload); err != nil {
		t.Fatalf("failed to decode metadata: %v", err)
	}
	return payload
}
