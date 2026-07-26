package service

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"sync"
	"time"

	"csu-star-backend/internal/model"
	"csu-star-backend/internal/repo"
	"csu-star-backend/logger"
	"csu-star-backend/pkg/mailer"
	"csu-star-backend/pkg/utils"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

var (
	ErrMailProviderNotFound = errors.New("mail provider not found")
	ErrMailProviderInvalid  = errors.New("mail provider invalid payload")
)

// providerCacheTTL 是通道列表的缓存时长。发信路径每封邮件都要读一次通道列表，
// 不缓存就是每封验证码一次数据库查询；管理端写入会立即失效缓存，
// 所以这个 TTL 只影响「绕过后台直接改库」的场景。
const providerCacheTTL = 30 * time.Second

type MailProviderService struct {
	repo repo.MailProviderRepository

	cacheMu sync.RWMutex
	cache   []mailer.Provider
	cacheAt time.Time
}

func NewMailProviderService(r repo.MailProviderRepository) *MailProviderService {
	return &MailProviderService{repo: r}
}

// Install 把本服务注册为 mailer 的通道来源。
// 注册之后 mailer 优先用数据库里的通道，本表为空时自动回落到 config.yaml。
func (s *MailProviderService) Install() {
	mailer.SetProviderSource(s.activeProviders)
}

// activeProviders 返回启用中的通道，带短 TTL 缓存。
func (s *MailProviderService) activeProviders() []mailer.Provider {
	s.cacheMu.RLock()
	fresh := time.Since(s.cacheAt) < providerCacheTTL && s.cacheAt.After(time.Time{})
	cached := s.cache
	s.cacheMu.RUnlock()
	if fresh {
		return cached
	}

	rows, err := s.repo.List()
	if err != nil {
		logger.Log.Error("读取邮件通道失败，回落到配置文件", zap.Error(err))
		// 返回 nil 让 mailer 走 config.yaml 兜底，而不是让发信直接失败。
		return nil
	}

	out := make([]mailer.Provider, 0, len(rows))
	for _, row := range rows {
		if !row.Enabled {
			continue
		}
		password, err := utils.DecryptSecret(row.Password)
		if err != nil {
			logger.Log.Error("邮件通道密码解密失败，已跳过该通道",
				zap.String("provider", row.Name), zap.Error(err))
			continue
		}
		out = append(out, toMailerProvider(row, password))
	}

	s.cacheMu.Lock()
	s.cache, s.cacheAt = out, time.Now()
	s.cacheMu.Unlock()
	return out
}

func (s *MailProviderService) invalidateCache() {
	s.cacheMu.Lock()
	s.cache, s.cacheAt = nil, time.Time{}
	s.cacheMu.Unlock()
}

func toMailerProvider(row model.MailProviders, password string) mailer.Provider {
	return mailer.Provider{
		Name:          row.Name,
		Kind:          mailer.NormalizeKind(row.Kind),
		Host:          row.Host,
		Port:          row.Port,
		TLSMode:       row.TLSMode,
		Username:      row.Username,
		Password:      password,
		FromEmailAddr: row.FromEmailAddr,
		FromName:      row.FromName,
		Tier:          row.Tier,
		Disabled:      !row.Enabled,
	}
}

// MailProviderInput 是管理端提交的通道配置。
// Password 为空表示「不修改」，因此接口从不需要回传明文密码。
type MailProviderInput struct {
	Name          string
	Kind          string
	Host          string
	Port          int
	TLSMode       string
	Username      string
	Password      string
	FromEmailAddr string
	FromName      string
	Tier          int
	Enabled       bool
}

// MailProviderItem 是回传给管理端的通道视图，不含密码明文。
type MailProviderItem struct {
	ID            string     `json:"id"`
	Name          string     `json:"name"`
	Kind          string     `json:"kind"`
	Host          string     `json:"host"`
	Port          int        `json:"port"`
	TLSMode       string     `json:"tls_mode"`
	Username      string     `json:"username"`
	PasswordMask  string     `json:"password_mask"`
	FromEmailAddr string     `json:"from_email_addr"`
	FromName      string     `json:"from_name"`
	Tier          int        `json:"tier"`
	Enabled       bool       `json:"enabled"`
	IsCloud       bool       `json:"is_cloud"`
	LastOkAt      *time.Time `json:"last_ok_at"`
	LastErrAt     *time.Time `json:"last_err_at"`
	LastErr       string     `json:"last_err"`
}

func (s *MailProviderService) List() ([]MailProviderItem, error) {
	rows, err := s.repo.List()
	if err != nil {
		return nil, err
	}

	out := make([]MailProviderItem, 0, len(rows))
	for _, row := range rows {
		kind := mailer.NormalizeKind(row.Kind)
		out = append(out, MailProviderItem{
			ID:            strconv.FormatInt(row.ID, 10),
			Name:          row.Name,
			Kind:          string(kind),
			Host:          row.Host,
			Port:          row.Port,
			TLSMode:       row.TLSMode,
			Username:      row.Username,
			PasswordMask:  utils.MaskSecret(row.Password),
			FromEmailAddr: row.FromEmailAddr,
			FromName:      row.FromName,
			Tier:          row.Tier,
			Enabled:       row.Enabled,
			IsCloud:       kind.IsCloud(),
			LastOkAt:      row.LastOkAt,
			LastErrAt:     row.LastErrAt,
			LastErr:       row.LastErr,
		})
	}
	return out, nil
}

