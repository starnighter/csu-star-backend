package service

import (
	"csu-star-backend/internal/model"
	"testing"
	"time"
)

func TestViolationBanDurationProgression(t *testing.T) {
	tests := []struct {
		level int
		want  time.Duration
	}{
		{level: 1, want: 24 * time.Hour},
		{level: 2, want: 7 * 24 * time.Hour},
		{level: 3, want: 0},
		{level: 4, want: 0},
	}

	for _, tt := range tests {
		if got := violationBanDuration(tt.level); got != tt.want {
			t.Fatalf("violationBanDuration(%d) = %v, want %v", tt.level, got, tt.want)
		}
	}
}

func TestBuildRiskDataIncludesBanDetails(t *testing.T) {
	now := time.Date(2026, 4, 6, 10, 0, 0, 0, time.UTC)
	decision := &BanDecision{
		Banned:         true,
		BanUntil:       &now,
		BanReason:      "频繁创建上传会话",
		BanSource:      model.UserBanSourceSystem,
		ViolationCount: 2,
		Permanent:      false,
	}

	data := BuildRiskData(decision)
	if data["ban_reason"] != "频繁创建上传会话" {
		t.Fatalf("expected ban reason to be included, got %v", data["ban_reason"])
	}
	if data["ban_source"] != model.UserBanSourceSystem {
		t.Fatalf("expected ban source to be included, got %v", data["ban_source"])
	}
	if data["violation_count"] != 2 {
		t.Fatalf("expected violation count 2, got %v", data["violation_count"])
	}
	if data["ban_until"] != now.Format(time.RFC3339) {
		t.Fatalf("expected formatted ban until, got %v", data["ban_until"])
	}
}

func TestBanDecisionFromUserMarksPermanentBan(t *testing.T) {
	user := &model.Users{
		Status:         model.UserStatusBanned,
		BanReason:      "第三次违规",
		BanSource:      model.UserBanSourceSystem,
		ViolationCount: 3,
	}

	decision := BanDecisionFromUser(user)
	if decision == nil {
		t.Fatal("expected ban decision")
	}
	if !decision.Permanent {
		t.Fatal("expected permanent ban")
	}
	if decision.BanReason != "第三次违规" {
		t.Fatalf("unexpected ban reason: %s", decision.BanReason)
	}
}
