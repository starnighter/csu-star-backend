package handler

import (
	"csu-star-backend/internal/service"

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

func (h *AuthHandler) Register(c *gin.Context) {}
