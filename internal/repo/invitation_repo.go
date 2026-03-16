package repo

import (
	"csu-star-backend/internal/model"

	"gorm.io/gorm"
)

type InvitationRepository interface {
	CreateInvitation(invitation *model.Invitations) error
	UpdateInvitationByCode(invitation *model.Invitations, code string) error
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

func (r *invitationRepository) UpdateInvitationByCode(invitation *model.Invitations, code string) error {
	err := r.db.Transaction(func(tx *gorm.DB) error {
		result := tx.Model(invitation).Where("code = ?", code).Updates(invitation)
		if result.Error != nil {
			return result.Error
		}
		return nil
	})
	return err
}
