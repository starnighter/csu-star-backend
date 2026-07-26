package mailer

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"csu-star-backend/config"
	"csu-star-backend/logger"

	"go.uber.org/zap"
)

const (
	defaultPerTryTimeout = 10 * time.Second
	defaultTotalBudget   = 20 * time.Second
	defaultMaxAttempts   = 3
)

// tierCursors 保存每一层的轮询游标。按「层序号」索引会在通道增删时错位，
// 所以用层的稳定标识（rank:tier）做 key。
var tierCursors sync.Map // map[string]*atomic.Uint64

func cursorFor(key string) *atomic.Uint64 {
	if v, ok := tierCursors.Load(key); ok {
		return v.(*atomic.Uint64)
	}
	v, _ := tierCursors.LoadOrStore(key, new(atomic.Uint64))
	return v.(*atomic.Uint64)
}

// Pool 按层依次尝试投递：同层内轮询，整层全败才降级到下一层。
//
// Pool 每次发送都重新构建（见 newPoolFromConfig），不能在启动时缓存：
// config.ReloadConfig 会整体替换 *Config，缓存住 Pool 会让 SIGHUP 对邮件失效。
type Pool struct {
	tiers       [][]Sender
	tierKeys    []string
	perTry      time.Duration
	budget      time.Duration
	maxAttempts int
}

func (p *Pool) Name() string { return "pool" }

// Send 依次尝试各层通道，返回第一次成功的结果。
//
// 与旧实现的关键差别：整体有墙钟预算且有尝试次数上限。旧实现会遍历全部
// provider、每个独享 10s 超时——13 个通道全挂就是 130 秒同步阻塞在 HTTP
// 请求里，而 server.WriteTimeout 只有 30 秒。
func (p *Pool) Send(ctx context.Context, msg *Message) (*SendResult, error) {
	if len(p.tiers) == 0 {
		return nil, ErrNoSenders
	}

	if p.budget > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, p.budget)
		defer cancel()
	}

	var errs []error
	attempts := 0

	for ti, tier := range p.tiers {
		if len(tier) == 0 {
			continue
		}
		cursor := cursorFor(p.tierKeys[ti])
		start := int(cursor.Add(1)-1) % len(tier)

		for offset := range tier {
			if p.maxAttempts > 0 && attempts >= p.maxAttempts {
				return nil, errors.Join(append(errs, ErrAttemptsExhausted)...)
			}
			if err := ctx.Err(); err != nil {
				return nil, errors.Join(append(errs, err)...)
			}

			sender := tier[(start+offset)%len(tier)]
			attempts++

			tryCtx, cancel := context.WithTimeout(ctx, p.perTry)
			res, err := sender.Send(tryCtx, msg)
			cancel()

			if err == nil {
				if logger.Log != nil {
					logger.Log.Info("邮件发送成功",
						zap.String("channel", sender.Name()),
						zap.Int("tier", ti),
						zap.Int("attempts", attempts),
						zap.String("message_id", res.MessageID),
						zap.Strings("to", msg.To))
				}
				return res, nil
			}

			if logger.Log != nil {
				logger.Log.Warn("邮件发送失败，尝试下一个通道",
					zap.String("channel", sender.Name()),
					zap.Int("tier", ti),
					zap.Error(err))
			}
			errs = append(errs, fmt.Errorf("%s: %w", sender.Name(), err))
		}
	}

	if len(errs) == 0 {
		return nil, ErrNoSenders
	}
	joined := errors.Join(errs...)
	if logger.Log != nil {
		logger.Log.Error("邮件所有通道均发送失败", zap.Strings("to", msg.To), zap.Error(joined))
	}
	return nil, joined
}

// newPool 用给定的通道列表构建分层池，预算取自 mail.delivery。
func newPool(providers []Provider) *Pool {
	groups := groupIntoTiers(providers)

	pool := &Pool{
		tiers:       make([][]Sender, 0, len(groups)),
		tierKeys:    make([]string, 0, len(groups)),
		perTry:      defaultPerTryTimeout,
		budget:      defaultTotalBudget,
		maxAttempts: defaultMaxAttempts,
	}

	if cfg := config.GetConfig(); cfg != nil {
		d := cfg.Mail.Delivery
		if d.PerTryTimeoutSec > 0 {
			pool.perTry = time.Duration(d.PerTryTimeoutSec) * time.Second
		}
		if d.TotalBudgetSec > 0 {
			pool.budget = time.Duration(d.TotalBudgetSec) * time.Second
		}
		if d.MaxAttempts > 0 {
			pool.maxAttempts = d.MaxAttempts
		}
	}

	for _, group := range groups {
		senders := make([]Sender, 0, len(group))
		for _, p := range group {
			senders = append(senders, newSMTPSender(p))
		}
		pool.tiers = append(pool.tiers, senders)
		pool.tierKeys = append(pool.tierKeys, fmt.Sprintf("%d:%d", group[0].Kind.rank(), group[0].Tier))
	}
	return pool
}
