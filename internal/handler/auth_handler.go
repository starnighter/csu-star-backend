package handler

import (
	"csu-star-backend/internal/constant"
	"csu-star-backend/internal/req"
	"csu-star-backend/internal/resp"
	"csu-star-backend/internal/service"
	"errors"
	"net/http"

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
		resp.FailWithCode(c, http.StatusBadRequest, constant.BadRequestErr.Error())
	}

	err := h.authSvc.Register(r.Email, r.Password, r.Nickname, r.AvatarUrl, r.InviteCode)
	if errors.Is(err, &constant.InviteCodeNotExistErr) {
		resp.FailWithCode(c, http.StatusBadRequest, err.Error())
	}
	if err != nil {
		resp.Fail(c, constant.InternalServerErr.Error())
	}
	resp.Success(c, "注册成功，请登录")
}
