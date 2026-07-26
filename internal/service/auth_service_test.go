package service

import (
	"csu-star-backend/config"
	"csu-star-backend/internal/constant"
	"csu-star-backend/internal/emailpolicy"
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

func TestBindEmailRejectsMalformedEmail(t *testing.T) {
	userRepo := &authUserRepositoryStub{
		findUserByID: &model.Users{ID: 1001},
	}
	service := NewAuthService(userRepo, &authInvitationRepositoryStub{})

	err := service.BindEmail(1001, "not-an-email")
	if !errors.Is(err, &constant.InvalidEmailFormatErr) {
		t.Fatalf("expected invalid email format error, got %v", err)
	}
	if userRepo.updateEmailCalled {
		t.Fatal("expected email not to be updated for malformed email")
	}
}

// 域名策略默认放开，只有显式配成 allow_list 才会拒绝。
func TestBindEmailRejectsDisallowedDomain(t *testing.T) {
	config.SetConfig(&config.Config{
		Mail: config.MailConfig{
			AccountEmail: config.AccountEmailConfig{
				Policy:         emailpolicy.ModeAllowList,
				AllowedDomains: []string{"csu.edu.cn"},
			},
		},
	})
	t.Cleanup(func() { config.SetConfig(nil) })

	userRepo := &authUserRepositoryStub{
		findUserByID: &model.Users{ID: 1001},
	}
	service := NewAuthService(userRepo, &authInvitationRepositoryStub{})

	err := service.BindEmail(1001, "test@qq.com")
	if !errors.Is(err, &constant.EmailDomainNotAllowedErr) {
		t.Fatalf("expected domain not allowed error, got %v", err)
	}
	if userRepo.updateEmailCalled {
		t.Fatal("expected email not to be updated for disallowed domain")
	}
}

// 域名收紧只影响创建/绑定路径；登录必须仍然放行，
// 否则改一次配置就会把存量用户锁在自己的账号外面。
func TestLoginIgnoresDomainPolicy(t *testing.T) {
	config.SetConfig(&config.Config{
		Mail: config.MailConfig{
			AccountEmail: config.AccountEmailConfig{
				Policy:         emailpolicy.ModeAllowList,
				AllowedDomains: []string{"csu.edu.cn"},
			},
		},
	})
	t.Cleanup(func() { config.SetConfig(nil) })

	password := "password123"
	hash, err := utils.HashPassword(password)
	if err != nil {
		t.Fatalf("HashPassword() error = %v", err)
	}
	userRepo := &authUserRepositoryStub{
		findUserByEmail: &model.Users{
			ID:       1001,
			Email:    stringPtr("legacy@qq.com"),
			Password: hash,
			Status:   model.UserStatusActive,
		},
	}
	service := NewAuthService(userRepo, &authInvitationRepositoryStub{})

	if _, _, _, err := service.Login("legacy@qq.com", password); err != nil {
		t.Fatalf("expected login to succeed despite allow-list, got %v", err)
	}
}

func TestBindEmailNormalizesEmail(t *testing.T) {
	cases := []struct{ in, want string }{
		{"  TEST@CSU.EDU.CN ", "test@csu.edu.cn"},
		{"  User@QQ.Com ", "user@qq.com"},
	}

	for _, tc := range cases {
		userRepo := &authUserRepositoryStub{
			findUserByID: &model.Users{ID: 1001},
		}
		service := NewAuthService(userRepo, &authInvitationRepositoryStub{})

		if err := service.BindEmail(1001, tc.in); err != nil {
			t.Fatalf("BindEmail(%q) error = %v", tc.in, err)
		}
		if !userRepo.updateEmailCalled {
			t.Fatalf("BindEmail(%q): expected email update to be called", tc.in)
		}
		if userRepo.updateEmailValue != tc.want {
			t.Fatalf("BindEmail(%q) = %s, want %s", tc.in, userRepo.updateEmailValue, tc.want)
		}
	}
}

func TestCaptchaIDDistinguishesDomains(t *testing.T) {
	// 旧方案把邮箱去掉 @csu.edu.cn 当 key，12345@qq.com 不匹配后缀会原样返回，
	// 与校园邮箱的 12345 行为不一致。哈希方案必须对两者给出不同且定长的 key。
	campus := captchaID("12345@csu.edu.cn")
	qq := captchaID("12345@qq.com")

	if campus == qq {
		t.Fatal("expected different captcha ids for different domains")
	}
	if len(campus) != 32 || len(qq) != 32 {
		t.Fatalf("expected 32-char ids, got %d and %d", len(campus), len(qq))
	}
	if campus != captchaID("12345@csu.edu.cn") {
		t.Fatal("expected captchaID to be deterministic")
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
