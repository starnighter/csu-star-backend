package repo

import (
	"csu-star-backend/internal/model"
	"errors"
	"strings"
	"time"

	"gorm.io/gorm"
)

type UserRepository interface {
	CreateUser(user *model.Users) error
	RewardInviter(inviterID int64) error
	RewardInvitee(inviteeID int64) error
	UpdateEmailByID(userID int64, email string) error
	UpdatePasswordByID(userID int64, password string) error
	FindUserByID(userID int64) (*model.Users, error)
	FindUserByEmail(email string) (*model.Users, error)
	FindUserOauthBinding(provider model.OauthProvider, openID string) (*model.UserOauthBinding, error)
	FindOrCreateOauthUser(provider model.OauthProvider, userInfo *model.UserInfo) (*model.Users, error)
	CreateUserOauthBinding(userID int64, provider model.OauthProvider, userInfo *model.UserInfo) (*model.UserOauthBinding, error)
}

type userRepository struct {
	db *gorm.DB
}

func NewUserRepository(db *gorm.DB) UserRepository {
	return &userRepository{db: db}
}

func (r *userRepository) CreateUser(user *model.Users) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(user).Error; err != nil {
			return err
		}
		if user.Points <= 0 {
			return nil
		}
		return tx.Create(&model.PointsRecords{
			UserID:    user.ID,
			Type:      model.PointsTypeInitial,
			Delta:     user.Points,
			Balance:   user.Points,
			Reason:    "新用户注册初始积分",
			RelatedID: 0,
		}).Error
	})
}

func (r *userRepository) RewardInviter(inviterID int64) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		var inviter model.Users
		if err := tx.Select("id, points, status").Where("id = ?", inviterID).First(&inviter).Error; err != nil {
			return err
		}
		if inviter.Status != model.UserStatusActive {
			return gorm.ErrRecordNotFound
		}

		if err := tx.Model(&model.Users{}).
			Where("id = ?", inviterID).
			Update("points", gorm.Expr("points + ?", 3)).Error; err != nil {
			return err
		}
		inviter.Points += 3

		if err := tx.Create(&model.PointsRecords{
			UserID:    inviterID,
			Type:      model.PointsTypeInvite,
			Delta:     3,
			Balance:   inviter.Points,
			Reason:    "邀请新用户注册奖励积分",
			RelatedID: 0,
		}).Error; err != nil {
			return err
		}

		return tx.Create(&model.Notifications{
			UserID:    inviterID,
			Type:      model.NotificationPointsChanged,
			Title:     "邀请奖励到账",
			Content:   "你邀请的新用户已完成注册，获得 3 积分奖励。",
			RelatedID: 0,
			IsRead:    false,
			IsGlobal:  false,
		}).Error
	})
}

func (r *userRepository) RewardInvitee(inviteeID int64) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		var invitee model.Users
		if err := tx.Select("id, points, status").Where("id = ?", inviteeID).First(&invitee).Error; err != nil {
			return err
		}
		if invitee.Status != model.UserStatusActive {
			return gorm.ErrRecordNotFound
		}

		if err := tx.Model(&model.Users{}).
			Where("id = ?", inviteeID).
			Update("points", gorm.Expr("points + ?", 3)).Error; err != nil {
			return err
		}
		invitee.Points += 3

		if err := tx.Create(&model.PointsRecords{
			UserID:    inviteeID,
			Type:      model.PointsTypeInvite,
			Delta:     3,
			Balance:   invitee.Points,
			Reason:    "填写邀请码注册奖励积分",
			RelatedID: 0,
		}).Error; err != nil {
			return err
		}

		return tx.Create(&model.Notifications{
			UserID:    inviteeID,
			Type:      model.NotificationPointsChanged,
			Title:     "邀请码奖励到账",
			Content:   "你已使用邀请码完成注册，获得 3 积分奖励。",
			RelatedID: 0,
			IsRead:    false,
			IsGlobal:  false,
		}).Error
	})
}

