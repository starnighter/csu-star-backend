package handler

import (
	"errors"
	"net/http"

	"csu-star-backend/internal/constant"
	"csu-star-backend/internal/req"
	"csu-star-backend/internal/resp"
	"csu-star-backend/internal/service"

	"github.com/gin-gonic/gin"
)

type MailProviderHandler struct {
	mailSvc *service.MailProviderService
}

func NewMailProviderHandler(mailSvc *service.MailProviderService) *MailProviderHandler {
	return &MailProviderHandler{mailSvc: mailSvc}
}

func (h *MailProviderHandler) List(c *gin.Context) {
	items, err := h.mailSvc.List()
	if err != nil {
		failInternalWithLog(c, err)
		return
	}
	errs, warns := h.mailSvc.Preflight()
	resp.Success(c, gin.H{"items": items, "errors": errs, "warnings": warns})
}

func (h *MailProviderHandler) Create(c *gin.Context) {
	var r req.MailProviderCreateReq
	if err := c.ShouldBindJSON(&r); err != nil {
		resp.FailWithCode(c, http.StatusBadRequest, resp.CodeFail, constant.BadRequestErr.Error())
		return
	}

	item, err := h.mailSvc.Create(service.MailProviderInput{
		Name:          r.Name,
		Kind:          r.Kind,
		Host:          r.Host,
		Port:          r.Port,
		TLSMode:       r.TLSMode,
		Username:      r.Username,
		Password:      r.Password,
		FromEmailAddr: r.FromEmailAddr,
		FromName:      r.FromName,
		Tier:          r.Tier,
		Enabled:       r.Enabled,
	})
	switch {
	case errors.Is(err, service.ErrMailProviderInvalid):
		resp.FailWithCode(c, http.StatusBadRequest, resp.CodeFail, "邮件通道参数无效")
	case err != nil:
		failInternalWithLog(c, err)
	default:
		resp.Success(c, item)
	}
}

func (h *MailProviderHandler) Update(c *gin.Context) {
	id, ok := parsePositiveID(c)
	if !ok {
		return
	}
	var r req.MailProviderUpdateReq
	if err := c.ShouldBindJSON(&r); err != nil {
		resp.FailWithCode(c, http.StatusBadRequest, resp.CodeFail, constant.BadRequestErr.Error())
		return
	}

	err := h.mailSvc.Update(id, service.MailProviderInput{
		Name:          r.Name,
		Kind:          r.Kind,
		Host:          r.Host,
		Port:          r.Port,
		TLSMode:       r.TLSMode,
		Username:      r.Username,
		Password:      r.Password,
		FromEmailAddr: r.FromEmailAddr,
		FromName:      r.FromName,
		Tier:          r.Tier,
		Enabled:       r.Enabled,
	})
	switch {
	case errors.Is(err, service.ErrMailProviderNotFound):
		resp.FailWithCode(c, http.StatusNotFound, resp.CodeFail, "邮件通道不存在")
	case errors.Is(err, service.ErrMailProviderInvalid):
		resp.FailWithCode(c, http.StatusBadRequest, resp.CodeFail, "邮件通道参数无效")
	case err != nil:
		failInternalWithLog(c, err)
	default:
		resp.SuccessMsg(c, "更新成功")
	}
}

func (h *MailProviderHandler) Delete(c *gin.Context) {
	id, ok := parsePositiveID(c)
	if !ok {
		return
	}
	err := h.mailSvc.Delete(id)
	switch {
	case errors.Is(err, service.ErrMailProviderNotFound):
		resp.FailWithCode(c, http.StatusNotFound, resp.CodeFail, "邮件通道不存在")
	case err != nil:
		failInternalWithLog(c, err)
	default:
		resp.SuccessMsg(c, "删除成功")
	}
}

// SendTest 用指定通道发一封测试邮件。失败时把 SMTP 的原始错误回传给管理员——
// 这类错误（认证失败、发信地址未认证、超额）只有原文才有排查价值。
func (h *MailProviderHandler) SendTest(c *gin.Context) {
	id, ok := parsePositiveID(c)
	if !ok {
		return
	}
	var r req.MailProviderTestReq
	if err := c.ShouldBindJSON(&r); err != nil {
		resp.FailWithCode(c, http.StatusBadRequest, resp.CodeFail, constant.BadRequestErr.Error())
		return
	}

	err := h.mailSvc.SendTest(id, r.To)
	switch {
	case errors.Is(err, service.ErrMailProviderNotFound):
		resp.FailWithCode(c, http.StatusNotFound, resp.CodeFail, "邮件通道不存在")
	case err != nil:
		resp.FailWithData(c, http.StatusBadGateway, resp.CodeFail, "测试邮件发送失败", gin.H{"detail": err.Error()})
	default:
		resp.SuccessMsg(c, "测试邮件已发送，请查收")
	}
}
