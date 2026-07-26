package mailer

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"csu-star-backend/config"
)

type stubSender struct {
	mu    sync.Mutex
	name  string
	err   error
	delay time.Duration
	calls int
}

func (s *stubSender) Name() string { return s.name }

func (s *stubSender) Send(ctx context.Context, msg *Message) (*SendResult, error) {
	s.mu.Lock()
	s.calls++
	s.mu.Unlock()

	if s.delay > 0 {
		select {
		case <-time.After(s.delay):
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	if s.err != nil {
		return nil, s.err
	}
	return &SendResult{Channel: s.name, MessageID: "<id@test>"}, nil
}

func (s *stubSender) callCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.calls
}

func testPool(tiers [][]Sender) *Pool {
	p := &Pool{
		tiers:       tiers,
		perTry:      time.Second,
		budget:      5 * time.Second,
		maxAttempts: 0,
	}
	for i := range tiers {
		p.tierKeys = append(p.tierKeys, "test-tier-"+strings.Repeat("x", i+1))
	}
	return p
}

func TestPoolPrefersLowerTier(t *testing.T) {
	primary := &stubSender{name: "cloud"}
	fallback := &stubSender{name: "legacy"}

	res, err := testPool([][]Sender{{primary}, {fallback}}).Send(context.Background(), &Message{To: []string{"a@b.com"}})
	if err != nil {
		t.Fatalf("Send() error = %v", err)
	}
	if res.Channel != "cloud" {
		t.Fatalf("Channel = %s, want cloud", res.Channel)
	}
	if fallback.callCount() != 0 {
		t.Fatal("fallback tier must not be touched when the primary tier succeeds")
	}
}

func TestPoolFallsThroughToHigherTier(t *testing.T) {
	boom := errors.New("boom")
	primary := &stubSender{name: "cloud", err: boom}
	fallback := &stubSender{name: "legacy"}

	res, err := testPool([][]Sender{{primary}, {fallback}}).Send(context.Background(), &Message{To: []string{"a@b.com"}})
	if err != nil {
		t.Fatalf("Send() error = %v", err)
	}
	if res.Channel != "legacy" {
		t.Fatalf("Channel = %s, want legacy", res.Channel)
	}
	if primary.callCount() != 1 {
		t.Fatalf("primary calls = %d, want 1", primary.callCount())
	}
}

func TestPoolRoundRobinsWithinTier(t *testing.T) {
	boom := errors.New("boom")
	a := &stubSender{name: "a", err: boom}
	b := &stubSender{name: "b", err: boom}
	c := &stubSender{name: "c", err: boom}

	// 一次调用会把整层走遍，每个 sender 恰好被试一次。
	pool := testPool([][]Sender{{a, b, c}})
	if _, err := pool.Send(context.Background(), &Message{To: []string{"a@b.com"}}); err == nil {
		t.Fatal("expected error when every sender fails")
	}
	for _, s := range []*stubSender{a, b, c} {
		if s.callCount() != 1 {
			t.Fatalf("%s calls = %d, want 1", s.name, s.callCount())
		}
	}
}

func TestPoolRespectsMaxAttempts(t *testing.T) {
	boom := errors.New("boom")
	senders := make([]Sender, 0, 13)
	stubs := make([]*stubSender, 0, 13)
	for i := 0; i < 13; i++ {
		s := &stubSender{name: "s", err: boom}
		stubs = append(stubs, s)
		senders = append(senders, s)
	}

	pool := testPool([][]Sender{senders})
	pool.maxAttempts = 3

	_, err := pool.Send(context.Background(), &Message{To: []string{"a@b.com"}})
	if !errors.Is(err, ErrAttemptsExhausted) {
		t.Fatalf("expected ErrAttemptsExhausted, got %v", err)
	}

	total := 0
	for _, s := range stubs {
		total += s.callCount()
	}
	if total != 3 {
		t.Fatalf("total attempts = %d, want 3", total)
	}
}

// 这是 130 秒同步阻塞那个 bug 的回归测试：
// 旧实现会遍历全部 13 个 provider，每个独享 10 秒超时，
// 远远超过 server.WriteTimeout。
func TestPoolRespectsTotalBudget(t *testing.T) {
	boom := errors.New("boom")
	senders := make([]Sender, 0, 20)
	stubs := make([]*stubSender, 0, 20)
	for i := 0; i < 20; i++ {
		s := &stubSender{name: "slow", err: boom, delay: 50 * time.Millisecond}
		stubs = append(stubs, s)
		senders = append(senders, s)
	}

	pool := testPool([][]Sender{senders})
	pool.budget = 120 * time.Millisecond
	pool.perTry = 50 * time.Millisecond

	start := time.Now()
	_, err := pool.Send(context.Background(), &Message{To: []string{"a@b.com"}})
	elapsed := time.Since(start)

	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected wrapped DeadlineExceeded, got %v", err)
	}
	if elapsed > 500*time.Millisecond {
		t.Fatalf("Send took %v, budget should have cut it short", elapsed)
	}

	total := 0
	for _, s := range stubs {
		total += s.callCount()
	}
	if total >= 20 {
		t.Fatalf("expected the budget to stop early, but all %d senders were tried", total)
	}
}

