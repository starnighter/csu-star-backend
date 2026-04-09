package repo

import (
	"csu-star-backend/internal/constant"
	"csu-star-backend/internal/model"
	"csu-star-backend/pkg/utils"
	"errors"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

type InviteCodeInfo struct {
	InviteCode string `json:"invite_code"`
	UsedCount  int64  `json:"used_count"`
}

type InvitationRepository interface {
	CreateInvitation(invitation *model.Invitations) error
	GetOrCreateActiveInvitation(inviterID int64) (*model.Invitations, error)
	CountUsedInvitations(inviterID int64) (int64, error)
	FindInviterIDByCode(code string) (int64, error)
	ConsumeInvitation(code string, inviteeID int64) (int64, error)
}

type invitationRepository struct {
	db *gorm.DB
}

var permanentInvitationExpiry = time.Date(9999, 12, 31, 23, 59, 59, 0, time.UTC)

const inviteCodeLength = 6

func NewInvitationRepository(db *gorm.DB) InvitationRepository {
	return &invitationRepository{
		db: db,
	}
}

func (r *invitationRepository) CreateInvitation(invitation *model.Invitations) error {
	if invitation.ExpiresAt == nil {
		invitation.ExpiresAt = permanentInvitationExpiryPtr()
	}
	if err := r.db.Create(invitation).Error; err != nil {
		return err
	}
	return utils.RDB.Set(utils.Ctx, constant.InviteCodePrefix+invitation.Code, invitation.InviterID, 0).Err()
}

func (r *invitationRepository) GetOrCreateActiveInvitation(inviterID int64) (*model.Invitations, error) {
	var invitation model.Invitations
	var staleCode string
	err := r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec("SELECT pg_advisory_xact_lock(?)", inviterID).Error; err != nil {
			return err
		}

		result := tx.Where("inviter_id = ? AND status = ?", inviterID, model.InvitationStatusPending).
			Order("created_at DESC").
			Limit(1).
			Find(&invitation)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected > 0 {
			if len(invitation.Code) == inviteCodeLength {
				return nil
			}

			staleCode = invitation.Code
			code, codeErr := r.generateUniqueInviteCode(tx)
			if codeErr != nil {
				return codeErr
			}
			if err := tx.Model(&model.Invitations{}).
				Where("id = ?", invitation.ID).
				Update("code", code).Error; err != nil {
				return err
			}
			invitation.Code = code
			return nil
		}

		code, codeErr := r.generateUniqueInviteCode(tx)
		if codeErr != nil {
			return codeErr
		}

		invitation = model.Invitations{
			InviterID: inviterID,
			Code:      code,
			Status:    model.InvitationStatusPending,
			ExpiresAt: permanentInvitationExpiryPtr(),
		}
		return tx.Create(&invitation).Error
	})
	if err != nil {
		return nil, err
	}

	if staleCode != "" {
		_ = utils.RDB.Del(utils.Ctx, constant.InviteCodePrefix+staleCode).Err()
	}

	if err := utils.RDB.Set(utils.Ctx, constant.InviteCodePrefix+invitation.Code, invitation.InviterID, 0).Err(); err != nil {
		return nil, err
	}
	return &invitation, nil
}

func (r *invitationRepository) generateUniqueInviteCode(tx *gorm.DB) (string, error) {
	for range 5 {
		code, err := utils.GenerateInviteCode()
		if err != nil {
			return "", err
		}

		var count int64
		if err := tx.Model(&model.Invitations{}).Where("code = ?", code).Count(&count).Error; err != nil {
			return "", err
		}
		if count > 0 {
			continue
		}

		return code, nil
	}

	return "", errors.New("generate invite code failed")
}

func (r *invitationRepository) CountUsedInvitations(inviterID int64) (int64, error) {
	var count int64
	err := r.db.Model(&model.Invitations{}).
		Where("inviter_id = ? AND status = ?", inviterID, model.InvitationStatusInvited).
		Count(&count).Error
	return count, err
}

func (r *invitationRepository) FindInviterIDByCode(code string) (int64, error) {
	inviterIDStr, err := utils.RDB.Get(utils.Ctx, constant.InviteCodePrefix+code).Result()
	if errors.Is(err, redis.Nil) {
		var invitation model.Invitations
		err = r.db.Select("inviter_id").
			Where("code = ? AND status = ?", code, model.InvitationStatusPending).
			First(&invitation).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return 0, &constant.InviteCodeNotExistErr
		}
		if err != nil {
			return 0, err
		}
		_ = utils.RDB.Set(utils.Ctx, constant.InviteCodePrefix+code, invitation.InviterID, 0).Err()
		return invitation.InviterID, nil
	}
	if err != nil {
		return 0, err
	}

	inviterID, err := strconv.ParseInt(inviterIDStr, 10, 64)
	if err != nil {
		return 0, err
	}
	return inviterID, nil
}

func (r *invitationRepository) ConsumeInvitation(code string, inviteeID int64) (int64, error) {
	var invitation model.Invitations
	err := r.db.Transaction(func(tx *gorm.DB) error {
		result := tx.Where("code = ? AND status = ?", code, model.InvitationStatusPending).
			First(&invitation)
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return &constant.InviteCodeNotExistErr
		}
		if result.Error != nil {
			return result.Error
		}

		now := time.Now()
		updateResult := tx.Model(&model.Invitations{}).
			Where("id = ? AND status = ?", invitation.ID, model.InvitationStatusPending).
			Updates(map[string]interface{}{
				"invitee_id": inviteeID,
				"status":     model.InvitationStatusInvited,
				"used_at":    &now,
			})
		if updateResult.Error != nil {
			return updateResult.Error
		}
		if updateResult.RowsAffected == 0 {
			return &constant.InviteCodeNotExistErr
		}
		return nil
	})
	if err != nil {
		return 0, err
	}

	_ = utils.RDB.Del(utils.Ctx, constant.InviteCodePrefix+code).Err()
	return invitation.InviterID, nil
}

func permanentInvitationExpiryPtr() *time.Time {
	t := permanentInvitationExpiry
	return &t
}
