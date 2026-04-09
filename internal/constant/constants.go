package constant

import (
	"csu-star-backend/internal/errs"
	"errors"
)

const (
	// redis key 前缀
	BlackListPrefix             = "blacklist:"
	InviteCodePrefix            = "invite:"
	CaptchaPrefix               = "captcha:"
	CaptchaRepeatPrefix         = "captcha:repeat:"
	RateLimitPrefix             = "ratelimit:"
	AbusePrefix                 = "abuse:"
	CacheRandomCoursesPrefix    = "random:courses:"
	CacheRandomTeachersPrefix   = "random:teachers:"
	ResourceUploadSessionPrefix = "resource:upload:session:"

	// 腾讯云相关常量
	TencentCosAvatarsKeyPrefix          = "avatars/"
	TencentCosResourcesKeyPrefix        = "resources/"
	TencentCosPendingResourcesKeyPrefix = "resources/pending/"

	// 其他常量
	SchoolEmailSuffix  = "@csu.edu.cn"
	GinUserID          = "user_id"
	GinUserRole        = "user_role"
	GinAccessTokenHash = "access_token_hash"
	RandCoursesMin     = 1
	RandCoursesMax     = 6187
	RandCoursesCount   = 20
	RandTeachersMin    = 1
	RandTeachersMax    = 2504
	RandTeachersCount  = 20
)

var (
	// 业务错误代码及消息
	// 通用错误
	InternalServerErr = errors.New("服务器内部错误")
	BadRequestErr     = errors.New("参数错误")

	// 登录相关错误
	InviteCodeNotExistErr               = errs.BusinessErr{Code: 1001, Msg: "邀请码不存在"}
	SendCaptchaRepeatedlyIn60sErr       = errs.BusinessErr{Code: 1002, Msg: "60s内重复发送邮箱验证码，等会再试试吧"}
	CaptchaNotMatchErr                  = errs.BusinessErr{Code: 1003, Msg: "验证码错误，请重新输入哦"}
	PasswordIncorrectErr                = errs.BusinessErr{Code: 1004, Msg: "输入的密码不正确，请重新输入哦"}
	UserNotExistErr                     = errs.BusinessErr{Code: 1005, Msg: "用户不存在，请先注册呐"}
	EmailIsExistErr                     = errs.BusinessErr{Code: 1006, Msg: "您已绑定过邮箱，请勿重复绑定哦"}
	EmailHasBeenBoundErr                = errs.BusinessErr{Code: 1007, Msg: "该邮箱已被绑定，换一个试试吧～"}
	ProviderNotSupportErr               = errs.BusinessErr{Code: 1008, Msg: "不支持的提供商，也许以后会支持的吧（可能"}
	DownloadAvatarFromProviderFailedErr = errs.BusinessErr{Code: 1009, Msg: "从提供商下载用户头像失败，万分抱歉，请再重新登录一次吧～"}
	LoginByQQFailedErr                  = errs.BusinessErr{Code: 1010, Msg: "啊哦，QQ登录失败了，请再试一次吧"}
	LoginByWechatFailedErr              = errs.BusinessErr{Code: 1011, Msg: "啊哦，微信登录失败了，请再试一次吧"}
	LoginByGitHubFailedErr              = errs.BusinessErr{Code: 1012, Msg: "啊哦，GitHub登录失败了，请再试一次吧"}
	LoginByGoogleFailedErr              = errs.BusinessErr{Code: 1013, Msg: "啊哦，Google登录失败了，请再试一次吧"}
	AccessTokenExpiredErr               = errs.BusinessErr{Code: 1014, Msg: "access_token已过期，请尝试刷新token或重新登录吧"}
	RefreshTokenExpiredErr              = errs.BusinessErr{Code: 1015, Msg: "refresh_token已过期，请重新登录把"}
	UserBannedErr                       = errs.BusinessErr{Code: 1016, Msg: "用户已被封禁，无法登录"}
	NotRefreshTokenErr                  = errs.BusinessErr{Code: 1017, Msg: "传入的token不是refresh_token，请重新传入"}
	UserHasRegisteredErr                = errs.BusinessErr{Code: 1018, Msg: "用户已注册，请登录"}
	InvalidSchoolEmailErr               = errs.BusinessErr{Code: 1019, Msg: "仅支持绑定校园邮箱"}
	TooManyRequestsErr                  = errs.BusinessErr{Code: 1020, Msg: "请求过于频繁，请稍后再试"}
	UserAutoBannedErr                   = errs.BusinessErr{Code: 1021, Msg: "账号因异常行为已被系统限制"}
	OauthHasBeenBoundErr                = errs.BusinessErr{Code: 1022, Msg: "该第三方账号已绑定其他账号，请先使用原账号登录或先解绑"}

	// 学院相关错误
	QueryDepartmentsFailedErr = errs.BusinessErr{Code: 2001, Msg: "查询学院列表失败"}
)
