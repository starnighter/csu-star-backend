package handler

import (
	"csu-star-backend/internal/constant"
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