func (r *userRepository) UpdateEmailByID(userID int64, email string) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		result := tx.Model(&model.Users{}).Where("id = ?", userID).Updates(map[string]interface{}{
			"email":          email,
			"email_verified": true,
		})
		if result.Error != nil {
			return result.Error
		}
		return nil
	})
}

func (r *userRepository) FindUserByID(userID int64) (*model.Users, error) {
	var user model.Users
	err := r.db.Where("id = ?", userID).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *userRepository) FindUserByEmail(email string) (*model.Users, error) {
	var user model.Users
	err := r.db.Where("email = ?", email).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *userRepository) FindUserOauthBinding(provider model.OauthProvider, openID string) (*model.UserOauthBinding, error) {
	var binding model.UserOauthBinding
	if err := r.db.Where("provider = ? AND openid = ?", provider, openID).First(&binding).Error; err != nil {
		return nil, err
	}
	return &binding, nil
}

func (r *userRepository) FindOrCreateOauthUser(provider model.OauthProvider, userInfo *model.UserInfo) (*model.Users, error) {
	var userOauthBinding model.UserOauthBinding
	var user model.Users

	err := r.db.Transaction(func(tx *gorm.DB) error {
		err := tx.Where("provider = ? AND openid = ?", provider, userInfo.OpenID).First(&userOauthBinding).Error
		if err == nil {
			updates := map[string]interface{}{
				"last_login_at": time.Now(),
			}
			if avatarURL := strings.TrimSpace(userInfo.AvatarUrl); avatarURL != "" {
				updates["avatar_url"] = avatarURL
			}
			if err := tx.Model(&model.Users{}).
				Where("id = ?", userOauthBinding.UserID).
				Updates(updates).Error; err != nil {
				return err
			}

			// 用户已使用该提供商注册过，刷新头像与登录时间后直接返回
			return tx.First(&user, userOauthBinding.UserID).Error
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}

		// 用户未使用该提供商注册过，创建新用户并关联
		user = model.Users{
			Nickname:    userInfo.Nickname,
			AvatarUrl:   userInfo.AvatarUrl,
			Status:      model.UserStatusActive,
			LastLoginAt: time.Now(),
		}
		err = tx.Create(&user).Error
		if err != nil {
			return err
		}

		userOauthBinding = model.UserOauthBinding{
			UserID:   user.ID,
			Provider: provider,
			OpenID:   userInfo.OpenID,
			BoundAt:  time.Now(),
		}
		if provider == model.OauthProviderWechat {
			userOauthBinding.UnionID = userInfo.UnionId
		}
		return tx.Create(&userOauthBinding).Error
	})
	return &user, err
}

func (r *userRepository) CreateUserOauthBinding(userID int64, provider model.OauthProvider, userInfo *model.UserInfo) (*model.UserOauthBinding, error) {
	var userOauthBinding model.UserOauthBinding

	err := r.db.Transaction(func(tx *gorm.DB) error {
		userOauthBinding = model.UserOauthBinding{
			UserID:   userID,
			Provider: provider,
			OpenID:   userInfo.OpenID,
			BoundAt:  time.Now(),
		}
		if provider == model.OauthProviderWechat {
			userOauthBinding.UnionID = userInfo.UnionId
		}
		if err := tx.Create(&userOauthBinding).Error; err != nil {
			return err
		}

		if avatarURL := strings.TrimSpace(userInfo.AvatarUrl); avatarURL != "" {
			if err := tx.Model(&model.Users{}).
				Where("id = ?", userID).
				Update("avatar_url", avatarURL).Error; err != nil {
				return err
			}
		}
		return nil
	})
	return &userOauthBinding, err
}

func (r *userRepository) UpdatePasswordByID(userID int64, password string) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		err := tx.Model(&model.Users{}).Where("id = ?", userID).Update("password", password).Error
		if err != nil {
			return err
		}
		return nil
	})
}
