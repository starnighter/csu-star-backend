// Package mailer 负责所有出站邮件的构造与投递。
//
// 从 pkg/utils 独立出来的原因：邮件已经是一个有自己的配置、重试策略、MIME
// 构造和多通道编排的子系统，而 pkg/utils 是基础设施杂物间（redis/cos/snowflake）。
//
// 依赖方向：mailer 只 import config 和 logger。pkg/utils 反过来依赖 mailer
// （imap_client 需要 ReplyHeaderKey），所以 mailer 绝不能 import pkg/utils。
package mailer

import (
	"context"
	"errors"
)

// ReplyHeaderKey 标记由本系统发出的注册回信。IMAP 轮询器靠它跳过自己的回信，
// 否则 poller 会对自己的回信无限回复。
const ReplyHeaderKey = "X-CSU-Star-Reply"

// Kind 是投递通道的类型。它决定优先级：只要配置了任何一个云厂商通道，
// 自建/消费级 SMTP 就只作兜底。
type Kind string

const (
	// KindAliyunDM 阿里云邮件推送 DirectMail（SMTP 端点 smtpdm.aliyun.com:465）
	KindAliyunDM Kind = "aliyun_dm"
	// KindTencentSES 腾讯云邮件推送 SES（SMTP 端点 smtp.qcloudmail.com:465）
	KindTencentSES Kind = "tencent_ses"
	// KindCustomSMTP 管理员自填的普通 SMTP 邮箱
	KindCustomSMTP Kind = "custom_smtp"
)

// NormalizeKind 把任意输入收敛成已知的 Kind，未知值一律按自填 SMTP 处理。
func NormalizeKind(raw string) Kind {
	switch Kind(raw) {
	case KindAliyunDM:
		return KindAliyunDM
	case KindTencentSES:
		return KindTencentSES
	default:
		return KindCustomSMTP
	}
}

// IsCloud 报告该类型是否为云厂商的专业投递服务。
func (k Kind) IsCloud() bool {
	return k == KindAliyunDM || k == KindTencentSES
}

// rank 是通道类型的优先级，数字越小越优先。
// 这就是「配了 SES 就以 SES 为主」的实现点：任何云通道都排在自填 SMTP 之前，
// 与管理员填的 tier 无关，tier 只在同类型内部排序。
func (k Kind) rank() int {
	if k.IsCloud() {
		return 0
	}
	return 1
}

// Message 是一封待发送的邮件。HTMLBody 与 TextBody 二选一。
type Message struct {
	To           []string
	Subject      string
	HTMLBody     string
	TextBody     string
	ReplyTo      string
	ExtraHeaders map[string]string
}

// SendResult 描述一次成功投递。
type SendResult struct {
	// Channel 是通道显示名，用于日志。
	Channel string
	// MessageID 是我们自己生成的 Message-ID 头。
	// 阿里云/腾讯云控制台都支持按它检索投递记录，
	// 这补回了 SMTP 模式下拿不到厂商 MessageId 的可观测性缺口。
	MessageID string
}

// Sender 是一条投递通道。Pool 自身也实现 Sender，因此通道可以嵌套。
type Sender interface {
	Name() string
	Send(ctx context.Context, msg *Message) (*SendResult, error)
}

var (
	// ErrNoSenders 表示没有任何可用通道。
	ErrNoSenders = errors.New("mailer: 没有配置可用的邮件通道")
	// ErrAttemptsExhausted 表示达到 max_attempts 上限仍未成功。
	ErrAttemptsExhausted = errors.New("mailer: 已达最大尝试次数")
)
