package repo

import (
	"time"

	"csu-star-backend/internal/model"

	"gorm.io/gorm"
)

type MailProviderRepository interface {
	List() ([]model.MailProviders, error)
	FindByID(id int64) (*model.MailProviders, error)
	Create(p *model.MailProviders) error
	Update(id int64, fields map[string]any) error
	Delete(id int64) error
	MarkResult(id int64, sendErr error) error
}

type mailProviderRepository struct {
	db *gorm.DB
}

func NewMailProviderRepository(db *gorm.DB) MailProviderRepository {
	return &mailProviderRepository{db: db}
}

func (r *mailProviderRepository) List() ([]model.MailProviders, error) {
	var out []model.MailProviders
	err := r.db.Order("tier ASC, id ASC").Find(&out).Error
	return out, err
}

func (r *mailProviderRepository) FindByID(id int64) (*model.MailProviders, error) {
	var p model.MailProviders
	if err := r.db.Where("id = ?", id).First(&p).Error; err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *mailProviderRepository) Create(p *model.MailProviders) error {
	return r.db.Create(p).Error
}

func (r *mailProviderRepository) Update(id int64, fields map[string]any) error {
	if len(fields) == 0 {
		return nil
	}
	return r.db.Model(&model.MailProviders{}).Where("id = ?", id).Updates(fields).Error
}

func (r *mailProviderRepository) Delete(id int64) error {
	return r.db.Where("id = ?", id).Delete(&model.MailProviders{}).Error
}

// MarkResult 记录最近一次投递结果，供后台展示通道健康度。
// 它刻意不返回业务错误：健康度写失败不应该影响发信本身。
func (r *mailProviderRepository) MarkResult(id int64, sendErr error) error {
	now := time.Now()
	fields := map[string]any{}
	if sendErr == nil {
		fields["last_ok_at"] = now
		fields["last_err"] = ""
	} else {
		fields["last_err_at"] = now
		msg := sendErr.Error()
		if len(msg) > 500 {
			msg = msg[:500]
		}
		fields["last_err"] = msg
	}
	return r.db.Model(&model.MailProviders{}).Where("id = ?", id).Updates(fields).Error
}
