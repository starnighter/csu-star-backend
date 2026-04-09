package service

import (
	"context"
	"csu-star-backend/config"
	"csu-star-backend/internal/model"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

var ErrSecurityUserBanned = errors.New("security user banned")

type RateLimitDecision struct {
	Allowed    bool
	Remaining  int64
	RetryAfter time.Duration
	Current    int64
}

type BanDecision struct {
	Banned         bool       `json:"banned"`
	BanUntil       *time.Time `json:"ban_until,omitempty"`
	BanReason      string     `json:"ban_reason,omitempty"`
	BanSource      string     `json:"ban_source,omitempty"`
	ViolationCount int        `json:"violation_count"`
	Permanent      bool       `json:"permanent"`
}

type SecurityService struct {
	db *gorm.DB
}

func NewSecurityService(db *gorm.DB) *SecurityService {
	return &SecurityService{db: db}
}

func (s *SecurityService) Mode() string {
	if config.GlobalConfig == nil {
		return "enforce"
	}
	mode := strings.TrimSpace(strings.ToLower(config.GlobalConfig.Security.Mode))
	if mode == "" {
		return "enforce"
	}
	return mode
}

func (s *SecurityService) IsObserveMode() bool {
	return s.Mode() == "observe"
}

func (s *SecurityService) RateLimit(ctx context.Context, rdb redis.UniversalClient, key string, limit int64, window time.Duration) (RateLimitDecision, error) {
	if rdb == nil {
		return RateLimitDecision{Allowed: true, Remaining: limit}, nil
	}

	pipe := rdb.TxPipeline()
	incr := pipe.Incr(ctx, key)
	ttl := pipe.TTL(ctx, key)
	_, err := pipe.Exec(ctx)
	if err != nil {
		return RateLimitDecision{}, err
	}

	current := incr.Val()
	currentTTL := ttl.Val()
	if current == 1 || currentTTL <= 0 {
		if err := rdb.Expire(ctx, key, window).Err(); err != nil {
			return RateLimitDecision{}, err
		}
		currentTTL = window
	}

	remaining := limit - current
	if remaining < 0 {
		remaining = 0
	}

	return RateLimitDecision{
		Allowed:    current <= limit,
		Remaining:  remaining,
		RetryAfter: currentTTL,
		Current:    current,
	}, nil
}

func (s *SecurityService) EnforceUserAccess(userID int64) (*model.Users, *BanDecision, error) {
	if s == nil || s.db == nil {
		return nil, nil, nil
	}

	var user model.Users
	if err := s.db.Where("id = ?", userID).First(&user).Error; err != nil {
		return nil, nil, err
	}

	if user.Status != model.UserStatusBanned {
		return &user, nil, nil
	}

	now := time.Now()
	if user.BanSource == model.UserBanSourceSystem && user.BanUntil != nil && user.BanUntil.Before(now) {
		oldValues := jsonMap(
			"status", user.Status,
			"ban_until", user.BanUntil,
			"ban_source", user.BanSource,
			"ban_reason", user.BanReason,
		)
		user.Status = model.UserStatusActive
		user.BanUntil = nil
		user.BanReason = ""
		user.BanSource = ""
		if err := s.db.Transaction(func(tx *gorm.DB) error {
			if err := tx.Model(&model.Users{}).Where("id = ?", user.ID).Updates(map[string]interface{}{
				"status":     model.UserStatusActive,
				"ban_until":  nil,
				"ban_reason": "",
				"ban_source": "",
			}).Error; err != nil {
				return err
			}
			return tx.Create(&model.AuditLogs{
				OperatorID: 0,
				Action:     model.AuditActionAutoUnban,
				TargetType: "user",
				TargetID:   user.ID,
				OldValues:  oldValues,
				NewValues: jsonMap(
					"status", model.UserStatusActive,
					"ban_until", nil,
					"ban_source", "",
					"ban_reason", "",
				),
				Reason: "系统自动解封",
			}).Error
		}); err != nil {
			return nil, nil, err
		}
		return &user, nil, nil
	}

	return &user, &BanDecision{
		Banned:         true,
		BanUntil:       user.BanUntil,
		BanReason:      user.BanReason,
		BanSource:      user.BanSource,
		ViolationCount: user.ViolationCount,
		Permanent:      user.BanUntil == nil,
	}, ErrSecurityUserBanned
}

func (s *SecurityService) RecordViolation(userID int64, scope, triggerKey, reason string, evidence map[string]interface{}) (*BanDecision, error) {
	if s == nil || s.db == nil || userID <= 0 {
		return nil, nil
	}

	var result *BanDecision
	err := s.db.Transaction(func(tx *gorm.DB) error {
		var user model.Users
		if err := tx.Where("id = ?", userID).First(&user).Error; err != nil {
			return err
		}

		now := time.Now()
		nextCount := user.ViolationCount + 1
		banDuration := violationBanDuration(nextCount)
		var banUntil *time.Time
		if banDuration > 0 {
			until := now.Add(banDuration)
			banUntil = &until
		}

		rawEvidence, err := json.Marshal(evidence)
		if err != nil {
			return err
		}

		if err := tx.Create(&model.UserViolations{
			UserID:             userID,
			Scope:              strings.TrimSpace(scope),
			TriggerKey:         strings.TrimSpace(triggerKey),
			Reason:             strings.TrimSpace(reason),
			Evidence:           datatypes.JSON(rawEvidence),
			PenaltyLevel:       nextCount,
			BanDurationSeconds: int64(banDuration.Seconds()),
		}).Error; err != nil {
			return err
		}

		oldValues := jsonMap(
			"status", user.Status,
			"ban_until", user.BanUntil,
			"ban_reason", user.BanReason,
			"ban_source", user.BanSource,
			"violation_count", user.ViolationCount,
		)
		update := map[string]interface{}{
			"violation_count":   nextCount,
			"last_violation_at": now,
		}
		if !s.IsObserveMode() {
			update["ban_reason"] = reason
			update["ban_source"] = model.UserBanSourceSystem
			update["status"] = model.UserStatusBanned
			update["ban_until"] = banUntil
		}
		if err := tx.Model(&model.Users{}).Where("id = ?", userID).Updates(update).Error; err != nil {
			return err
		}

		if err := tx.Create(&model.AuditLogs{
			OperatorID: 0,
			Action:     model.AuditActionAutoViolation,
			TargetType: "user",
			TargetID:   userID,
			OldValues:  jsonMap("violation_count", user.ViolationCount),
			NewValues: jsonMap(
				"violation_count", nextCount,
				"scope", scope,
				"ban_duration_seconds", int64(banDuration.Seconds()),
			),
			Reason: reason,
		}).Error; err != nil {
			return err
		}

		if !s.IsObserveMode() {
			if err := tx.Create(&model.AuditLogs{
				OperatorID: 0,
				Action:     model.AuditActionAutoBan,
				TargetType: "user",
				TargetID:   userID,
				OldValues:  oldValues,
				NewValues: jsonMap(
					"status", model.UserStatusBanned,
					"ban_until", banUntil,
					"ban_reason", reason,
					"ban_source", model.UserBanSourceSystem,
					"violation_count", nextCount,
				),
				Reason: reason,
			}).Error; err != nil {
				return err
			}
		}

		result = &BanDecision{
			Banned:         !s.IsObserveMode(),
			BanUntil:       banUntil,
			BanReason:      reason,
			BanSource:      model.UserBanSourceSystem,
			ViolationCount: nextCount,
			Permanent:      !s.IsObserveMode() && banUntil == nil,
		}
		return nil
	})

	return result, err
}

func (s *SecurityService) CountAbuseAndMaybeBan(
	ctx context.Context,
	rdb redis.UniversalClient,
	userID int64,
	scope, triggerKey, reason string,
	abuseLimit int64,
	window time.Duration,
	evidence map[string]interface{},
) (*BanDecision, int64, error) {
	if rdb == nil || userID <= 0 {
		return nil, 0, nil
	}
	key := fmt.Sprintf("abuse:%s:%d", triggerKey, userID)
	decision, err := s.RateLimit(ctx, rdb, key, abuseLimit, window)
	if err != nil {
		return nil, 0, err
	}
	if decision.Allowed {
		return nil, decision.Current, nil
	}
	banDecision, err := s.RecordViolation(userID, scope, triggerKey, reason, evidence)
	return banDecision, decision.Current, err
}

func violationBanDuration(level int) time.Duration {
	switch {
	case level <= 1:
		return 24 * time.Hour
	case level == 2:
		return 7 * 24 * time.Hour
	default:
		return 0
	}
}

func BuildRiskData(decision *BanDecision) map[string]interface{} {
	if decision == nil {
		return nil
	}
	data := map[string]interface{}{
		"banned":          decision.Banned,
		"ban_reason":      decision.BanReason,
		"ban_source":      decision.BanSource,
		"violation_count": decision.ViolationCount,
		"permanent":       decision.Permanent,
	}
	if decision.BanUntil != nil {
		data["ban_until"] = decision.BanUntil.Format(time.RFC3339)
	}
	return data
}

func BuildRateLimitData(retryAfter time.Duration, scope string) map[string]interface{} {
	data := map[string]interface{}{
		"scope":       scope,
		"retry_after": int(retryAfter.Seconds()),
	}
	if data["retry_after"].(int) < 1 {
		data["retry_after"] = 1
	}
	return data
}

func BanDecisionFromUser(user *model.Users) *BanDecision {
	if user == nil || user.Status != model.UserStatusBanned {
		return nil
	}
	return &BanDecision{
		Banned:         true,
		BanUntil:       user.BanUntil,
		BanReason:      user.BanReason,
		BanSource:      user.BanSource,
		ViolationCount: user.ViolationCount,
		Permanent:      user.BanUntil == nil,
	}
}

func BuildTriggerKey(parts ...string) string {
	filtered := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		filtered = append(filtered, part)
	}
	return strings.Join(filtered, ":")
}

func BuildRateLimitKey(scope string, parts ...string) string {
	return "ratelimit:" + BuildTriggerKey(append([]string{scope}, parts...)...)
}

func BuildViolationEvidence(pairs ...interface{}) map[string]interface{} {
	evidence := make(map[string]interface{}, len(pairs)/2)
	for i := 0; i+1 < len(pairs); i += 2 {
		key, ok := pairs[i].(string)
		if !ok {
			continue
		}
		evidence[key] = pairs[i+1]
	}
	return evidence
}

func RetryAfterSeconds(decision RateLimitDecision) string {
	seconds := int(decision.RetryAfter.Seconds())
	if seconds < 1 {
		seconds = 1
	}
	return strconv.Itoa(seconds)
}
