package service

import (
	"csu-star-backend/config"
	"csu-star-backend/internal/constant"
	"csu-star-backend/internal/model"
	"csu-star-backend/internal/repo"
	"csu-star-backend/internal/resp"
	"csu-star-backend/pkg/utils"
	"encoding/json"
	"fmt"
	"net/http"
)

type OauthService struct {
	userRepo   repo.UserRepository
	httpClient *http.Client
}

func NewOauthService(ur repo.UserRepository, c *http.Client) *OauthService {
	return &OauthService{userRepo: ur, httpClient: c}
}

func (s *OauthService) OauthLogin(provider model.OauthProvider, code string) (string, string, error) {
	userInfo, err := s.fetchProviderUserInfo(provider, code)
	if err != nil {
		return "", "", err
	}

	user, err := s.userRepo.FindOrCreateOauthUser(provider, &userInfo)
	if err != nil {
		return "", "", err
	}
	return utils.GenerateTokenPair(user.ID, string(user.Role))
}

func (s *OauthService) fetchProviderUserInfo(provider model.OauthProvider, code string) (model.UserInfo, error) {
	switch provider {
	case model.OauthProviderQQ:
		return s.handleQQ(code)
	case model.OauthProviderWechat:
		return s.handleWechat(code)
	case model.OauthProviderGithub:
		return s.handleGithub(code)
	case model.OauthProviderGoogle:
		return s.handleGoogle(code)
	default:
		return model.UserInfo{}, &constant.ProviderNotSupportErr
	}
}

func (s *OauthService) handleQQ(code string) (model.UserInfo, error) {
	// TODO 完成QQ登录
	var userInfo model.UserInfo

	return userInfo, nil
}

func (s *OauthService) handleWechat(code string) (model.UserInfo, error) {
	var userInfo model.UserInfo

	// 通过code获取accessToken
	tokenReqUrl := fmt.Sprintf(
		"https://api.weixin.qq.com/sns/oauth2/access_token?appid=%s&secret=%s&code=%s&grant_type=authorization_code",
		config.GlobalConfig.Oauth.Wechat.AppID,
		config.GlobalConfig.Oauth.Wechat.AppSecret,
		code,
	)
	tokenReq, _ := http.NewRequest(http.MethodPost, tokenReqUrl, nil)
	tokenResp, err := s.httpClient.Do(tokenReq)
	if err != nil {
		return userInfo, err
	}
	defer tokenResp.Body.Close()

	var wechatTokenResp resp.WechatTokenResp
	err = json.NewDecoder(tokenResp.Body).Decode(&wechatTokenResp)
	if err != nil {
		return userInfo, err
	}

	// 通过accessToken获取userInfo
	userReqUrl := fmt.Sprintf(
		"https://api.weixin.qq.com/sns/userinfo?access_token=%s&openid=%s",
		wechatTokenResp.AccessToken,
		wechatTokenResp.OpenId,
	)
	userReq, _ := http.NewRequest(http.MethodGet, userReqUrl, nil)
	userResp, err := s.httpClient.Do(userReq)
	if err != nil {
		return userInfo, err
	}
	defer userResp.Body.Close()

	var wechatUserResp resp.WechatUserResp
	err = json.NewDecoder(userResp.Body).Decode(&wechatUserResp)
	if err != nil {
		return userInfo, err
	}

	avatarResp, err := s.httpClient.Get(wechatUserResp.HeadImgUrl)
	if err != nil {
		return userInfo, err
	}
	defer avatarResp.Body.Close()

	if avatarResp.StatusCode != http.StatusOK {
		return userInfo, &constant.DownloadAvatarFromProviderFailedErr
	}

	// 上传获取到的微信用户头像 URL 到 COS，并获取保存后的 COS 永久下载链接
	avatarUrl, err := utils.TencentCosUploadByStream(avatarResp.Body, constant.TencentCosAvatarsKeyPrefix, ".jpg")

	userInfo.Nickname = wechatUserResp.Nickname
	userInfo.AvatarUrl = avatarUrl

	return userInfo, nil
}

func (s *OauthService) handleGithub(code string) (model.UserInfo, error) {
	// TODO 完成Github登录
	var userInfo model.UserInfo

	return userInfo, nil
}

func (s *OauthService) handleGoogle(code string) (model.UserInfo, error) {
	// TODO 完成Google登录
	var userInfo model.UserInfo

	return userInfo, nil
}
