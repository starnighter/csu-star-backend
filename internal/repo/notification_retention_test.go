package repo

import (
	"testing"
	"time"
)

func TestNotificationRetentionCutoff(t *testing.T) {
	now := time.Date(2026, 4, 6, 12, 0, 0, 0, time.UTC)
	got := NotificationRetentionCutoff(now)
	want := now.Add(-30 * 24 * time.Hour)
	if !got.Equal(want) {
		t.Fatalf("expected cutoff %s, got %s", want, got)
	}
}
