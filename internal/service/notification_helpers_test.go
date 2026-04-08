package service

import (
	"csu-star-backend/internal/model"
	"testing"
)

func TestBuildReportedContentHandledNotification(t *testing.T) {
	report := &model.Reports{
		ID:         11,
		UserID:     1,
		TargetType: model.ReportTargetTypeComment,
		TargetID:   22,
	}

	notification := buildReportedContentHandledNotification(report, 9, "包含辱骂内容")
	if notification == nil {
		t.Fatalf("expected notification to be created")
	}
	if notification.UserID != 9 {
		t.Fatalf("expected recipient 9, got %d", notification.UserID)
	}
	if notification.Category != model.NotificationCategoryReport {
		t.Fatalf("expected report category, got %s", notification.Category)
	}
	if notification.Result != model.NotificationResultRejected {
		t.Fatalf("expected rejected result, got %s", notification.Result)
	}
	if notification.Title != "你的内容因举报已被处理" {
		t.Fatalf("unexpected title: %s", notification.Title)
	}
	if notification.RelatedID != report.ID {
		t.Fatalf("expected related id %d, got %d", report.ID, notification.RelatedID)
	}
	if string(notification.Metadata) == "" || string(notification.Metadata) == "{}" {
		t.Fatalf("expected metadata to contain source information")
	}
}