func TestPoolReturnsErrNoSendersWhenEmpty(t *testing.T) {
	if _, err := testPool(nil).Send(context.Background(), &Message{To: []string{"a@b.com"}}); !errors.Is(err, ErrNoSenders) {
		t.Fatalf("expected ErrNoSenders, got %v", err)
	}
}

// 「有 SES 就以 SES 为主」：云通道恒排在自填 SMTP 之前，
// 即使管理员给自填邮箱设了更小的 tier。
func TestGroupIntoTiersPutsCloudFirst(t *testing.T) {
	providers := []Provider{
		{Name: "legacy-163", Kind: KindCustomSMTP, Tier: 0},
		{Name: "aliyun", Kind: KindAliyunDM, Tier: 5},
		{Name: "tencent", Kind: KindTencentSES, Tier: 5},
		{Name: "self-hosted", Kind: KindCustomSMTP, Tier: 9},
	}

	tiers := groupIntoTiers(providers)
	if len(tiers) != 3 {
		t.Fatalf("tiers = %d, want 3", len(tiers))
	}
	if len(tiers[0]) != 2 {
		t.Fatalf("first tier size = %d, want 2 cloud channels", len(tiers[0]))
	}
	for _, p := range tiers[0] {
		if !p.Kind.IsCloud() {
			t.Fatalf("first tier contains non-cloud channel %s", p.Name)
		}
	}
	if tiers[1][0].Name != "legacy-163" {
		t.Fatalf("second tier = %s, want legacy-163", tiers[1][0].Name)
	}
	if tiers[2][0].Name != "self-hosted" {
		t.Fatalf("third tier = %s, want self-hosted", tiers[2][0].Name)
	}
}

func TestProvidersFallBackToConfigWhenSourceEmpty(t *testing.T) {
	config.SetConfig(&config.Config{Mail: config.MailConfig{
		Verification: config.VerificationMailConfig{
			Providers: []config.SMTPConfig{{
				Name: "163-legacy", Host: "smtp.163.com", Port: 465,
				Username: "u", Password: "p", FromEmailAddr: "u@163.com",
			}},
		},
	}})
	t.Cleanup(func() {
		config.SetConfig(nil)
		SetProviderSource(nil)
	})

	SetProviderSource(func() []Provider { return nil })

	got := Providers()
	if len(got) != 1 || got[0].Name != "163-legacy" {
		t.Fatalf("expected config fallback, got %+v", got)
	}
	if got[0].Kind != KindCustomSMTP {
		t.Fatalf("config providers should be custom_smtp, got %s", got[0].Kind)
	}
}

func TestProvidersSkipsIncompleteAndDisabled(t *testing.T) {
	t.Cleanup(func() {
		config.SetConfig(nil)
		SetProviderSource(nil)
	})
	config.SetConfig(&config.Config{})
	SetProviderSource(func() []Provider {
		return []Provider{
			{Name: "ok", Host: "h", Port: 465, Username: "u", Password: "p", FromEmailAddr: "u@h.com"},
			{Name: "no-password", Host: "h", Port: 465, Username: "u", FromEmailAddr: "u@h.com"},
			{Name: "disabled", Host: "h", Port: 465, Username: "u", Password: "p", FromEmailAddr: "u@h.com", Disabled: true},
		}
	})

	got := Providers()
	if len(got) != 1 || got[0].Name != "ok" {
		t.Fatalf("got %+v, want only the complete enabled provider", got)
	}
}
