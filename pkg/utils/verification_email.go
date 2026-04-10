package utils

import (
	"crypto/tls"
	"csu-star-backend/config"
	"csu-star-backend/logger"
	"errors"
	"fmt"
	"net"
	"net/smtp"
	"strings"

	"go.uber.org/zap"
)

const verificationEmailHTMLTemplate = `<html lang="zh-CN">
  <head>
    <meta charset="UTF-8"/>
    <meta/>
    <meta/>

    <title>CSU STAR | 验证码</title>
    <style type="text/css">
      @media only screen and (max-width: 600px) {
        .mobile-padding {
          padding: 24px 16px !important;
        }
        .mobile-header-padding {
          padding: 32px 20px !important;
        }
        .mobile-title {
          font-size: 28px !important;
        }
        .mobile-code {
          font-size: 32px !important;
          letter-spacing: 8px !important;
        }
        .mobile-cta {
          padding: 16px 20px !important;
          font-size: 14px !important;
          display: block !important;
          box-sizing: border-box !important;
        }
      }
    </style>
  </head>
  <body style="
      margin: 0;
      padding: 0;
      background-color: transparent;
      font-family:
        -apple-system, BlinkMacSystemFont, &quot;Segoe UI&quot;,
        &quot;Poppins&quot;, sans-serif;
    ">
    <div style="
        max-width: 600px;
        margin: 10px auto;
        background: linear-gradient(135deg, #fff 0%, #eff6ff 100%);
        border-radius: 8px;
        overflow: hidden;
        box-shadow: 0px 3px 10px rgba(0, 0, 0, 0.25);
      ">
      <div class="mobile-header-padding" style="padding: 20px 30px; position: relative; overflow: hidden">
        <div style="position: relative; z-index: 1">
          <div style="
              color: #1e40af;
              font-size: 14px;
              font-weight: 900;
              letter-spacing: 3px;
              margin-bottom: 16px;
            ">
            CSU STAR
          </div>
          <h1 class="mobile-title" style="
              margin: 0;
              color: #1a1a1a;
              font-size: 36px;
              font-weight: 900;
              line-height: 1.1;
              letter-spacing: -2px;
            ">
            邮箱验证
          </h1>
        </div>
      </div>

      <div class="mobile-padding" style="
          padding: 20px 30px;
          border-top: 1px dashed black;
          padding-top: 20px;
          position: relative;
        ">
        <img src="data:image/svg+xml;base64,PD94bWwgdmVyc2lvbj0iMS4wIiBlbmNvZGluZz0idXRmLTgiPz4NCjwhLS0gR2VuZXJhdG9yOiBBZG9iZSBJbGx1c3RyYXRvciAyMi4xLjAsIFNWRyBFeHBvcnQgUGx1Zy1JbiAuIFNWRyBWZXJzaW9uOiA2LjAwIEJ1aWxkIDApICAtLT4NCjxzdmcgdmVyc2lvbj0iMS4xIiBpZD0iTGF5ZXJfMSIgeG1sbnM9Imh0dHA6Ly93d3cudzMub3JnLzIwMDAvc3ZnIiB4bWxuczp4bGluaz0iaHR0cDovL3d3dy53My5vcmcvMTk5OS94bGluayIgeD0iMHB4IiB5PSIwcHgiDQoJIHZpZXdCb3g9IjAgMCAyNjMgMjIzIiBzdHlsZT0iZW5hYmxlLWJhY2tncm91bmQ6bmV3IDAgMCAyNjMgMjIzOyIgeG1sOnNwYWNlPSJwcmVzZXJ2ZSI+DQo8c3R5bGUgdHlwZT0idGV4dC9jc3MiPg0KCS5zdDB7ZmlsbC1ydWxlOmV2ZW5vZGQ7Y2xpcC1ydWxlOmV2ZW5vZGQ7ZmlsbDpub25lO3N0cm9rZTojMEYyODVBO3N0cm9rZS1taXRlcmxpbWl0OjEwO30NCgkuc3Qxe2ZpbGwtcnVsZTpldmVub2RkO2NsaXAtcnVsZTpldmVub2RkO2ZpbGw6IzBGMjg1QTt9DQoJLnN0MntmaWxsLXJ1bGU6ZXZlbm9kZDtjbGlwLXJ1bGU6ZXZlbm9kZDtmaWxsOiNGRkZGRkY7fQ0KPC9zdHlsZT4NCjxlbGxpcHNlIHRyYW5zZm9ybT0ibWF0cml4KDAuNzMwOSAtMC42ODI0IDAuNjgyNCAwLjczMDkgLTQxLjI1OTUgMTE4LjMxOTQpIiBjbGFzcz0ic3QwIiBjeD0iMTI5LjQiIGN5PSIxMTEuNSIgcng9IjEzOSIgcnk9IjMxIi8+DQo8Y2lyY2xlIGNsYXNzPSJzdDEiIGN4PSIxNzguNyIgY3k9IjM0LjEiIHI9IjcuNSIvPg0KPGNpcmNsZSBjbGFzcz0ic3QxIiBjeD0iMzIuMyIgY3k9IjIwOC42IiByPSI3LjUiLz4NCjxlbGxpcHNlIHRyYW5zZm9ybT0ibWF0cml4KDAuNjgyNCAtMC43MzA5IDAuNzMwOSAwLjY4MjQgLTM3Ljk0OTUgMTI4LjU2ODIpIiBjbGFzcz0ic3QwIiBjeD0iMTI5IiBjeT0iMTA4IiByeD0iMzEiIHJ5PSIxMzkiLz4NCjxjaXJjbGUgY2xhc3M9InN0MSIgY3g9IjQ4LjMiIGN5PSI2OC4zIiByPSI3LjUiLz4NCjxjaXJjbGUgY2xhc3M9InN0MSIgY3g9IjIyNi4yIiBjeT0iMjA1LjEiIHI9IjcuNSIvPg0KPHBhdGggY2xhc3M9InN0MiIgZD0iTTEyNC4yLDY2LjFsLTM4LDQ3bC05LjgtOWwtNy4zLTE0LjVsLTQ0LjQsNTUuM2MwLDAsMTQuMSw1LDE2LDVzMTUtMS41LDE1LTEuNWwxMC40LTIuOGw1LjYtNS42bDguNi01LjlsMy00DQoJbDEyLTguNUw3Mi43LDE0OWwtMS41LDEwLjFsMTMyLTUuNXYtMzMuOGwtMTQuNi0yNC41bC03LjItMTMuMWwtMTYuNywxNC45bC00LjgsNC44bC0xMC4zLTEybC00LDEuMUwxMjQuMiw2Ni4xeiIvPg0KPHBhdGggY2xhc3M9InN0MSIgZD0iTTkxLjQsMTE0LjdsLTAuMi02LjJsMzMuMi00MC4ybDE1LjQsMzAuM2wtMTUuMiwxNS4yTDEyMyw5Ni42bC04LjQsMTEuNGwtNC00LjJsLTkuMiwxMC44bDUuNy0xMS45bC04LjgsNy41DQoJbDE0LjktMjMuNUw5MS40LDExNC43TDkxLjQsMTE0Ljd6IE02OSw5NS4zbDYuMiwzNi45bC01LjctMTMuOGwtNi42LDIwLjJsLTguNi01LjVsLTkuMiw5LjVsMy4xLTE3LjZsLTE1LjgsMTAuOEw2OSw5NS4zTDY5LDk1LjN6DQoJIE0xNDUuNyw5MC45bC0yMS4zLTI4LjhsLTM2LjUsNDQuMkw3MC4xLDg1TDUuNCwxNTguNmw2Ny4zLTkuN2wzNy42LTM3LjFsNi42LDIyLjJsMzIuOC00MS4xbDEwLjYsMTFsMTguMi0xOC43bDEyLjcsMzcuNA0KCWwtMTYuOSw4LjFsLTIuMi0yOC4xTDE1MS40LDE0MWwtMjcuNyw4LjhsMzAuMS0zNi41bC0zLjctMTAuOGwtMTYuNSwyMi45bC0yLjYtMi45bC0xNS40LDE3LjZsLTYuOC0yMS44bC00LjQsMjUuN2wtNC4yLTIuMg0KCWwtNC4yLDMuN2wxLjEtOS43bC0xMS40LDE2LjVsLTE0LjUsNi42bDE4Mi4yLDEuOGwtNzQuMS04MC45TDE2MCwxMDAuMmwtMTAuMy0xNC4zTDE0NS43LDkwLjl6Ii8+DQo8L3N2Zz4NCg==" alt="" style="
            position: absolute;
            bottom: 0;
            right: 0;
            width: 280px;
            height: auto;
            opacity: 0.18;
            pointer-events: none;
          "/>
        <div style="margin-bottom: 30px">
          <p style="
              margin: 0 0 12px 0;
              color: #1a1a1a;
              font-size: 15px;
              line-height: 1.8;
            ">
            尊敬的用户，
          </p>
          <p style="margin: 0; color: #4a4a4a; font-size: 14px; line-height: 1.8">
            您作为湖南唯一985的学生，正在进行邮箱验证操作，请使用以下验证码完成操作：
          </p>
        </div>

        <div style="
            background: #fff;
            border-radius: 2px;
            padding: 32px 20px;
            text-align: center;
            margin-bottom: 30px;
            box-shadow: inset 0 2px 4px 0 rgba(0, 0, 0, 0.25);
          ">
          <div style="
              color: #2563eb;
              font-size: 11px;
              font-weight: 700;
              letter-spacing: 2px;
              margin-bottom: 12px;
              font-family: monospace;
            ">
            VERIFICATION CODE
          </div>
          <div class="mobile-code" style="
              color: #1e40af;
              font-size: 42px;
              font-weight: 900;
              font-family: monospace;
              letter-spacing: 12px;
              line-height: 1;
            ">
            {{code}}
          </div>
        </div>

        <div style="
            background: #fff;
            border-left: 4px solid #93c5fd;
            padding: 16px 20px;
            margin-bottom: 30px;
            box-shadow: 0 0 2px #ccc;
          ">
          <table cellpadding="0" cellspacing="0" border="0">
            <tr>
              <td style="vertical-align: top; padding-top: 2px">
              </td>
              <td style="padding-left: 12px">
                <div style="color: #4a4a4a; font-size: 13px; line-height: 1.6">
                  验证码有效期为<strong style="color: #2563eb"> 5 </strong>
                  分钟，请尽快使用。
                </div>
              </td>
            </tr>
          </table>
        </div>

        <div style="text-align: center; margin: 0 auto 30px auto">
          <a href="https://csustar.wiki/login" class="mobile-cta" style="
              display: inline-block;
              background: linear-gradient(135deg, #2563eb, #93c5fd);
              color: #fff;
              text-decoration: none;
              font-size: 15px;
              font-weight: 700;
              padding: 14px 40px;
              border-radius: 4px;
              letter-spacing: 1px;
            " rel="nofollow noopener" target="_blank">
            前往使用
          </a>
        </div>

        <div style="margin-bottom: 30px">
          <div style="
              color: #1a1a1a;
              font-size: 13px;
              font-weight: 700;
              margin-bottom: 16px;
              letter-spacing: 1px;
            ">
            // 安全提示
          </div>

          <table cellpadding="0" cellspacing="0" border="0" width="100%">
            <tr>
              <td style="padding-bottom: 10px">
                <table cellpadding="0" cellspacing="0" border="0">
                  <tr>
                    <td style="width: 20px; vertical-align: top; padding-top: 2px">
                    </td>
                    <td style="padding-left: 10px">
                      <div style="
                          color: #4a4a4a;
                          font-size: 13px;
                          line-height: 1.6;
                        ">
                        请勿将验证码分享给他人
                      </div>
                    </td>
                  </tr>
                </table>
              </td>
            </tr>
            <tr>
              <td style="padding-bottom: 10px">
                <table cellpadding="0" cellspacing="0" border="0">
                  <tr>
                    <td style="width: 20px; vertical-align: top; padding-top: 2px">
                    </td>
                    <td style="padding-left: 10px">
                      <div style="
                          color: #4a4a4a;
                          font-size: 13px;
                          line-height: 1.6;
                        ">
                        如非本人操作，请忽略本邮件
                      </div>
                    </td>
                  </tr>
                </table>
              </td>
            </tr>
          </table>
        </div>

        <div style="
            border-top: 1px dashed #dbeafe;
            padding-top: 20px;
            position: relative;
          ">
          <p style="
              margin: 0 0 8px 0;
              color: #999;
              font-size: 11px;
              line-height: 1.6;
            ">
            此邮件由系统自动发送，请勿直接回复。
          </p>
          <p style="margin: 0; color: #999; font-size: 11px">
            <span style="color: #2563eb; font-family: monospace; font-weight: 700">CSU STAR</span>
            · 南极星Team
          </p>
        </div>
      </div>
    </div>
  </body>
</html>`

