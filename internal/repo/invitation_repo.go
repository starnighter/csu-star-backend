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

type InvitationRepository interface {
	CreateInvitation(invitation *model.Invitations) error
	FindInviterIDByCode(code string) (int64, error)
	ConsumeInvitation(code string, inviteeID int64) (int64, error)
}

type invitationRepository struct {
	db *gorm.DB
}

func NewInvitationRepository(db *gorm.DB) InvitationRepository {
	return &invitationRepository{
		db: db,
	}
}

func (r *invitationRepository) CreateInvitation(invitation *model.Invitations) error {
	return r.db.Create(invitation).Error
}

func (r *invitationRepository) FindInviterIDByCode(code string) (int64, error) {
	inviterIDStr, err := utils.RDB.Get(utils.Ctx, constant.InviteCodePrefix+code).Result()
	if errors.Is(err, redis.Nil) {
		var invitation model.Invitations
		err = r.db.Select("inviter_id").
			Where("code = ? AND status = ? AND expires_at > ?", code, model.InvitationStatusPending, time.Now()).
			First(&invitation).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return 0, &constant.InviteCodeNotExistErr
		}
		if err != nil {
			return 0, err
		}
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
		result := tx.Where("code = ? AND status = ? AND expires_at > ?", code, model.InvitationStatusPending, time.Now()).
			First(&invitation)
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return &constant.InviteCodeNotExistErr
		}
		if result.Error != nil {
			return result.Error
		}

		updateResult := tx.Model(&model.Invitations{}).
			Where("id = ? AND status = ?", invitation.ID, model.InvitationStatusPending).
			Updates(map[string]interface{}{
				"invitee_id": inviteeID,
				"status":     model.InvitationStatusInvited,
				"used_at":    time.Now(),
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
