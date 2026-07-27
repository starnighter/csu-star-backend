// Package emailpolicy 负责账号邮箱的规范化与域名准入判断。
//
// 它只依赖 config 和 internal/constant，故意不放在 pkg/utils（那是基础设施层）
// 也不放在 internal/service（auth / admin 等多处服务都会用到）。
//
// 两条必须遵守的约束：
//  1. 所有判断都实时读 config.GetConfig()，绝不在包级缓存，否则 SIGHUP 热重载失效。
//  2. config 为 nil 时一律放行。单元测试不会调用 config.Init()，fail-open 是让
//     现有 service 测试不 panic 的前提，同时也保证配置写错不会把所有人挡在门外。
package emailpolicy

import (
	"regexp"
	"strings"

	"csu-star-backend/config"
	"csu-star-backend/internal/constant"

	"net/mail"
)

const (
	// ModeAllowAll 只校验邮箱格式，任意域名均可注册。
	ModeAllowAll = "allow_all"
	// ModeAllowList 额外要求域名命中白名单。
	ModeAllowList = "allow_list"
)

const (
	maxEmailLen  = 254
	maxLocalLen  = 64
	maxDomainLen = 253
)

// domainRe 收紧 net/mail.ParseAddress 的宽松之处：要求至少一个点、
// 每段以字母数字开头结尾、拒绝 IP 字面量与连续点。
var domainRe = regexp.MustCompile(`^[a-z0-9]([a-z0-9-]*[a-z0-9])?(\.[a-z0-9]([a-z0-9-]*[a-z0-9])?)+$`)

// Normalize 去空格、转小写并做语法校验，返回入库使用的规范形式。
//
// 注意：local part 一并转小写严格来说不符合 RFC 5321，但现有库里所有邮箱都是
// 这样写进去的（每条写路径都过了旧的 normalizeSchoolEmail），且所有真实邮件
// 服务商都按大小写不敏感处理。此处保持现状，改动会让存量数据对不上。
func Normalize(raw string) (string, error) {
	s := strings.ToLower(strings.TrimSpace(raw))
	if s == "" || len(s) > maxEmailLen {
		return "", &constant.InvalidEmailFormatErr
	}

	addr, err := mail.ParseAddress(s)
	if err != nil || addr.Name != "" {
		// addr.Name != "" 用于拒绝 `Foo <a@b.com>` 这种带显示名的形式。
		return "", &constant.InvalidEmailFormatErr
	}
	s = strings.ToLower(addr.Address)

	at := strings.LastIndex(s, "@")
	if at <= 0 || at == len(s)-1 {
		return "", &constant.InvalidEmailFormatErr
	}
	local, domain := s[:at], s[at+1:]
	if len(local) > maxLocalLen || len(domain) > maxDomainLen || !domainRe.MatchString(domain) {
		return "", &constant.InvalidEmailFormatErr
	}
	return s, nil
}

// Domain 返回已规范化邮箱的域名部分。
func Domain(normalized string) string {
	if i := strings.LastIndex(normalized, "@"); i >= 0 {
		return normalized[i+1:]
	}
	return ""
}

// Check = Normalize + 域名策略，供账号创建/变更路径（注册、绑定邮箱）使用。
//
// 登录、找回密码、校验验证码等路径应只调 Normalize：将来收紧白名单时，
// 域名已不在名单内的存量用户仍必须能登录和取回自己的数据。
func Check(raw string) (string, error) {
	email, err := Normalize(raw)
	if err != nil {
		return "", err
	}
	if !DomainAllowed(Domain(email)) {
		return "", &constant.EmailDomainNotAllowedErr
	}
	return email, nil
}

// DomainAllowed 判断域名是否被账号策略接受。config 为 nil 时放行。
func DomainAllowed(domain string) bool {
	cfg := config.GetConfig()
	if cfg == nil {
		return true
	}
	p := cfg.Mail.AccountEmail

	if MatchAny(domain, p.BlockedDomains) {
		return false
	}
	if !strings.EqualFold(strings.TrimSpace(p.Policy), ModeAllowList) {
		// allow_all，以及任何无法识别的取值，都放行：
		// SIGHUP 之后配置里的一个错别字不应该把所有人挡在门外。
		return true
	}
	if len(p.AllowedDomains) == 0 {
		// allow_list 却没给名单属于误配，同样 fail-open。
		return true
	}
	return MatchAny(domain, p.AllowedDomains)
}

// MatchAny 判断域名是否命中列表。
// 条目大小写不敏感；容忍写成 "@example.com" 的形式；
// 前导点（".example.com"）表示该域名及其所有子域。
func MatchAny(domain string, list []string) bool {
	domain = strings.ToLower(strings.TrimSpace(domain))
	if domain == "" {
		return false
	}
	for _, raw := range list {
		entry := strings.ToLower(strings.TrimSpace(raw))
		entry = strings.TrimPrefix(entry, "@")
		if entry == "" {
			continue
		}
		if strings.HasPrefix(entry, ".") {
			bare := entry[1:]
			if domain == bare || strings.HasSuffix(domain, entry) {
				return true
			}
			continue
		}
		if domain == entry {
			return true
		}
	}
	return false
}

// AllowedDomainsSnapshot 供公开的 /email-policy 接口使用，
// 让前端不必硬编码支持的邮箱列表。
func AllowedDomainsSnapshot() (mode string, domains []string) {
	cfg := config.GetConfig()
	if cfg == nil {
		return ModeAllowAll, nil
	}
	p := cfg.Mail.AccountEmail
	if !strings.EqualFold(strings.TrimSpace(p.Policy), ModeAllowList) || len(p.AllowedDomains) == 0 {
		return ModeAllowAll, nil
	}

	domains = make([]string, 0, len(p.AllowedDomains))
	for _, raw := range p.AllowedDomains {
		entry := strings.TrimPrefix(strings.ToLower(strings.TrimSpace(raw)), "@")
		if entry != "" {
			domains = append(domains, entry)
		}
	}
	return ModeAllowList, domains
}
