package constant

import "csu-star-backend/internal/errs"

const (
	// redis key 前缀
	BlackListPrefix  = "blacklist:"
	InviteCodePrefix = "invite:"
	CaptchaPrefix    = "captcha:"

	// 常量
	SchoolEmailSuffix = "@csu.edu.cn"
)

var (
	// 业务错误代码及消息
	InviteCodeNotExistErr         = errs.BusinessErr{Code: 1001, Msg: "邀请码不存在"}
	SendCaptchaRepeatedlyIn60sErr = errs.BusinessErr{Code: 1002, Msg: "60s内重复发送邮箱验证码，请稍候再试"}
)