type verificationEmailSender func(to []string, captcha string) error

var sendVerificationEmailWithFallbackFn = sendVerificationEmailWithFallback
var tencentVerificationEmailSender = func(from string, to []string, captcha string) error {
	return TencentSesSendEmail(from, to, captcha)
}
var smtpVerificationEmailSender = sendVerificationEmailViaSMTP

func SendVerificationEmail(to []string, captcha string) error {
	return sendVerificationEmailWithFallbackFn(to, captcha)
}

func sendVerificationEmailWithFallback(to []string, captcha string) error {
	attempts := []struct {
		name string
		send verificationEmailSender
	}{
		{
			name: "tencent_ses",
			send: func(to []string, captcha string) error {
				return tencentVerificationEmailSender(config.GlobalConfig.Tencent.Ses.FromEmailAddr, to, captcha)
			},
		},
		{
			name: "aliyun_directmail_smtp",
			send: func(to []string, captcha string) error {
				return smtpVerificationEmailSender(config.GlobalConfig.Mail.Verification.Aliyun, to, captcha)
			},
		},
		{
			name: "qq_smtp",
			send: func(to []string, captcha string) error {
				return smtpVerificationEmailSender(config.GlobalConfig.Mail.Verification.QQ, to, captcha)
			},
		},
	}

	var errs []error
	for _, attempt := range attempts {
		if err := attempt.send(to, captcha); err != nil {
			if logger.Log != nil {
				logger.Log.Warn("验证码邮件发送失败，尝试降级通道", zap.String("provider", attempt.name), zap.Error(err))
			}
			errs = append(errs, fmt.Errorf("%s: %w", attempt.name, err))
			continue
		}

		if logger.Log != nil {
			logger.Log.Info("验证码邮件发送成功", zap.String("provider", attempt.name), zap.Strings("to", to))
		}
		return nil
	}

	return errors.Join(errs...)
}

