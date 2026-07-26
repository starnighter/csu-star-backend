package emailpolicy

import (
	"errors"
	"testing"

	"csu-star-backend/config"
	"csu-star-backend/internal/constant"
)

func TestNormalize(t *testing.T) {
	valid := []struct{ in, want string }{
		{" A@B.COM ", "a@b.com"},
		{"12345@csu.edu.cn", "12345@csu.edu.cn"},
		{"User.Name+tag@QQ.com", "user.name+tag@qq.com"},
		{"a@mail.csu.edu.cn", "a@mail.csu.edu.cn"},
	}
	for _, tc := range valid {
		got, err := Normalize(tc.in)
		if err != nil {
			t.Fatalf("Normalize(%q) error = %v", tc.in, err)
		}
		if got != tc.want {
			t.Fatalf("Normalize(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}

	invalid := []string{
		"",
		"   ",
		"a",
		"a@",
		"@b.com",
		"a@b",              // 没有点，net/mail 会放行，靠 domainRe 拦下
		"a@[10.0.0.1]",     // IP 字面量
		"Foo <a@b.com>",    // 带显示名
		"a@b..com",         // 连续点
		"a@-b.com",         // 段以连字符开头
		"a@b.com.",         // 尾随点
		string(make([]byte, 300)),
	}
	for _, in := range invalid {
		if _, err := Normalize(in); !errors.Is(err, &constant.InvalidEmailFormatErr) {
			t.Fatalf("Normalize(%q) expected format error, got %v", in, err)
		}
	}

	longLocal := ""
	for i := 0; i < 70; i++ {
		longLocal += "a"
	}
	if _, err := Normalize(longLocal + "@b.com"); !errors.Is(err, &constant.InvalidEmailFormatErr) {
		t.Fatal("expected local part over 64 chars to be rejected")
	}
}

// nil config 必须放行：service 层的单元测试从不调 config.Init()，
// 这里一旦 fail-closed，那些测试会全部报错。
func TestDomainAllowedWithNilConfig(t *testing.T) {
	config.SetConfig(nil)
	if !DomainAllowed("anything.com") {
		t.Fatal("expected nil config to allow all domains")
	}
}

func TestDomainAllowed(t *testing.T) {
	cases := []struct {
		name   string
		policy config.AccountEmailConfig
		domain string
		want   bool
	}{
		{"allow_all 放行任意域名", config.AccountEmailConfig{Policy: ModeAllowAll}, "qq.com", true},
		{"未知 policy 取值 fail-open", config.AccountEmailConfig{Policy: "typo_here"}, "qq.com", true},
		{"allow_list 空名单 fail-open",
			config.AccountEmailConfig{Policy: ModeAllowList}, "qq.com", true},
		{"allow_list 命中",
			config.AccountEmailConfig{Policy: ModeAllowList, AllowedDomains: []string{"csu.edu.cn"}}, "csu.edu.cn", true},
		{"allow_list 未命中",
			config.AccountEmailConfig{Policy: ModeAllowList, AllowedDomains: []string{"csu.edu.cn"}}, "qq.com", false},
		{"黑名单优先于白名单",
			config.AccountEmailConfig{
				Policy:         ModeAllowList,
				AllowedDomains: []string{"qq.com"},
				BlockedDomains: []string{"qq.com"},
			}, "qq.com", false},
		{"allow_all 下黑名单仍生效",
			config.AccountEmailConfig{Policy: ModeAllowAll, BlockedDomains: []string{"mailinator.com"}}, "mailinator.com", false},
		{"@ 前缀写法被容忍",
			config.AccountEmailConfig{Policy: ModeAllowList, AllowedDomains: []string{"@qq.com"}}, "qq.com", true},
		{"前导点匹配子域",
			config.AccountEmailConfig{Policy: ModeAllowList, AllowedDomains: []string{".csu.edu.cn"}}, "mail.csu.edu.cn", true},
		{"前导点也匹配裸域",
			config.AccountEmailConfig{Policy: ModeAllowList, AllowedDomains: []string{".csu.edu.cn"}}, "csu.edu.cn", true},
		{"后缀伪装不匹配",
			config.AccountEmailConfig{Policy: ModeAllowList, AllowedDomains: []string{".csu.edu.cn"}}, "csu.edu.cn.evil.com", false},
		{"大小写不敏感",
			config.AccountEmailConfig{Policy: ModeAllowList, AllowedDomains: []string{"CSU.EDU.CN"}}, "csu.edu.cn", true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			config.SetConfig(&config.Config{Mail: config.MailConfig{AccountEmail: tc.policy}})
			t.Cleanup(func() { config.SetConfig(nil) })

			if got := DomainAllowed(tc.domain); got != tc.want {
				t.Fatalf("DomainAllowed(%q) = %v, want %v", tc.domain, got, tc.want)
			}
		})
	}
}

// 格式错误必须优先于域名错误：否则用户会因为「域名不支持」而困惑于
// 一个根本不是邮箱的输入。
func TestCheckPrefersFormatError(t *testing.T) {
	config.SetConfig(&config.Config{Mail: config.MailConfig{
		AccountEmail: config.AccountEmailConfig{
			Policy:         ModeAllowList,
			AllowedDomains: []string{"csu.edu.cn"},
		},
	}})
	t.Cleanup(func() { config.SetConfig(nil) })

	if _, err := Check("not-an-email"); !errors.Is(err, &constant.InvalidEmailFormatErr) {
		t.Fatalf("expected format error, got %v", err)
	}
	if _, err := Check("a@qq.com"); !errors.Is(err, &constant.EmailDomainNotAllowedErr) {
		t.Fatalf("expected domain error, got %v", err)
	}
}

func TestAllowedDomainsSnapshot(t *testing.T) {
	config.SetConfig(nil)
	if mode, domains := AllowedDomainsSnapshot(); mode != ModeAllowAll || len(domains) != 0 {
		t.Fatalf("nil config: got %s %v", mode, domains)
	}

	config.SetConfig(&config.Config{Mail: config.MailConfig{
		AccountEmail: config.AccountEmailConfig{
			Policy:         ModeAllowList,
			AllowedDomains: []string{"@QQ.com", " csu.edu.cn "},
		},
	}})
	t.Cleanup(func() { config.SetConfig(nil) })

	mode, domains := AllowedDomainsSnapshot()
	if mode != ModeAllowList {
		t.Fatalf("mode = %s, want %s", mode, ModeAllowList)
	}
	if len(domains) != 2 || domains[0] != "qq.com" || domains[1] != "csu.edu.cn" {
		t.Fatalf("domains = %v", domains)
	}
}
