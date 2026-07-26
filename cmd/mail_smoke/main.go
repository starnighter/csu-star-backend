// mail_smoke 是邮件通道的冒烟测试工具。
//
// 它不会被部署脚本打包：deploy_backend_hot_update.sh 构建的是 ./cmd 而非 ./...，
// 所以这个 package main 永远不会进入服务端二进制。
//
//	go run ./cmd/mail_smoke -to you@example.com -dry-run
//	go run ./cmd/mail_smoke -to you@example.com -provider aliyun-directmail
//
// -dry-run 价值最高：直接把完整 MIME 打到 stdout，让你在不烧发送额度、
// 不等 DNS 生效的情况下肉眼确认 RFC 2047 主题、Date 和 Message-ID。
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"

	"csu-star-backend/config"
	"csu-star-backend/logger"
	"csu-star-backend/pkg/mailer"
)

func main() {
	var (
		to        = flag.String("to", "", "收件地址（必填）")
		provider  = flag.String("provider", "", "只用指定名字的通道，留空表示走整个分层池")
		kind      = flag.String("kind", "verification", "邮件类型：verification | reply")
		code      = flag.String("code", "123456", "验证码内容")
		configDir = flag.String("config", ".", "配置文件所在目录")
		dryRun    = flag.Bool("dry-run", false, "只渲染报文并打印，不联网发送")
		listOnly  = flag.Bool("list", false, "只列出当前生效的通道后退出")
	)
	flag.Parse()

	logger.Init()

	// config.Init 写死了 viper.AddConfigPath(".")，切目录比改 config.go 更省事。
	if err := os.Chdir(*configDir); err != nil {
		fatalf("切换到配置目录失败: %v", err)
	}
	if err := config.Init(); err != nil {
		fatalf("加载配置失败: %v", err)
	}

	providers := mailer.Providers()
	if *provider != "" {
		filtered := providers[:0:0]
		for _, p := range providers {
			if strings.EqualFold(p.DisplayName(), *provider) {
				filtered = append(filtered, p)
			}
		}
		if len(filtered) == 0 {
			fatalf("没有找到名为 %q 的可用通道", *provider)
		}
		providers = filtered
	}

	fmt.Printf("当前生效通道（%d 条，按优先级）：\n", len(providers))
	for _, p := range providers {
		cloud := ""
		if p.Kind.IsCloud() {
			cloud = "  [云通道]"
		}
		fmt.Printf("  - %-24s %s:%d  kind=%s tier=%d%s\n",
			p.DisplayName(), p.Host, p.Port, p.Kind, p.Tier, cloud)
	}

	if errs, warns := mailer.ValidateConfig(); len(errs) > 0 || len(warns) > 0 {
		fmt.Println()
		for _, m := range errs {
			fmt.Println("  [错误] " + m)
		}
		for _, m := range warns {
			fmt.Println("  [提醒] " + m)
		}
	}

	if *listOnly {
		return
	}
	if strings.TrimSpace(*to) == "" {
		fatalf("必须用 -to 指定收件地址")
	}

	if *dryRun {
		fmt.Printf("\n===== DRY RUN：以下是将要投递的完整报文 =====\n\n")
		raw, messageID, err := mailer.RenderForSmoke(providers[0], *kind, *code, *to)
		if err != nil {
			fatalf("渲染报文失败: %v", err)
		}
		fmt.Println(string(raw))
		fmt.Printf("\n===== Message-ID: %s =====\n", messageID)
		return
	}

	fmt.Printf("\n正在发送 %s 邮件到 %s ...\n", *kind, *to)
	var err error
	if *provider != "" {
		err = mailer.SendTestEmail(context.Background(), providers[0], *to)
	} else if *kind == "reply" {
		err = mailer.SendRegistrationReplyEmail(*to, true)
	} else {
		err = mailer.SendVerificationEmail([]string{*to}, *code)
	}
	if err != nil {
		fatalf("发送失败: %v", err)
	}
	fmt.Println("发送成功，请查收。")
}

func fatalf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
