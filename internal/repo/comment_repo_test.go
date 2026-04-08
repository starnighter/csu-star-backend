package repo

import (
	"csu-star-backend/internal/model"
	"encoding/json"
	"testing"
)

func TestBuildResourceCommentNotificationMetadataForComment(t *testing.T) {
	metadata := buildResourceCommentNotificationMetadata(&model.Comments{
		ID:         11,
		TargetType: model.CommentTargetTypeResource,
		TargetID:   22,
	})

	payload := decodeCommentNotificationMetadata(t, metadata)
	if payload["target_page"] != "resource" {
		t.Fatalf("expected target_page resource, got %v", payload["target_page"])
	}
	if payload["target_id"] != float64(22) {
		t.Fatalf("expected target_id 22, got %v", payload["target_id"])
	}
	if payload["comment_id"] != float64(11) {
		t.Fatalf("expected comment_id 11, got %v", payload["comment_id"])
	}
	if _, exists := payload["reply_id"]; exists {
		t.Fatalf("expected no reply_id for top-level comment, got %v", payload["reply_id"])
	}
}

func TestBuildResourceCommentNotificationMetadataForReply(t *testing.T) {
	parentID := int64(11)
	metadata := buildResourceCommentNotificationMetadata(&model.Comments{
		ID:         33,
		TargetType: model.CommentTargetTypeResource,
		TargetID:   22,
		ParentID:   &parentID,
	})

	payload := decodeCommentNotificationMetadata(t, metadata)
	if payload["comment_id"] != float64(11) {
		t.Fatalf("expected comment_id 11, got %v", payload["comment_id"])
	}
	if payload["reply_id"] != float64(33) {
		t.Fatalf("expected reply_id 33, got %v", payload["reply_id"])
	}
}

func decodeCommentNotificationMetadata(t *testing.T, metadata []byte) map[string]interface{} {
	t.Helper()
	var payload map[string]interface{}
	if err := json.Unmarshal(metadata, &payload); err != nil {
		t.Fatalf("failed to decode metadata: %v", err)
	}
	return payload
}
