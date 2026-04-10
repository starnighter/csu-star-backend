package utils

import (
	"csu-star-backend/config"
	"csu-star-backend/logger"
	"errors"
	"strings"

	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	tencErr "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/errors"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/profile"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/regions"
	ses "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/ses/v20201002"
	"go.uber.org/zap"
)

var (
	sesClient *ses.Client
)

func InitTencentSes() error {
	var err error
	credential := common.NewCredential(
		config.GlobalConfig.Tencent.SecretID,
		config.GlobalConfig.Tencent.SecretKey,
	)
	cpf := profile.NewClientProfile()
	sesClient, err = ses.NewClient(credential, regions.Guangzhou, cpf)
	if err != nil {
		return err
	}

	return nil
}

func TencentSesSendEmail(from string, to []string, captcha string) error {
	if sesClient == nil {
		return errors.New("tencent ses client is not initialized")
	}
	if strings.TrimSpace(from) == "" {
		return errors.New("tencent ses from email is empty")
	}

	req := ses.NewSendEmailRequest()
	req.FromEmailAddress = common.StringPtr(from)
	req.Subject = common.StringPtr(config.GlobalConfig.Tencent.Ses.Subject)
	req.Destination = common.StringPtrs(to)
	req.TriggerType = common.Uint64Ptr(1)
	captchaJson := common.StringPtr("{\"code\":\"" + captcha + "\"}")
	req.Template = &ses.Template{
		TemplateID:   common.Uint64Ptr(config.GlobalConfig.Tencent.Ses.TemplateID),
		TemplateData: captchaJson,
	}

	resp, err := sesClient.SendEmail(req)
	var tencentCloudSDKError *tencErr.TencentCloudSDKError
	if errors.As(err, &tencentCloudSDKError) {
		logger.Log.Error("调用腾讯云SES SDK发送邮件Api失败：", zap.Error(err))
		return err
	}
	if err != nil {
		logger.Log.Error("非腾讯云SES SDK错误，未知错误：", zap.Error(err))
		return err
	}

	logger.Log.Info("调用腾讯云SES SDK发送邮件Api成功！本次请求对应响应体：", zap.String("resp", resp.ToJsonString()))
	return nil
}
