package handler

import (
	"csu-star-backend/internal/req"
	"errors"
	"testing"
)

func TestBuildAnnouncementModelRejectsInvalidTime(t *testing.T) {
	input := req.AdminAnnouncementInput{
		Title:       "公告",
		Content:     "内容",
		PublishedAt: stringPtr("not-a-time"),
	}

	_, err := buildAnnouncementModel(input)
	if !errors.Is(err, errInvalidAdminTime) {
		t.Fatalf("expected invalid admin time error, got %v", err)
	}
}

func TestBuildAnnouncementUpdatesRejectsInvalidTime(t *testing.T) {
	input := req.AdminAnnouncementInput{
		ExpiresAt: stringPtr("bad-time"),
	}

	_, err := buildAnnouncementUpdates(input)
	if !errors.Is(err, errInvalidAdminTime) {
		t.Fatalf("expected invalid admin time error, got %v", err)
	}
}

func TestBuildAnnouncementModelParsesRFC3339Time(t *testing.T) {
	input := req.AdminAnnouncementInput{
		Title:       "公告",
		Content:     "内容",
		PublishedAt: stringPtr("2026-04-05T10:00:00Z"),
	}

	item, err := buildAnnouncementModel(input)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if got := item.PublishedAt.Format("2006-01-02T15:04:05Z07:00"); got != "2026-04-05T10:00:00Z" {
		t.Fatalf("unexpected published_at: %s", got)
	}
}

func stringPtr(value string) *string {
	return &value
}
