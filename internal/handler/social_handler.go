package handler

import (
	"csu-star-backend/internal/constant"
	"csu-star-backend/internal/model"
	"csu-star-backend/internal/req"
	"csu-star-backend/internal/resp"
	"csu-star-backend/internal/service"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
)

type SocialHandler struct {
	socialSvc *service.SocialService
}

func NewSocialHandler(svc *service.SocialService) *SocialHandler {
	return &SocialHandler{socialSvc: svc}
}

func (h *SocialHandler) Like(c *gin.Context) {
	var r req.LikeReq
	if err := c.ShouldBindBodyWithJSON(&r); err != nil {
		resp.FailWithCode(c, http.StatusBadRequest, resp.CodeFail, constant.BadRequestErr.Error())
		return
	}
	userID := c.MustGet(constant.GinUserID).(int64)
	err := h.socialSvc.Like(userID, model.LikeTargetType(r.TargetType), r.TargetID)
	switch {
	case errors.Is(err, service.ErrSocialTargetNotFound):
		resp.FailWithCode(c, http.StatusNotFound, resp.CodeFail, "目标不存在")
		return
	case errors.Is(err, service.ErrAlreadyLiked):
		resp.FailWithCode(c, http.StatusConflict, resp.CodeFail, "请勿重复点赞")
		return
	case err != nil:
		resp.Fail(c, constant.InternalServerErr.Error())
		return
	}
	resp.SuccessMsg(c, "点赞成功")
}

func (h *SocialHandler) Unlike(c *gin.Context) {
	var r req.LikeReq
	if err := c.ShouldBindBodyWithJSON(&r); err != nil {
		resp.FailWithCode(c, http.StatusBadRequest, resp.CodeFail, constant.BadRequestErr.Error())
		return
	}
	userID := c.MustGet(constant.GinUserID).(int64)
	if err := h.socialSvc.Unlike(userID, model.LikeTargetType(r.TargetType), r.TargetID); err != nil {
		resp.Fail(c, constant.InternalServerErr.Error())
		return
	}
	resp.SuccessMsg(c, "取消点赞成功")
}

func (h *SocialHandler) Favorite(c *gin.Context) {
	var r req.FavoriteReq
	if err := c.ShouldBindBodyWithJSON(&r); err != nil {
		resp.FailWithCode(c, http.StatusBadRequest, resp.CodeFail, constant.BadRequestErr.Error())
		return
	}
	userID := c.MustGet(constant.GinUserID).(int64)
	err := h.socialSvc.Favorite(userID, model.FavoriteTargetType(r.TargetType), r.TargetID)
	switch {
	case errors.Is(err, service.ErrSocialTargetNotFound):
		resp.FailWithCode(c, http.StatusNotFound, resp.CodeFail, "目标不存在")
		return
	case errors.Is(err, service.ErrAlreadyFavorited):
		resp.FailWithCode(c, http.StatusConflict, resp.CodeFail, "请勿重复收藏")
		return
	case err != nil:
		resp.Fail(c, constant.InternalServerErr.Error())
		return
	}
	resp.SuccessMsg(c, "收藏成功")
}

func (h *SocialHandler) Unfavorite(c *gin.Context) {
	var r req.FavoriteReq
	if err := c.ShouldBindBodyWithJSON(&r); err != nil {
		resp.FailWithCode(c, http.StatusBadRequest, resp.CodeFail, constant.BadRequestErr.Error())
		return
	}
	userID := c.MustGet(constant.GinUserID).(int64)
	if err := h.socialSvc.Unfavorite(userID, model.FavoriteTargetType(r.TargetType), r.TargetID); err != nil {
		resp.Fail(c, constant.InternalServerErr.Error())
		return
	}
	resp.SuccessMsg(c, "取消收藏成功")
}
