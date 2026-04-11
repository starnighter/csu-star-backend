package service

import (
	"csu-star-backend/internal/constant"
	"csu-star-backend/internal/model"
	"csu-star-backend/pkg/utils"
	"errors"
	"testing"

	"gorm.io/gorm"
)

type authUserRepositoryStub struct {
	createdUser         *model.Users
	findUserByEmail     *model.Users
	findUserByEmailErr  error
	findUserByID        *model.Users
	findUserByIDErr     error
	rewardInviterID     int64
	rewardInviteeID     int64
	rewardInviterCalled bool
	rewardInviteeCalled bool
	updateEmailUserID   int64
	updateEmailValue    string
	updateEmailCalled   bool
}

func (s *authUserRepositoryStub) CreateUser(user *model.Users) error {
	user.ID = 1001
	s.createdUser = user
	return nil
}

func (s *authUserRepositoryStub) RewardInviter(inviterID int64) error {
	s.rewardInviterCalled = true
	s.rewardInviterID = inviterID
	return nil
}

func (s *authUserRepositoryStub) RewardInvitee(inviteeID int64) error {
	s.rewardInviteeCalled = true
	s.rewardInviteeID = inviteeID
	return nil
}

func (s *authUserRepositoryStub) UpdateEmailByID(userID int64, email string) error {
	s.updateEmailCalled = true
	s.updateEmailUserID = userID
	s.updateEmailValue = email
	return nil
}

func (s *authUserRepositoryStub) UpdatePasswordByID(userID int64, password string) error {
	return nil
}

func (s *authUserRepositoryStub) FindUserByID(userID int64) (*model.Users, error) {
	return s.findUserByID, s.findUserByIDErr
}

func (s *authUserRepositoryStub) FindUserByEmail(email string) (*model.Users, error) {
	if s.findUserByEmail != nil || s.findUserByEmailErr == nil {
		return s.findUserByEmail, s.findUserByEmailErr
	}
	return nil, s.findUserByEmailErr
}

func (s *authUserRepositoryStub) FindUserOauthBinding(provider model.OauthProvider, openID string) (*model.UserOauthBinding, error) {
	return nil, gorm.ErrRecordNotFound
}

func (s *authUserRepositoryStub) FindOrCreateOauthUser(provider model.OauthProvider, userInfo *model.UserInfo) (*model.Users, error) {
	return nil, nil
}

func (s *authUserRepositoryStub) CreateUserOauthBinding(userID int64, provider model.OauthProvider, userInfo *model.UserInfo) (*model.UserOauthBinding, error) {
	return nil, nil
}

type authInvitationRepositoryStub struct {
	inviterID         int64
	findErr           error
	consumedInviterID int64
	consumeErr        error
}

func (s *authInvitationRepositoryStub) CreateInvitation(invitation *model.Invitations) error {
	return nil
}

func (s *authInvitationRepositoryStub) GetOrCreateActiveInvitation(inviterID int64) (*model.Invitations, error) {
	return nil, nil
}

func (s *authInvitationRepositoryStub) CountUsedInvitations(inviterID int64) (int64, error) {
	return 0, nil
}

func (s *authInvitationRepositoryStub) FindInviterIDByCode(code string) (int64, error) {
	return s.inviterID, s.findErr
}

func (s *authInvitationRepositoryStub) ConsumeInvitation(code string, inviteeID int64) (int64, error) {
	return s.consumedInviterID, s.consumeErr
}

func TestRegisterWithInviteCodeRewardsBothSides(t *testing.T) {
	userRepo := &authUserRepositoryStub{
		findUserByEmailErr: gorm.ErrRecordNotFound,
	}
	invitationRepo := &authInvitationRepositoryStub{
		inviterID:         2001,
		consumedInviterID: 2001,
	}
	service := NewAuthService(userRepo, invitationRepo)

	err := service.Register("test@csu.edu.cn", "password123", "tester", "", "INV001")
	if err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	if userRepo.createdUser == nil {
		t.Fatal("expected user to be created")
	}
	if userRepo.createdUser.InviterID == nil || *userRepo.createdUser.InviterID != 2001 {
		t.Fatalf("expected inviter id 2001, got %v", userRepo.createdUser.InviterID)
	}
	if !userRepo.rewardInviteeCalled || userRepo.rewardInviteeID != 1001 {
		t.Fatalf("expected invitee reward for user 1001, got called=%v id=%d", userRepo.rewardInviteeCalled, userRepo.rewardInviteeID)
	}
	if !userRepo.rewardInviterCalled || userRepo.rewardInviterID != 2001 {
		t.Fatalf("expected inviter reward for user 2001, got called=%v id=%d", userRepo.rewardInviterCalled, userRepo.rewardInviterID)
	}
}

func TestRegisterWithInvalidInviteCodeReturnsBusinessError(t *testing.T) {
	userRepo := &authUserRepositoryStub{
		findUserByEmailErr: gorm.ErrRecordNotFound,
	}
	service := NewAuthService(userRepo, &authInvitationRepositoryStub{
		findErr: &constant.InviteCodeNotExistErr,
	})

	err := service.Register("test@csu.edu.cn", "password123", "tester", "", "INV001")
	if !errors.Is(err, &constant.InviteCodeNotExistErr) {
		t.Fatalf("expected invite code error, got %v", err)
	}
}

func TestBindEmailRequiresCampusSuffix(t *testing.T) {
	userRepo := &authUserRepositoryStub{
		findUserByID: &model.Users{ID: 1001},
	}
	service := NewAuthService(userRepo, &authInvitationRepositoryStub{})

	err := service.BindEmail(1001, "test@example.com")
	if !errors.Is(err, &constant.InvalidSchoolEmailErr) {
		t.Fatalf("expected invalid school email error, got %v", err)
	}
	if userRepo.updateEmailCalled {
		t.Fatal("expected email not to be updated for invalid school email")
	}
}

func TestBindEmailNormalizesCampusEmail(t *testing.T) {
	userRepo := &authUserRepositoryStub{
		findUserByID: &model.Users{ID: 1001},
	}
	service := NewAuthService(userRepo, &authInvitationRepositoryStub{})

	err := service.BindEmail(1001, "  TEST@CSU.EDU.CN ")
	if err != nil {
		t.Fatalf("BindEmail() error = %v", err)
	}
	if !userRepo.updateEmailCalled {
		t.Fatal("expected email update to be called")
	}
	if userRepo.updateEmailValue != "test@csu.edu.cn" {
		t.Fatalf("expected normalized school email, got %s", userRepo.updateEmailValue)
	}
}

func TestLoginRejectsBannedUserWithoutSecurityService(t *testing.T) {
	password := "password123"
	hash, err := utils.HashPassword(password)
	if err != nil {
		t.Fatalf("HashPassword() error = %v", err)
	}

	userRepo := &authUserRepositoryStub{
		findUserByEmail: &model.Users{
			ID:       1001,
			Email:    stringPtr("test@csu.edu.cn"),
			Password: hash,
			Status:   model.UserStatusBanned,
		},
	}
	service := NewAuthService(userRepo, &authInvitationRepositoryStub{})

	_, _, _, err = service.Login("test@csu.edu.cn", password)
	if !errors.Is(err, &constant.UserBannedErr) {
		t.Fatalf("expected banned error, got %v", err)
	}
}

func stringPtr(value string) *string {
	return &value
}
