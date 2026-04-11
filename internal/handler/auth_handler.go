package handler

import (
	"csu-star-backend/internal/constant"
	"csu-star-backend/internal/model"
	"csu-star-backend/internal/req"
	"csu-star-backend/internal/resp"
	"csu-star-backend/internal/service"
	"csu-star-backend/pkg/utils"
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

type AuthHandler struct {
	authSvc  *service.AuthService
	oauthSvc *service.OauthService
}

var getTokenRemainingTime = utils.GetTokenRemainingTime

func NewAuthHandler(authSvc *service.AuthService, oauthSvc *service.OauthService) *AuthHandler {
	return &AuthHandler{
		authSvc:  authSvc,
		oauthSvc: oauthSvc,
	}
}

func (h *AuthHandler) Register(c *gin.Context) {
	var r req.RegisterReq
	if err := c.ShouldBindBodyWithJSON(&r); err != nil {
		resp.FailWithCode(c, http.StatusBadRequest, resp.CodeFail, constant.BadRequestErr.Error())
		return
	}

	err := h.authSvc.Register(r.Email, r.Password, r.Nickname, r.AvatarUrl, r.InviteCode)
	if errors.Is(err, &constant.InvalidSchoolEmailErr) {
		resp.FailWithCode(c, http.StatusBadRequest, constant.InvalidSchoolEmailErr.Code, constant.InvalidSchoolEmailErr.Msg)
		return
	}
	if errors.Is(err, &constant.InviteCodeNotExistErr) {
		resp.FailWithCode(c, http.StatusBadRequest, constant.InviteCodeNotExistErr.Code, constant.InviteCodeNotExistErr.Msg)
		return
	}
	if err != nil {
		failInternalWithLog(c, err)
		return
	}

	resp.Success(c, "注册成功，请登录")
}

func (h *AuthHandler) SendCaptcha(c *gin.Context) {
	var r req.SendCaptchaReq
	if err := c.ShouldBindBodyWithJSON(&r); err != nil {
		resp.FailWithCode(c, http.StatusBadRequest, resp.CodeFail, constant.BadRequestErr.Error())
		return
	}
	_, isNotExists := c.Get(constant.GinUserID)
	msg, err := h.authSvc.SendCaptcha(r.Email, isNotExists)
	if errors.Is(err, &constant.InvalidSchoolEmailErr) {
		resp.FailWithCode(c, http.StatusBadRequest, constant.InvalidSchoolEmailErr.Code, constant.InvalidSchoolEmailErr.Msg)
		return
	}
	if errors.Is(err, &constant.SendCaptchaRepeatedlyIn60sErr) {
		resp.FailWithCode(c, http.StatusTooManyRequests, constant.SendCaptchaRepeatedlyIn60sErr.Code, constant.SendCaptchaRepeatedlyIn60sErr.Msg)
		return
	}
	if errors.Is(err, &constant.UserHasRegisteredErr) {
		resp.FailWithCode(c, http.StatusBadRequest, constant.UserHasRegisteredErr.Code, constant.UserHasRegisteredErr.Msg)
		return
	}
	if errors.Is(err, &constant.CampusMailboxNotFoundErr) {
		resp.FailWithCode(c, http.StatusBadRequest, constant.CampusMailboxNotFoundErr.Code, constant.CampusMailboxNotFoundErr.Msg)
		return
	}
	if errors.Is(err, &constant.CampusMailboxCheckRetryErr) {
		resp.FailWithCode(c, http.StatusServiceUnavailable, constant.CampusMailboxCheckRetryErr.Code, constant.CampusMailboxCheckRetryErr.Msg)
		return
	}
	if err != nil {
		failInternalWithLog(c, err)
		return
	}

	resp.SuccessMsg(c, msg)
}

func (h *AuthHandler) VerifyCaptcha(c *gin.Context) {
	var r req.VerifyCaptchaReq
	if err := c.ShouldBindBodyWithJSON(&r); err != nil {
		resp.FailWithCode(c, http.StatusBadRequest, resp.CodeFail, constant.BadRequestErr.Error())
		return
	}

	err := h.authSvc.VerifyCaptcha(r.Email, r.Captcha)
	if errors.Is(err, &constant.InvalidSchoolEmailErr) {
		resp.FailWithCode(c, http.StatusBadRequest, constant.InvalidSchoolEmailErr.Code, constant.InvalidSchoolEmailErr.Msg)
		return
	}
	if errors.Is(err, &constant.CaptchaNotMatchErr) {
		resp.FailWithCode(c, http.StatusBadRequest, constant.CaptchaNotMatchErr.Code, constant.CaptchaNotMatchErr.Msg)
		return
	}
	if err != nil {
		failInternalWithLog(c, err)
		return
	}

	resp.SuccessMsg(c, "验证码校验成功")
}

func (h *AuthHandler) Login(c *gin.Context) {
	var r req.LoginReq
	if err := c.ShouldBindBodyWithJSON(&r); err != nil {
		resp.FailWithCode(c, http.StatusBadRequest, resp.CodeFail, constant.BadRequestErr.Error())
		return
	}

	user, accessToken, refreshToken, err := h.authSvc.Login(r.Email, r.Password)
	if errors.Is(err, &constant.InvalidSchoolEmailErr) {
		resp.FailWithCode(c, http.StatusBadRequest, constant.InvalidSchoolEmailErr.Code, constant.InvalidSchoolEmailErr.Msg)
		return
	}
	if errors.Is(err, &constant.UserNotExistErr) {
		resp.FailWithCode(c, http.StatusBadRequest, constant.UserNotExistErr.Code, constant.UserNotExistErr.Msg)
		return
	}
	if errors.Is(err, &constant.PasswordIncorrectErr) {
		resp.FailWithCode(c, http.StatusBadRequest, constant.PasswordIncorrectErr.Code, constant.PasswordIncorrectErr.Msg)
		return
	}
	if errors.Is(err, &constant.UserBannedErr) {
		resp.FailWithCode(c, http.StatusForbidden, constant.UserBannedErr.Code, constant.UserBannedErr.Msg)
		return
	}
	if err != nil {
		failInternalWithLog(c, err)
		return
	}

	respData, err := buildLoginResp(user, accessToken, refreshToken)
	if err != nil {
		failInternalWithLog(c, err)
		return
	}

	resp.Success(c, respData)
}

func (h *AuthHandler) BindEmail(c *gin.Context) {
	var r req.BindEmailReq
	value, _ := c.Get(constant.GinUserID)
	userID := value.(int64)

	if err := c.ShouldBindBodyWithJSON(&r); err != nil {
		resp.FailWithCode(c, http.StatusBadRequest, resp.CodeFail, constant.BadRequestErr.Error())
		return
	}

	err := h.authSvc.VerifyCaptcha(r.Email, r.Captcha)
	if errors.Is(err, &constant.InvalidSchoolEmailErr) {
		resp.FailWithCode(c, http.StatusBadRequest, constant.InvalidSchoolEmailErr.Code, constant.InvalidSchoolEmailErr.Msg)
		return
	}
	if errors.Is(err, &constant.CaptchaNotMatchErr) {
		resp.FailWithCode(c, http.StatusBadRequest, constant.CaptchaNotMatchErr.Code, constant.CaptchaNotMatchErr.Msg)
		return
	}
	if err != nil {
		failInternalWithLog(c, err)
		return
	}
	err = h.authSvc.BindEmail(userID, r.Email)
	switch {
	case errors.Is(err, &constant.InvalidSchoolEmailErr):
		resp.FailWithCode(c, http.StatusBadRequest, constant.InvalidSchoolEmailErr.Code, constant.InvalidSchoolEmailErr.Msg)
		return
	case errors.Is(err, &constant.UserNotExistErr):
		resp.FailWithCode(c, http.StatusBadRequest, constant.UserNotExistErr.Code, constant.UserNotExistErr.Msg)
		return
	case errors.Is(err, &constant.EmailIsExistErr):
		resp.FailWithCode(c, http.StatusBadRequest, constant.EmailIsExistErr.Code, constant.EmailIsExistErr.Msg)
		return
	case errors.Is(err, &constant.EmailHasBeenBoundErr):
		resp.FailWithCode(c, http.StatusBadRequest, constant.EmailHasBeenBoundErr.Code, constant.EmailHasBeenBoundErr.Msg)
		return
	case err != nil:
		failInternalWithLog(c, err)
		return
	}

	resp.SuccessMsg(c, "邮箱绑定成功")
}

func (h *AuthHandler) OauthLogin(c *gin.Context) {
	var r req.OauthLoginReq
	if err := c.ShouldBindBodyWithJSON(&r); err != nil {
		resp.FailWithCode(c, http.StatusBadRequest, resp.CodeFail, constant.BadRequestErr.Error())
		return
	}

	user, accessToken, refreshToken, err := h.oauthSvc.OauthLogin(model.OauthProvider(r.Provider), r.Code, r.Meta.CodeVerifier)
	switch {
	case errors.Is(err, &constant.LoginByQQFailedErr):
		resp.FailWithCode(c, http.StatusBadRequest, constant.LoginByQQFailedErr.Code, constant.LoginByQQFailedErr.Msg)
		return
	case errors.Is(err, &constant.LoginByWechatFailedErr):
		resp.FailWithCode(c, http.StatusBadRequest, constant.LoginByWechatFailedErr.Code, constant.LoginByWechatFailedErr.Msg)
		return
	case errors.Is(err, &constant.LoginByGitHubFailedErr):
		resp.FailWithCode(c, http.StatusBadRequest, constant.LoginByGitHubFailedErr.Code, constant.LoginByGitHubFailedErr.Msg)
		return
	case errors.Is(err, &constant.LoginByGoogleFailedErr):
		resp.FailWithCode(c, http.StatusBadRequest, constant.LoginByGoogleFailedErr.Code, constant.LoginByGoogleFailedErr.Msg)
		return
	case errors.Is(err, &constant.UserBannedErr):
		resp.FailWithCode(c, http.StatusForbidden, constant.UserBannedErr.Code, constant.UserBannedErr.Msg)
		return
	case err != nil:
		failInternalWithLog(c, err)
		return
	}

	respData, err := buildLoginResp(user, accessToken, refreshToken)
	if err != nil {
		failInternalWithLog(c, err)
		return
	}

	resp.Success(c, respData)
}

func (h *AuthHandler) OauthBind(c *gin.Context) {
	var r req.OauthBindReq
	value, _ := c.Get(constant.GinUserID)
	userID := value.(int64)
	if err := c.ShouldBindBodyWithJSON(&r); err != nil {
		resp.FailWithCode(c, http.StatusBadRequest, resp.CodeFail, constant.BadRequestErr.Error())
		return
	}

	userOauthBinding, err := h.oauthSvc.OauthBind(userID, model.OauthProvider(r.Provider), r.Code, r.Meta.CodeVerifier)
	switch {
	case errors.Is(err, &constant.LoginByQQFailedErr):
		resp.FailWithCode(c, http.StatusBadRequest, constant.LoginByQQFailedErr.Code, constant.LoginByQQFailedErr.Msg)
		return
	case errors.Is(err, &constant.LoginByWechatFailedErr):
		resp.FailWithCode(c, http.StatusBadRequest, constant.LoginByWechatFailedErr.Code, constant.LoginByWechatFailedErr.Msg)
		return
	case errors.Is(err, &constant.LoginByGitHubFailedErr):
		resp.FailWithCode(c, http.StatusBadRequest, constant.LoginByGitHubFailedErr.Code, constant.LoginByGitHubFailedErr.Msg)
		return
	case errors.Is(err, &constant.LoginByGoogleFailedErr):
		resp.FailWithCode(c, http.StatusBadRequest, constant.LoginByGoogleFailedErr.Code, constant.LoginByGoogleFailedErr.Msg)
		return
	case errors.Is(err, &constant.OauthHasBeenBoundErr):
		resp.FailWithCode(c, http.StatusConflict, constant.OauthHasBeenBoundErr.Code, constant.OauthHasBeenBoundErr.Msg)
		return
	case err != nil:
		failInternalWithLog(c, err)
		return
	}

	o := resp.OauthBindResp{
		Provider: r.Provider,
		BoundAt:  strconv.FormatInt(userOauthBinding.BoundAt.UnixMilli(), 10),
	}

	resp.Success(c, o)
}

func (h *AuthHandler) Refresh(c *gin.Context) {
	var r req.RefreshTokenReq
	if err := c.ShouldBindBodyWithJSON(&r); err != nil {
		resp.FailWithCode(c, http.StatusBadRequest, resp.CodeFail, constant.BadRequestErr.Error())
		return
	}

	accessToken, refreshToken, banDecision, err := h.authSvc.Refresh(r.RefreshToken)
	if errors.Is(err, &constant.RefreshTokenExpiredErr) {
		resp.FailWithCode(c, http.StatusUnauthorized, constant.RefreshTokenExpiredErr.Code, constant.RefreshTokenExpiredErr.Msg)
		return
	}
	if errors.Is(err, &constant.UserBannedErr) {
		code := constant.UserBannedErr.Code
		msg := constant.UserBannedErr.Msg
		if banDecision != nil && banDecision.BanSource == model.UserBanSourceSystem {
			code = constant.UserAutoBannedErr.Code
			msg = constant.UserAutoBannedErr.Msg
		}
		resp.FailWithData(c, http.StatusForbidden, code, msg, service.BuildRiskData(banDecision))
		return
	}
	if err != nil {
		resp.FailWithCode(c, http.StatusUnauthorized, resp.CodeFail, "refresh_token无效，请重新登录")
		return
	}

	remainingTime, err := utils.GetTokenRemainingTime(accessToken)
	if err != nil {
		failInternalWithLog(c, err)
		return
	}

	rt := resp.RefreshTokenResp{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    strconv.FormatInt(remainingTime, 10),
	}

	resp.Success(c, rt)
}

func (h *AuthHandler) Logout(c *gin.Context) {
	v, _ := c.Get(constant.GinAccessTokenHash)
	tokenHash := v.(string)

	err := h.authSvc.Logout(tokenHash)
	if err != nil {
		failInternalWithLog(c, err)
		return
	}

	resp.SuccessMsg(c, "登出成功！")
}

func (h *AuthHandler) ForgetPwd(c *gin.Context) {
	var r req.ForgetPwdReq
	if err := c.ShouldBindBodyWithJSON(&r); err != nil {
		resp.FailWithCode(c, http.StatusBadRequest, resp.CodeFail, constant.BadRequestErr.Error())
		return
	}

	err := h.authSvc.ForgetPwd(r.Email, r.Captcha, r.Password)
	if errors.Is(err, &constant.InvalidSchoolEmailErr) {
		resp.FailWithCode(c, http.StatusBadRequest, constant.InvalidSchoolEmailErr.Code, constant.InvalidSchoolEmailErr.Msg)
		return
	}
	if errors.Is(err, &constant.UserNotExistErr) {
		resp.FailWithCode(c, http.StatusBadRequest, constant.UserNotExistErr.Code, constant.UserNotExistErr.Msg)
		return
	}
	if errors.Is(err, &constant.CaptchaNotMatchErr) {
		resp.FailWithCode(c, http.StatusBadRequest, constant.CaptchaNotMatchErr.Code, constant.CaptchaNotMatchErr.Msg)
		return
	}
	if err != nil {
		failInternalWithLog(c, err)
		return
	}

	resp.SuccessMsg(c, "修改密码成功，请重新登录！")
}

func buildLoginResp(user *model.Users, accessToken, refreshToken string) (resp.LoginResp, error) {
	remaining, err := getTokenRemainingTime(accessToken)
	if err != nil {
		return resp.LoginResp{}, err
	}

	return resp.LoginResp{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    strconv.FormatInt(remaining, 10),
		UserProfile: resp.UserProfileResp{
			ID:                strconv.FormatInt(user.ID, 10),
			Nickname:          user.Nickname,
			AvatarUrl:         user.AvatarUrl,
			Role:              string(user.Role),
			EmailVerified:     user.EmailVerified,
			FreeDownloadCount: strconv.Itoa(user.FreeDownloadCount),
		},
	}, nil
}
