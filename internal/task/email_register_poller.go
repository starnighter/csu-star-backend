package task

import (
	"context"
	"fmt"
	"time"

	"csu-star-backend/config"
	"csu-star-backend/logger"
	"csu-star-backend/pkg/utils"

	"go.uber.org/zap"
)

const (
	defaultPollInterval = 30 * time.Minute
	minReconnectBackoff = 5 * time.Second
	maxReconnectBackoff = 5 * time.Minute
	idleStopTimeout     = 30 * time.Second
	initialConnectDelay = 5 * time.Second
	maxIdleRefresh      = 29 * time.Minute // RFC 3501 要求服务器至少每 29 分钟响应一次 IDLE
)

// EmailRegisterProcessor defines the interface for processing registration emails.
// This avoids an import cycle with the service package.
type EmailRegisterProcessor interface {
	ProcessRegistrationEmail(senderEmail, subject, body string) (bool, error)
}

type EmailRegisterPoller struct {
	processor EmailRegisterProcessor
}

func NewEmailRegisterPoller(processor EmailRegisterProcessor) *EmailRegisterPoller {
	return &EmailRegisterPoller{processor: processor}
}

func (p *EmailRegisterPoller) Start(ctx context.Context) {
	go p.run(ctx)
}

func (p *EmailRegisterPoller) run(ctx context.Context) {
	logger.Log.Info("邮箱注册IDLE服务启动")

	// 初始延迟，等待服务完全就绪
	select {
	case <-ctx.Done():
		return
	case <-time.After(initialConnectDelay):
	}

	backoff := minReconnectBackoff

	for {
		select {
		case <-ctx.Done():
			logger.Log.Info("邮箱注册IDLE服务停止")
			return
		default:
		}

		imapClient, err := p.connectWithBackoff(ctx, backoff)
		if err != nil {
			return // ctx 已取消
		}

		var loopErr error
		if imapClient.IsIdleSupported() {
			logger.Log.Info("IMAP服务器支持IDLE，进入长连接模式")
			loopErr = p.idleLoop(ctx, imapClient)
		} else {
			logger.Log.Warn("IMAP服务器不支持IDLE，回退到轮询模式")
			loopErr = p.pollLoop(ctx, imapClient)
		}

		_ = imapClient.Close()

		if loopErr == nil {
			return // 正常关闭
		}

		logger.Log.Error("IDLE循环异常，准备重连", zap.Error(loopErr))
		backoff = p.nextBackoff(backoff)
	}
}

func (p *EmailRegisterPoller) connectWithBackoff(ctx context.Context, initialBackoff time.Duration) (*utils.ImapClient, error) {
	backoff := initialBackoff

	for {
		cfg := config.GetConfig()
		if cfg == nil || !cfg.Mail.EmailRegister.Enabled || cfg.Mail.Imap.Host == "" {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(backoff):
			}
			continue
		}

		imapCfg := cfg.Mail.Imap
		imapClient, err := utils.NewImapClient(imapCfg.Host, imapCfg.Port, imapCfg.Username, imapCfg.Password)
		if err != nil {
			logger.Log.Error("IMAP连接失败，稍后重试",
				zap.Error(err),
				zap.Duration("backoff", backoff),
			)
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(backoff):
			}
			backoff = p.nextBackoff(backoff)
			continue
		}

		return imapClient, nil
	}
}

func (p *EmailRegisterPoller) nextBackoff(current time.Duration) time.Duration {
	next := current * 2
	if next > maxReconnectBackoff {
		next = maxReconnectBackoff
	}
	return next
}

func (p *EmailRegisterPoller) idleLoop(ctx context.Context, imapClient *utils.ImapClient) error {
	// 首次拉取并处理
	if err := p.fetchProcessDelete(imapClient); err != nil {
		return err
	}

	refreshInterval := p.idleRefreshInterval()

	for {
		select {
		case <-ctx.Done():
			return nil
		default:
		}

		// 进入 IDLE
		stop := make(chan struct{})
		idleErrCh := make(chan error, 1)
		go func() {
			idleErrCh <- imapClient.Idle(stop)
		}()

		refreshTimer := time.NewTimer(refreshInterval)

		select {
		case <-ctx.Done():
			if !refreshTimer.Stop() {
				<-refreshTimer.C
			}
			return p.stopIdle(ctx, stop, idleErrCh)

		case <-refreshTimer.C:
			// 定时刷新：退出 IDLE，主动检查邮件
			if err := p.stopIdle(ctx, stop, idleErrCh); err != nil {
				return err
			}
			if err := p.fetchProcessDelete(imapClient); err != nil {
				return err
			}

		case idleErr := <-idleErrCh:
			// 服务器通知（新邮件到达）或连接异常
			if !refreshTimer.Stop() {
				<-refreshTimer.C
			}
			if idleErr != nil {
				return idleErr
			}
			if err := p.fetchProcessDelete(imapClient); err != nil {
				return err
			}
		}
	}
}

