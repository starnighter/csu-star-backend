package utils

import (
	"csu-star-backend/config"
	"errors"
	"log"

	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	tencErr "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/errors"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/profile"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/regions"
	ses "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/ses/v20201002"
)

func TencentSesSendEmail(from string, to []string, captcha string) error {
	credential := common.NewCredential(
		config.GlobalConfig.Tencent.SecretID,
		config.GlobalConfig.Tencent.SecretKey,
	)
	cpf := profile.NewClientProfile()
	client, err := ses.NewClient(credential, regions.Guangzhou, cpf)
	if err != nil {
		return err
	}

	req := ses.NewSendEmailRequest()
	req.FromEmailAddress = common.StringPtr(from)
	req.Subject = common.StringPtr(config.GlobalConfig.Tencent.Ses.Subject)
	req.Destination = common.StringPtrs(to)
	req.TriggerType = common.Uint64Ptr(1)
	captchaJson := common.StringPtr("{\"captcha\":\"" + captcha + "\"}")
	req.Template = &ses.Template{
		TemplateID:   common.Uint64Ptr(config.GlobalConfig.Tencent.Ses.TemplateID),
		TemplateData: captchaJson,
	}

	resp, err := client.SendEmail(req)
	var tencentCloudSDKError *tencErr.TencentCloudSDKError
	if errors.As(err, &tencentCloudSDKError) {
		log.Fatalf("调用腾讯云SES SDK发送邮件Api失败：%v", err)
		return err
	}
	if err != nil {
		log.Fatalf("非腾讯云SES SDK错误，未知错误：%v", err)
		return err
	}

	log.Printf("调用腾讯云SES SDK发送邮件Api成功！本次请求对应响应体：%v", resp.ToJsonString())
	return nil
}
