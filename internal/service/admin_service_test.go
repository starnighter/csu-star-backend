package service

import (
	"csu-star-backend/internal/model"
	"errors"
	"testing"
)

func TestUpdateAnnouncementRejectsEmptyUpdates(t *testing.T) {
	svc := &AdminService{}

	_, err := svc.UpdateAnnouncement(1, 1, nil, nil)
	if !errors.Is(err, ErrAdminInvalidPayload) {
		t.Fatalf("expected ErrAdminInvalidPayload, got %v", err)
	}
}

func TestUpdateCourseRejectsEmptyUpdates(t *testing.T) {
	svc := &AdminService{}

	_, err := svc.UpdateCourse(1, 1, nil, nil)
	if !errors.Is(err, ErrAdminInvalidPayload) {
		t.Fatalf("expected ErrAdminInvalidPayload, got %v", err)
	}
}

func TestUpdateTeacherRejectsEmptyUpdates(t *testing.T) {
	svc := &AdminService{}

	_, err := svc.UpdateTeacher(1, 1, nil, nil, nil, nil, nil)
	if !errors.Is(err, ErrAdminInvalidPayload) {
		t.Fatalf("expected ErrAdminInvalidPayload, got %v", err)
	}
}

func TestShouldBroadcastAnnouncementUpdate(t *testing.T) {
	item := &model.Announcements{
		Title:       "新公告",
		Content:     "新的内容",
		IsPublished: true,
	}

	if !shouldBroadcastAnnouncementUpdate(false, "", "", item) {
		t.Fatalf("expected newly published announcement to broadcast")
	}

	if !shouldBroadcastAnnouncementUpdate(true, "旧公告", "新的内容", item) {
		t.Fatalf("expected title change to broadcast")
	}

	if !shouldBroadcastAnnouncementUpdate(true, "新公告", "旧内容", item) {
		t.Fatalf("expected content change to broadcast")
	}

	if shouldBroadcastAnnouncementUpdate(true, "新公告", "新的内容", item) {
		t.Fatalf("did not expect unchanged published announcement to broadcast")
	}

	item.IsPublished = false
	if shouldBroadcastAnnouncementUpdate(true, "新公告", "新的内容", item) {
		t.Fatalf("did not expect unpublished announcement to broadcast")
	}
}
