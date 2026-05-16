package task

import (
	"context"
	"sync/atomic"
	"time"

	"csu-star-backend/config"
	"csu-star-backend/logger"
	"csu-star-backend/pkg/utils"

	"go.uber.org/zap"
)

const defaultPollInterval = 30 * time.Second

// EmailRegisterProcessor defines the interface for processing registration emails.
// This avoids an import cycle with the service package.
// The password is extracted from the email subject.
type EmailRegisterProcessor interface {
	ProcessRegistrationEmail(senderEmail, subject string) (bool, error)
}

type EmailRegisterPoller struct {
	processor EmailRegisterProcessor
	running   atomic.Bool
}

func NewEmailRegisterPoller(processor EmailRegisterProcessor) *EmailRegisterPoller {
	return &EmailRegisterPoller{processor: processor}
}

func (p *EmailRegisterPoller) Start(ctx context.Context) {
	go p.run(ctx)
}

func (p *EmailRegisterPoller) run(ctx context.Context) {
	logger.Log.Info("邮箱注册轮询服务启动")

	interval := p.pollInterval()
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	// Run once immediately after a short delay
	select {
	case <-ctx.Done():
		return
	case <-time.After(5 * time.Second):
	}
	p.poll()

	for {
		select {
		case <-ctx.Done():
			logger.Log.Info("邮箱注册轮询服务停止")
			return
		case <-ticker.C:
			newInterval := p.pollInterval()
			if newInterval != interval {
				ticker.Reset(newInterval)
				interval = newInterval
			}
			p.poll()
		}
	}
}

func (p *EmailRegisterPoller) poll() {
	if !p.running.CompareAndSwap(false, true) {
		logger.Log.Debug("跳过邮箱注册轮询：上一次仍在执行")
		return
	}
	defer p.running.Store(false)

	cfg := config.GlobalConfig
	if cfg == nil || !cfg.Mail.EmailRegister.Enabled {
		return
	}

	imapCfg := cfg.Mail.Imap
	if imapCfg.Host == "" {
		logger.Log.Warn("IMAP配置不完整，跳过轮询")
		return
	}

	// Connect to IMAP
	imapClient, err := utils.NewImapClient(imapCfg.Host, imapCfg.Port, imapCfg.Username, imapCfg.Password)
	if err != nil {
		logger.Log.Error("IMAP连接失败", zap.Error(err))
		return
	}
	defer imapClient.Close()

	// Fetch unseen messages
	messages, err := imapClient.FetchUnseenMessages()
	if err != nil {
		logger.Log.Error("IMAP获取未读邮件失败", zap.Error(err))
		return
	}

	if len(messages) == 0 {
		return
	}

	logger.Log.Info("发现未读注册邮件", zap.Int("count", len(messages)))

	var allUIDs []uint32
	for _, msg := range messages {
		allUIDs = append(allUIDs, msg.UID)

		logger.Log.Debug("处理注册邮件",
			zap.String("from", msg.From),
			zap.String("subject", msg.Subject),
		)

		registered, err := p.processor.ProcessRegistrationEmail(msg.From, msg.Subject)
		if err != nil {
			logger.Log.Error("处理注册邮件异常（已删除）",
				zap.String("from", msg.From),
				zap.Uint32("uid", msg.UID),
				zap.Error(err),
			)
		} else if registered {
			logger.Log.Info("通过邮件注册新用户", zap.String("email", msg.From))
		}
	}

	// 删除已处理邮件
	if len(allUIDs) > 0 {
		if err := imapClient.DeleteMessages(allUIDs); err != nil {
			logger.Log.Warn("IMAP删除邮件失败", zap.Error(err))
		}
	}
}

func (p *EmailRegisterPoller) pollInterval() time.Duration {
	cfg := config.GlobalConfig
	if cfg != nil && cfg.Mail.EmailRegister.PollIntervalSec > 0 {
		return time.Duration(cfg.Mail.EmailRegister.PollIntervalSec) * time.Second
	}
	return defaultPollInterval
}