func (p *EmailRegisterPoller) stopIdle(ctx context.Context, stop chan struct{}, idleErrCh <-chan error) error {
	close(stop)

	select {
	case err := <-idleErrCh:
		return err
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(idleStopTimeout):
		return fmt.Errorf("IDLE did not respond to DONE within %v", idleStopTimeout)
	}
}

func (p *EmailRegisterPoller) fetchProcessDelete(imapClient *utils.ImapClient) error {
	cfg := config.GetConfig()
	if cfg == nil || !cfg.Mail.EmailRegister.Enabled {
		return nil
	}

	messages, err := imapClient.FetchUnseenMessages()
	if err != nil {
		return fmt.Errorf("fetch unseen: %w", err)
	}

	if len(messages) == 0 {
		return nil
	}

	logger.Log.Info("发现未读邮件", zap.Int("count", len(messages)))

	var uidsToDelete []uint32
	for _, msg := range messages {
		// Skip reply emails to prevent bounce loops
		if msg.IsReply {
			logger.Log.Debug("跳过回复邮件", zap.String("from", msg.From), zap.Uint32("uid", msg.UID))
			uidsToDelete = append(uidsToDelete, msg.UID)
			continue
		}

		logger.Log.Debug("处理注册邮件",
			zap.String("from", msg.From),
			zap.String("subject", msg.Subject),
		)

		registered, err := p.processor.ProcessRegistrationEmail(msg.From, msg.Subject, msg.Body)
		if err != nil {
			// Processing failed — do NOT delete, retry on next cycle
			logger.Log.Error("处理注册邮件异常（保留重试）",
				zap.String("from", msg.From),
				zap.Uint32("uid", msg.UID),
				zap.Error(err),
			)
			continue
		}

		// Successfully processed (registered or already exists) — safe to delete
		uidsToDelete = append(uidsToDelete, msg.UID)
		if registered {
			logger.Log.Info("通过邮件注册新用户", zap.String("email", msg.From))
		}
	}

	if len(uidsToDelete) > 0 {
		if err := imapClient.DeleteMessages(uidsToDelete); err != nil {
			logger.Log.Warn("IMAP删除邮件失败", zap.Error(err))
		}
	}

	return nil
}

func (p *EmailRegisterPoller) pollLoop(ctx context.Context, imapClient *utils.ImapClient) error {
	interval := p.pollInterval()
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	if err := p.fetchProcessDelete(imapClient); err != nil {
		return err
	}

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			newInterval := p.pollInterval()
			if newInterval != interval {
				ticker.Reset(newInterval)
				interval = newInterval
			}
			if err := p.fetchProcessDelete(imapClient); err != nil {
				return err
			}
		}
	}
}

func (p *EmailRegisterPoller) idleRefreshInterval() time.Duration {
	cfg := config.GetConfig()
	if cfg != nil && cfg.Mail.EmailRegister.IdleRefreshSec > 0 {
		d := time.Duration(cfg.Mail.EmailRegister.IdleRefreshSec) * time.Second
		if d > maxIdleRefresh {
			d = maxIdleRefresh
		}
		return d
	}
	interval := p.pollInterval()
	if interval > maxIdleRefresh {
		interval = maxIdleRefresh
	}
	return interval
}

func (p *EmailRegisterPoller) pollInterval() time.Duration {
	cfg := config.GetConfig()
	if cfg != nil && cfg.Mail.EmailRegister.PollIntervalSec > 0 {
		return time.Duration(cfg.Mail.EmailRegister.PollIntervalSec) * time.Second
	}
	return defaultPollInterval
}
