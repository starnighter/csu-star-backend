package mailer

import (
	"fmt"
	"strings"

	"csu-star-backend/config"
)

// consumerMailHosts 是消费级邮箱的 SMTP 主机。生产环境把它们当主通道
// 意味着在绕发送限额，值得在启动时提醒。
var consumerMailHosts = map[string]bool{
	"smtp.163.com":   true,
	"smtp.126.com":   true,
	"smtp.qq.com":    true,
	"smtp.gmail.com": true,
	"smtp.sina.com":  true,
}

// cloudRelayHosts 是云厂商的 SMTP 中继端点。
var cloudRelayHosts = map[string]Kind{
	"smtpdm.aliyun.com":                KindAliyunDM,
	"smtpdm-ap-southeast-1.aliyun.com": KindAliyunDM,
	"smtp.qcloudmail.com":              KindTencentSES,
}

// ValidateConfig 检查邮件配置，返回致命错误与警告。
//
// 刻意保持非致命：邮件不可用不应该让 API 停止提供读服务。调用方（main）
// 只记日志。这里抓的都是「凌晨两点才会发现」的那类误配。
func ValidateConfig() (errs []string, warns []string) {
	providers := Providers()
	if len(providers) == 0 {
		return []string{"没有配置任何可用的邮件通道，验证码邮件将全部失败"}, nil
	}

	hasCloud := false
	for _, p := range providers {
		if p.Kind.IsCloud() {
			hasCloud = true
		}

		name := p.DisplayName()
		host := strings.ToLower(strings.TrimSpace(p.Host))

		if consumerMailHosts[host] {
			warns = append(warns, fmt.Sprintf("通道 %s 使用消费级邮箱 %s，仅适合作为兜底", name, host))
		}

		// 阿里云/腾讯云都要求信封 From 等于已认证的发信地址，
		// 不一致会在 DATA 阶段被拒。这是最常见的一类误配。
		if kind, ok := cloudRelayHosts[host]; ok {
			if !strings.EqualFold(strings.TrimSpace(p.FromEmailAddr), strings.TrimSpace(p.Username)) {
				warns = append(warns, fmt.Sprintf("通道 %s 的发信地址与认证账号不一致，%s 会拒收", name, kind))
			}
			if p.Kind == KindCustomSMTP {
				warns = append(warns, fmt.Sprintf("通道 %s 指向云厂商端点 %s，但类型标为自填 SMTP，将不会享有云通道优先级", name, host))
			}
		}

		if p.Port != 25 && p.Port != 465 && p.Port != 587 {
			warns = append(warns, fmt.Sprintf("通道 %s 使用了非常见端口 %d", name, p.Port))
		}
		if mode := strings.ToLower(strings.TrimSpace(p.TLSMode)); mode != "" &&
			mode != TLSModeImplicit && mode != TLSModeStartTLS {
			errs = append(errs, fmt.Sprintf("通道 %s 的 tls_mode=%q 不合法", name, p.TLSMode))
		}
		if !validAddress(p.FromEmailAddr) {
			errs = append(errs, fmt.Sprintf("通道 %s 的发信地址 %q 不是合法邮箱", name, p.FromEmailAddr))
		}
	}

	if !hasCloud {
		warns = append(warns, "未配置任何云厂商邮件通道，当前全部依赖自填 SMTP 邮箱")
	}

	// 整体投递预算必须小于 HTTP 写超时，否则整池故障会把请求拖过写超时，
	// 客户端已断开而 goroutine 还在拨号。
	if cfg := config.GetConfig(); cfg != nil {
		budget := cfg.Mail.Delivery.TotalBudgetSec
		if budget <= 0 {
			budget = int(defaultTotalBudget.Seconds())
		}
		if write := cfg.Server.WriteTimeoutSec; write > 0 && budget >= write {
			warns = append(warns, fmt.Sprintf(
				"mail.delivery.total_budget_sec=%d 不小于 server.write_timeout_sec=%d，整池故障时请求会超时",
				budget, write))
		}
	}

	return errs, warns
}
