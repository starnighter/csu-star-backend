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
	if errors.Is(err, &constant.InviteCodeNotExistErr) {
		resp.FailWithCode(c, http.StatusBadRequest, constant.InviteCodeNotExistErr.Code, constant.InviteCodeNotExistErr.Msg)
		return
	}
	if err != nil {
		resp.Fail(c, constant.InternalServerErr.Error())
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

	err := h.authSvc.SendCaptcha(r.Email)
	if errors.Is(err, &constant.SendCaptchaRepeatedlyIn60sErr) {
		resp.FailWithCode(c, http.StatusTooManyRequests, constant.SendCaptchaRepeatedlyIn60sErr.Code, constant.SendCaptchaRepeatedlyIn60sErr.Msg)
		return
	}
	if err != nil {
		resp.Fail(c, constant.InternalServerErr.Error())
		return
	}

	resp.SuccessMsg(c, "验证码发送成功，请注意查收")
}

func (h *AuthHandler) VerifyCaptcha(c *gin.Context) {
	var r req.VerifyCaptchaReq
	if err := c.ShouldBindBodyWithJSON(&r); err != nil {
		resp.FailWithCode(c, http.StatusBadRequest, resp.CodeFail, constant.BadRequestErr.Error())
		return
	}

	err := h.authSvc.VerifyCaptcha(r.Email, r.Captcha)
	if errors.Is(err, &constant.CaptchaNotMatchErr) {
		resp.FailWithCode(c, http.StatusBadRequest, constant.CaptchaNotMatchErr.Code, constant.CaptchaNotMatchErr.Msg)
		return
	}
	if err != nil {
		resp.Fail(c, constant.InternalServerErr.Error())
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
	if errors.Is(err, &constant.UserNotExistErr) {
		resp.FailWithCode(c, http.StatusBadRequest, constant.UserNotExistErr.Code, constant.UserNotExistErr.Msg)
		return
	}
	if errors.Is(err, &constant.PasswordIncorrectErr) {
		resp.FailWithCode(c, http.StatusBadRequest, constant.PasswordIncorrectErr.Code, constant.PasswordIncorrectErr.Msg)
		return
	}
	if err != nil {
		resp.Fail(c, constant.InternalServerErr.Error())
		return
	}

	u := resp.UserProfileResp{
		ID:                strconv.FormatInt(user.ID, 10),
		Nickname:          user.Nickname,
		AvatarUrl:         user.AvatarUrl,
		Role:              string(user.Role),
		EmailVerified:     user.EmailVerified,
		FreeDownloadCount: strconv.Itoa(user.FreeDownloadCount),
	}

	remaining, err := utils.GetTokenRemainingTime(accessToken)
	respData := resp.LoginResp{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    strconv.FormatInt(remaining, 10),
		UserProfile:  u,
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

	err := h.authSvc.BindEmail(userID, r.Email)
	switch {
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
		resp.Fail(c, constant.InternalServerErr.Error())
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

	user, accessToken, refreshToken, err := h.oauthSvc.OauthLogin(model.OauthProvider(r.Provider), r.Code)
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
	case err != nil:
		resp.Fail(c, constant.InternalServerErr.Error())
		return
	}

	u := resp.UserProfileResp{
		ID:                strconv.FormatInt(user.ID, 10),
		Nickname:          user.Nickname,
		AvatarUrl:         user.AvatarUrl,
		Role:              string(user.Role),
		EmailVerified:     user.EmailVerified,
		FreeDownloadCount: strconv.Itoa(user.FreeDownloadCount),
	}

	remaining, err := utils.GetTokenRemainingTime(accessToken)
	respData := resp.LoginResp{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    strconv.FormatInt(remaining, 10),
		UserProfile:  u,
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

	userOauthBinding, err := h.oauthSvc.OauthBind(userID, model.OauthProvider(r.Provider), r.Code)
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
	case err != nil:
		resp.Fail(c, constant.InternalServerErr.Error())
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

	v, _ := c.Get(constant.GinAccessTokenHash)
	tokenHash := v.(string)
	v, _ = c.Get(constant.GinUserID)
	userID := v.(int64)
	v, _ = c.Get(constant.GinUserRole)
	userRole, _ := v.(string)

	accessToken, _, err := h.authSvc.Refresh(userID, userRole, r.RefreshToken, tokenHash)
	if errors.Is(err, &constant.RefreshTokenExpiredErr) {
		resp.FailWithCode(c, http.StatusUnauthorized, constant.RefreshTokenExpiredErr.Code, constant.RefreshTokenExpiredErr.Msg)
		return
	}
	if err != nil {
		resp.Fail(c, constant.InternalServerErr.Error())
		return
	}

	remainingTime, err := utils.GetTokenRemainingTime(accessToken)
	if err != nil {
		resp.Fail(c, constant.InternalServerErr.Error())
		return
	}

	rt := resp.RefreshTokenResp{
		AccessToken: accessToken,
		ExpiresIn:   strconv.FormatInt(remainingTime, 10),
	}

	resp.Success(c, rt)
}

func (h *AuthHandler) Logout(c *gin.Context) {
	v, _ := c.Get(constant.GinAccessTokenHash)
	tokenHash := v.(string)

	err := h.authSvc.Logout(tokenHash)
	if err != nil {
		resp.Fail(c, constant.InternalServerErr.Error())
		return
	}

	resp.SuccessMsg(c, "登出成功！")
}

func (h *AuthHandler) ForgetPwd(c *gin.Context) {
	var r req.ForgetPwdReq

	err := h.authSvc.ForgetPwd(r.Email, r.Password)
	if errors.Is(err, &constant.UserNotExistErr) {
		resp.FailWithCode(c, http.StatusBadRequest, constant.UserNotExistErr.Code, constant.UserNotExistErr.Msg)
		return
	}
	if err != nil {
		resp.Fail(c, constant.InternalServerErr.Error())
		return
	}

	resp.SuccessMsg(c, "修改密码成功，请重新登录！")
}
