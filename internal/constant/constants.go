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
	SendCaptchaRepeatedlyIn60sErr = errs.BusinessErr{Code: 1002, Msg: "60s内重复发送邮箱验证码，等会再试试吧"}
	CaptchaNotMatchErr            = errs.BusinessErr{Code: 1003, Msg: "验证码错误，请重新输入哦"}
	PasswordIncorrectErr          = errs.BusinessErr{Code: 1004, Msg: "输入的密码不正确，请重新输入哦"}
	UserNotExistErr               = errs.BusinessErr{Code: 1005, Msg: "用户不存在，请先注册呐"}
	EmailIsExistErr               = errs.BusinessErr{Code: 1006, Msg: "您已绑定过邮箱，请勿重复绑定哦"}
	EmailHasBeenBoundErr          = errs.BusinessErr{Code: 1007, Msg: "该邮箱已被绑定，换一个试试吧～"}
	ProviderNotSupportErr         = errs.BusinessErr{Code: 1008, Msg: "不支持的提供商，也许以后会支持的吧（可能"}
)
