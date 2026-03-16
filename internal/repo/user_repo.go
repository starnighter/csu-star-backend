package repo

import (
	"csu-star-backend/internal/constant"
	"csu-star-backend/internal/model"
	"csu-star-backend/pkg/utils"
	"errors"
	"strconv"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

type UserRepository interface {
	FindInviterAndAddPoints(inviteCode string) (int64, error)
	CreateUser(user *model.Users) error
}

type userRepository struct {
	db *gorm.DB
}

func NewUserRepository(db *gorm.DB) UserRepository {
	return &userRepository{db: db}
}

func (r *userRepository) FindInviterAndAddPoints(inviteCode string) (int64, error) {
	inviterIDStr, err := utils.RDB.GetDel(utils.Ctx, constant.InviteCodePrefix+inviteCode).Result()
	if errors.Is(err, redis.Nil) {
		return 0, nil
	} else if err != nil {
		return 0, err
	}

	inviterID, err := strconv.ParseInt(inviterIDStr, 10, 64)
	if err != nil {
		return 0, err
	}

	var user model.Users
	err = r.db.Transaction(func(tx *gorm.DB) error {
		result := tx.Model(&user).Where(
			"id = ? AND status = ?",
			inviterID,
			model.UserStatusActive,
		).Update("points", gorm.Expr("points + ?", 3))
		if result.Error != nil {
			return result.Error
		}

		return nil
	})
	if err != nil {
		return 0, err
	}

	return inviterID, nil
}

func (r *userRepository) CreateUser(user *model.Users) error {
	return r.db.Create(user).Error
}