func sendVerificationEmailViaSMTP(cfg config.SMTPConfig, to []string, captcha string) error {
	if strings.TrimSpace(cfg.Host) == "" || cfg.Port == 0 || strings.TrimSpace(cfg.Username) == "" || strings.TrimSpace(cfg.Password) == "" || strings.TrimSpace(cfg.FromEmailAddr) == "" {
		return errors.New("smtp config is incomplete")
	}

	subject := strings.TrimSpace(config.GlobalConfig.Mail.Verification.Subject)
	if subject == "" {
		subject = strings.TrimSpace(config.GlobalConfig.Tencent.Ses.Subject)
	}
	if subject == "" {
		subject = "CSU Star | 南极星邮箱验证码"
	}

	body := renderVerificationEmailHTML(captcha)
	message := buildHTMLMessage(cfg.FromName, cfg.FromEmailAddr, to, subject, body)
	addr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)

	auth := smtp.PlainAuth("", cfg.Username, cfg.Password, cfg.Host)
	return sendMailUsingTLS(addr, auth, cfg.FromEmailAddr, to, message)
}

func renderVerificationEmailHTML(captcha string) string {
	return strings.ReplaceAll(verificationEmailHTMLTemplate, "{{code}}", captcha)
}

func buildHTMLMessage(fromName, fromEmail string, to []string, subject, body string) []byte {
	headers := map[string]string{
		"From":         formatMailAddress(fromName, fromEmail),
		"To":           strings.Join(to, ","),
		"Subject":      subject,
		"MIME-Version": "1.0",
		"Content-Type": "text/html; charset=UTF-8",
	}

	var builder strings.Builder
	for _, key := range []string{"From", "To", "Subject", "MIME-Version", "Content-Type"} {
		builder.WriteString(fmt.Sprintf("%s: %s\r\n", key, headers[key]))
	}
	builder.WriteString("\r\n")
	builder.WriteString(body)
	return []byte(builder.String())
}