func (s *MailProviderService) Create(in MailProviderInput) (*MailProviderItem, error) {
	normalized, err := normalizeProviderInput(in, true)
	if err != nil {
		return nil, err
	}

	encrypted, err := utils.EncryptSecret(normalized.Password)
	if err != nil {
		return nil, err
	}

	row := &model.MailProviders{
		Name:          normalized.Name,
		Kind:          normalized.Kind,
		Host:          normalized.Host,
		Port:          normalized.Port,
		TLSMode:       normalized.TLSMode,
		Username:      normalized.Username,
		Password:      encrypted,
		FromEmailAddr: normalized.FromEmailAddr,
		FromName:      normalized.FromName,
		Tier:          normalized.Tier,
		Enabled:       normalized.Enabled,
	}
	if err := s.repo.Create(row); err != nil {
		return nil, err
	}
	s.invalidateCache()

	items, err := s.List()
	if err != nil {
		return nil, err
	}
	for i := range items {
		if items[i].ID == strconv.FormatInt(row.ID, 10) {
			return &items[i], nil
		}
	}
	return nil, ErrMailProviderNotFound
}

func (s *MailProviderService) Update(id int64, in MailProviderInput) error {
	if _, err := s.repo.FindByID(id); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrMailProviderNotFound
		}
		return err
	}

	normalized, err := normalizeProviderInput(in, false)
	if err != nil {
		return err
	}

	fields := map[string]any{
		"name":            normalized.Name,
		"kind":            normalized.Kind,
		"host":            normalized.Host,
		"port":            normalized.Port,
		"tls_mode":        normalized.TLSMode,
		"username":        normalized.Username,
		"from_email_addr": normalized.FromEmailAddr,
		"from_name":       normalized.FromName,
		"tier":            normalized.Tier,
		"enabled":         normalized.Enabled,
	}
	// 密码留空表示不修改，这样后台编辑界面无需回显明文。
	if normalized.Password != "" {
		encrypted, err := utils.EncryptSecret(normalized.Password)
		if err != nil {
			return err
		}
		fields["password"] = encrypted
	}

	if err := s.repo.Update(id, fields); err != nil {
		return err
	}
	s.invalidateCache()
	return nil
}

func (s *MailProviderService) Delete(id int64) error {
	if _, err := s.repo.FindByID(id); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrMailProviderNotFound
		}
		return err
	}
	if err := s.repo.Delete(id); err != nil {
		return err
	}
	s.invalidateCache()
	return nil
}

// SendTest 用指定通道单独发一封测试邮件，绕过分层池，
// 这样管理员能确认「这一条通道」是否可用，而不是「某条通道」可用。
func (s *MailProviderService) SendTest(id int64, to string) error {
	row, err := s.repo.FindByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrMailProviderNotFound
		}
		return err
	}

	password, err := utils.DecryptSecret(row.Password)
	if err != nil {
		return err
	}

	sendErr := mailer.SendTestEmail(context.Background(), toMailerProvider(*row, password), to)
	if markErr := s.repo.MarkResult(id, sendErr); markErr != nil {
		logger.Log.Warn("记录邮件通道测试结果失败", zap.Error(markErr))
	}
	return sendErr
}

// Preflight 返回当前生效配置的自检结果，供后台展示。
func (s *MailProviderService) Preflight() (errs []string, warns []string) {
	return mailer.ValidateConfig()
}

func normalizeProviderInput(in MailProviderInput, requirePassword bool) (MailProviderInput, error) {
	out := in
	out.Name = strings.TrimSpace(in.Name)
	out.Host = strings.TrimSpace(strings.ToLower(in.Host))
	out.Username = strings.TrimSpace(in.Username)
	out.Password = strings.TrimSpace(in.Password)
	out.FromEmailAddr = strings.TrimSpace(strings.ToLower(in.FromEmailAddr))
	out.FromName = strings.TrimSpace(in.FromName)
	out.Kind = string(mailer.NormalizeKind(strings.TrimSpace(in.Kind)))

	out.TLSMode = strings.TrimSpace(strings.ToLower(in.TLSMode))
	if out.TLSMode == "" {
		out.TLSMode = mailer.TLSModeImplicit
	}
	if out.TLSMode != mailer.TLSModeImplicit && out.TLSMode != mailer.TLSModeStartTLS {
		return out, ErrMailProviderInvalid
	}

	if out.Name == "" || out.Host == "" || out.Username == "" || out.FromEmailAddr == "" {
		return out, ErrMailProviderInvalid
	}
	if out.Port <= 0 || out.Port > 65535 {
		return out, ErrMailProviderInvalid
	}
	if requirePassword && out.Password == "" {
		return out, ErrMailProviderInvalid
	}
	if !mailer.ValidAddress(out.FromEmailAddr) {
		return out, ErrMailProviderInvalid
	}
	if out.Tier < 0 {
		out.Tier = 0
	}
	return out, nil
}
