package mailer

import (
	"sort"
	"strings"
	"sync"

	"csu-star-backend/config"
)

// Provider 是一条投递通道的完整配置，与来源（数据库或 config.yaml）无关。
type Provider struct {
	Name          string
	Kind          Kind
	Host          string
	Port          int
	TLSMode       string
	Username      string
	Password      string
	FromEmailAddr string
	FromName      string
	// Tier 只在同一 Kind 内部排序；跨 Kind 的优先级由 Kind.rank() 决定。
	Tier     int
	Disabled bool
}

// Complete 报告这条配置是否足以发信。缺字段的通道会被静默跳过。
func (p Provider) Complete() bool {
	return strings.TrimSpace(p.Host) != "" &&
		p.Port != 0 &&
		strings.TrimSpace(p.Username) != "" &&
		strings.TrimSpace(p.Password) != "" &&
		strings.TrimSpace(p.FromEmailAddr) != ""
}

// DisplayName 返回日志里用的通道名。
func (p Provider) DisplayName() string {
	if name := strings.TrimSpace(p.Name); name != "" {
		return name
	}
	if from := strings.TrimSpace(p.FromEmailAddr); from != "" {
		return from
	}
	return p.Host
}

// ProviderSource 提供当前生效的通道列表。
// 生产环境由数据库实现（管理端可配），未注册时回落到 config.yaml。
type ProviderSource func() []Provider

var (
	sourceMu       sync.RWMutex
	providerSource ProviderSource
)

// SetProviderSource 注册通道来源。传 nil 可恢复为只读 config.yaml。
// 由 main 在数据库就绪后调用。
func SetProviderSource(src ProviderSource) {
	sourceMu.Lock()
	providerSource = src
	sourceMu.Unlock()
}

// Providers 返回当前生效的、配置完整且未停用的通道。
//
// 优先使用注册的来源（数据库）；来源为空或没有可用通道时回落到 config.yaml。
// 这个回落不是冗余：它让数据库尚未初始化、管理员还没配过任何通道时，
// 现有的 YAML 通道继续工作，从而允许「先发二进制、后配通道」的部署顺序。
func Providers() []Provider {
	sourceMu.RLock()
	src := providerSource
	sourceMu.RUnlock()

	if src != nil {
		if list := filterUsable(src()); len(list) > 0 {
			return list
		}
	}
	return filterUsable(configProviders())
}

func filterUsable(in []Provider) []Provider {
	out := make([]Provider, 0, len(in))
	for _, p := range in {
		if p.Disabled || !p.Complete() {
			continue
		}
		p.Kind = NormalizeKind(string(p.Kind))
		out = append(out, p)
	}
	return out
}

// configProviders 读取 config.yaml 里的 mail.verification.providers。
// 未写 kind 的条目按自填 SMTP 处理——存量条目都是 163/126 消费级邮箱，
// 正确的语义就是「云通道之后的兜底」。
func configProviders() []Provider {
	cfg := config.GetConfig()
	if cfg == nil {
		return nil
	}

	out := make([]Provider, 0, len(cfg.Mail.Verification.Providers))
	for _, p := range cfg.Mail.Verification.Providers {
		out = append(out, Provider{
			Name:          p.Name,
			Kind:          NormalizeKind(p.Kind),
			Host:          p.Host,
			Port:          p.Port,
			TLSMode:       p.TLSMode,
			Username:      p.Username,
			Password:      p.Password,
			FromEmailAddr: p.FromEmailAddr,
			FromName:      p.FromName,
			Tier:          p.Tier,
			Disabled:      p.Disabled,
		})
	}
	return out
}

// groupIntoTiers 把通道按 (Kind.rank, Tier) 分组排序，返回从高优先级到低优先级
// 的分层列表。同一层内部由调用方轮询；只有整层全部失败才降级到下一层。
//
// Kind.rank 排在 Tier 前面，这就是「有 SES 就以 SES 为主」：管理员即使给自填
// 邮箱设了 tier 0，也不会抢在云通道前面。
func groupIntoTiers(providers []Provider) [][]Provider {
	if len(providers) == 0 {
		return nil
	}

	sorted := make([]Provider, len(providers))
	copy(sorted, providers)
	sort.SliceStable(sorted, func(i, j int) bool {
		if ri, rj := sorted[i].Kind.rank(), sorted[j].Kind.rank(); ri != rj {
			return ri < rj
		}
		return sorted[i].Tier < sorted[j].Tier
	})

	var tiers [][]Provider
	for i, p := range sorted {
		sameGroup := i > 0 &&
			sorted[i-1].Kind.rank() == p.Kind.rank() &&
			sorted[i-1].Tier == p.Tier
		if sameGroup {
			tiers[len(tiers)-1] = append(tiers[len(tiers)-1], p)
			continue
		}
		tiers = append(tiers, []Provider{p})
	}
	return tiers
}

// ReplyProviders 返回可用于发送注册回信的通道，排除 IMAP 收信箱本身，
// 避免回信被自己的轮询器再次读取。
func ReplyProviders() []Provider {
	cfg := config.GetConfig()
	imapUser := ""
	if cfg != nil {
		imapUser = strings.TrimSpace(strings.ToLower(cfg.Mail.Imap.Username))
	}

	all := Providers()
	if imapUser == "" {
		return all
	}

	out := make([]Provider, 0, len(all))
	for _, p := range all {
		if strings.EqualFold(strings.TrimSpace(p.Username), imapUser) {
			continue
		}
		out = append(out, p)
	}
	return out
}