func formatMailAddress(name, email string) string {
	trimmedName := strings.TrimSpace(name)
	if trimmedName == "" {
		return email
	}
	return fmt.Sprintf("%s <%s>", trimmedName, email)
}

func dialSMTPOverTLS(addr string) (*smtp.Client, error) {
	conn, err := tls.Dial("tcp", addr, &tls.Config{MinVersion: tls.VersionTLS12})
	if err != nil {
		return nil, err
	}

	host, _, splitErr := net.SplitHostPort(addr)
	if splitErr != nil {
		_ = conn.Close()
		return nil, splitErr
	}

	client, err := smtp.NewClient(conn, host)
	if err != nil {
		_ = conn.Close()
		return nil, err
	}

	return client, nil
}

func sendMailUsingTLS(addr string, auth smtp.Auth, from string, to []string, msg []byte) (err error) {
	client, err := dialSMTPOverTLS(addr)
	if err != nil {
		return err
	}
	defer client.Close()

	if auth != nil {
		if ok, _ := client.Extension("AUTH"); ok {
			if err = client.Auth(auth); err != nil {
				return err
			}
		}
	}

	if err = client.Mail(from); err != nil {
		return err
	}
	for _, recipient := range to {
		if err = client.Rcpt(recipient); err != nil {
			return err
		}
	}

	writer, err := client.Data()
	if err != nil {
		return err
	}
	if _, err = writer.Write(msg); err != nil {
		return err
	}
	if err = writer.Close(); err != nil {
		return err
	}

	return client.Quit()
}
