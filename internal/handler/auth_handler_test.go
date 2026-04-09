package handler

import (
	"csu-star-backend/internal/model"
	"errors"
	"testing"
)

func TestBuildLoginRespReturnsErrorWhenTokenRemainingFails(t *testing.T) {
	previous := getTokenRemainingTime
	getTokenRemainingTime = func(string) (int64, error) {
		return 0, errors.New("token parse failed")
	}
	defer func() {
		getTokenRemainingTime = previous
	}()

	_, err := buildLoginResp(&model.Users{
		ID:                1,
		Nickname:          "tester",
		AvatarUrl:         "avatar",
		Role:              model.UserRoleUser,
		EmailVerified:     true,
		FreeDownloadCount: 3,
	}, "access", "refresh")
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
}

func TestBuildLoginRespBuildsResponse(t *testing.T) {
	previous := getTokenRemainingTime
	getTokenRemainingTime = func(string) (int64, error) {
		return 3600, nil
	}
	defer func() {
		getTokenRemainingTime = previous
	}()

	resp, err := buildLoginResp(&model.Users{
		ID:                1,
		Nickname:          "tester",
		AvatarUrl:         "avatar",
		Role:              model.UserRoleAdmin,
		EmailVerified:     true,
		FreeDownloadCount: 3,
	}, "access", "refresh")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if resp.ExpiresIn != "3600" {
		t.Fatalf("expected expires_in 3600, got %s", resp.ExpiresIn)
	}
	if resp.UserProfile.Role != string(model.UserRoleAdmin) {
		t.Fatalf("expected role %s, got %s", model.UserRoleAdmin, resp.UserProfile.Role)
	}
}
