package service

import (
	"bytes"
	"csu-star-backend/config"
	"csu-star-backend/internal/constant"
	"csu-star-backend/internal/model"
	"csu-star-backend/internal/repo"
	"csu-star-backend/internal/req"
	"csu-star-backend/internal/resp"
	"csu-star-backend/pkg/utils"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
)

type OauthService struct {
	userRepo   repo.UserRepository
	httpClient *http.Client
}

func NewOauthService(ur repo.UserRepository, c *http.Client) *OauthService {
	return &OauthService{userRepo: ur, httpClient: c}
}

func (s *OauthService) OauthLogin(provider model.OauthProvider, code string, meta req.OauthLoginMeta) (string, string, error) {
	userInfo, err := s.fetchProviderUserInfo(provider, code, meta)
	if err != nil {
		return "", "", err
	}

	user, err := s.userRepo.FindOrCreateOauthUser(provider, &userInfo)
	if err != nil {
		return "", "", err
	}
	return utils.GenerateTokenPair(user.ID, string(user.Role))
}

func (s *OauthService) fetchProviderUserInfo(provider model.OauthProvider, code string, meta req.OauthLoginMeta) (model.UserInfo, error) {
	switch provider {
	case model.OauthProviderQQ:
		return s.handleQQ(code)
	case model.OauthProviderWechat:
		return s.handleWechat(code)
	case model.OauthProviderGithub:
		return s.handleGitHub(code, meta)
	case model.OauthProviderGoogle:
		return s.handleGoogle(code)
	default:
		return model.UserInfo{}, &constant.ProviderNotSupportErr
	}
}

func (s *OauthService) handleQQ(code string) (model.UserInfo, error) {
	var userInfo model.UserInfo
	loginErr := &constant.LoginByQQFailedErr

	// 通过code获取accessToken
	tokenReqUrl := fmt.Sprintf(
		"https://graph.qq.com/oauth2.0/token?client_id=%s&client_secret=%s&code=%s&redirect_uri=%s&fmt=json&need_openid=1&grant_type=authorization_code",
		config.GlobalConfig.Oauth.QQ.AppID,
		config.GlobalConfig.Oauth.QQ.AppKey,
		code,
		config.GlobalConfig.Oauth.QQ.RedirectUri,
	)
	tokenReq, _ := http.NewRequest(http.MethodGet, tokenReqUrl, nil)
	tokenResp, err := s.httpClient.Do(tokenReq)
	if err != nil {
		return userInfo, err
	}
	if tokenResp.StatusCode != http.StatusOK {
		return userInfo, loginErr
	}
	defer tokenResp.Body.Close()

	var qqTokenResp resp.QQTokenResp
	err = json.NewDecoder(tokenResp.Body).Decode(&qqTokenResp)
	if err != nil {
		return userInfo, err
	}

	// 通过accessToken获取userInfo
	userReqUrl := fmt.Sprintf(
		"https://graph.qq.com/user/get_user_info?access_token=%s&oauth_consumer_key=%s&openid=%s",
		qqTokenResp.AccessToken,
		config.GlobalConfig.Oauth.QQ.AppID,
		qqTokenResp.OpenID,
	)
	userReq, _ := http.NewRequest(http.MethodGet, userReqUrl, nil)
	userResp, err := s.httpClient.Do(userReq)
	if err != nil {
		return userInfo, err
	}
	defer userResp.Body.Close()

	var qqUserResp resp.QQUserResp
	err = json.NewDecoder(userResp.Body).Decode(&qqUserResp)
	if err != nil {
		return userInfo, err
	}

	userInfo.Nickname = qqUserResp.Nickname
	userInfo.AvatarUrl = qqUserResp.FigureurlQQ2
	userInfo.OpenID = qqTokenResp.OpenID

	return userInfo, nil
}

func (s *OauthService) handleWechat(code string) (model.UserInfo, error) {
	var userInfo model.UserInfo
	loginErr := &constant.LoginByWechatFailedErr

	// 通过code获取accessToken
	tokenReqUrl := fmt.Sprintf(
		"https://api.weixin.qq.com/sns/oauth2/access_token?appid=%s&secret=%s&code=%s&grant_type=authorization_code",
		config.GlobalConfig.Oauth.Wechat.AppID,
		config.GlobalConfig.Oauth.Wechat.AppSecret,
		code,
	)
	tokenReq, _ := http.NewRequest(http.MethodGet, tokenReqUrl, nil)
	tokenResp, err := s.httpClient.Do(tokenReq)
	if err != nil {
		return userInfo, err
	}
	if tokenResp.StatusCode != http.StatusOK {
		return userInfo, loginErr
	}
	defer tokenResp.Body.Close()

	var wechatTokenResp resp.WechatTokenResp
	err = json.NewDecoder(tokenResp.Body).Decode(&wechatTokenResp)
	if err != nil {
		return userInfo, err
	}
	if wechatTokenResp.AccessToken == "" {
		return userInfo, loginErr
	}

	// 通过accessToken获取userInfo
	userReqUrl := fmt.Sprintf(
		"https://api.weixin.qq.com/sns/userinfo?access_token=%s&openid=%s",
		wechatTokenResp.AccessToken,
		wechatTokenResp.OpenID,
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

	// 上传获取到的头像 URL 到 COS，并获取保存后的 COS 永久下载链接
	avatarUrl, err := utils.TencentCosUploadByStream(avatarResp.Body, constant.TencentCosAvatarsKeyPrefix, ".jpg")

	userInfo.Nickname = wechatUserResp.Nickname
	userInfo.AvatarUrl = avatarUrl
	userInfo.OpenID = wechatUserResp.OpenId
	userInfo.UnionId = wechatUserResp.Unionid

	return userInfo, nil
}

func (s *OauthService) handleGitHub(code string, meta req.OauthLoginMeta) (model.UserInfo, error) {
	var userInfo model.UserInfo
	loginErr := &constant.LoginByGitHubFailedErr

	// 通过code和meta获取accessToken
	tokenReqUrl := "https://github.com/login/oauth/access_token"
	payload := map[string]interface{}{
		"client_id":     config.GlobalConfig.Oauth.GitHub.ClientID,
		"client_secret": config.GlobalConfig.Oauth.GitHub.ClientSecret,
		"code":          code,
		"redirect_uri":  config.GlobalConfig.Oauth.GitHub.RedirectUri,
		"code_verifier": meta.CodeChallenge,
	}
	bodyData, _ := json.Marshal(payload)

	tokenReq, _ := http.NewRequest(http.MethodPost, tokenReqUrl, bytes.NewBuffer(bodyData))
	tokenReq.Header.Set("Content-Type", "application/json")
	tokenReq.Header.Set("Accept", "application/json")
	tokenResp, err := s.httpClient.Do(tokenReq)
	if err != nil {
		return userInfo, err
	}
	if tokenResp.StatusCode != http.StatusOK {
		return userInfo, loginErr
	}
	defer tokenResp.Body.Close()

	var githubTokenResp resp.GitHubTokenResp
	err = json.NewDecoder(tokenResp.Body).Decode(&githubTokenResp)
	if err != nil {
		return userInfo, err
	}
	if githubTokenResp.AccessToken == "" {
		return userInfo, loginErr
	}

	// 通过accessToken获取用户信息
	userReq, err := http.NewRequest("GET", "https://api.github.com/user", nil)
	if err != nil {
		return userInfo, err
	}
	userReq.Header.Set("Authorization", "Bearer "+githubTokenResp.AccessToken)
	userResp, err := s.httpClient.Do(userReq)
	if err != nil {
		return userInfo, err
	}
	defer userResp.Body.Close()

	var githubUserResp resp.GitHubUserResp
	err = json.NewDecoder(userResp.Body).Decode(&githubUserResp)
	if err != nil {
		return userInfo, err
	}

	userInfo.Nickname = githubUserResp.Login
	userInfo.AvatarUrl = githubUserResp.AvatarURL
	userInfo.OpenID = strconv.FormatInt(githubUserResp.ID, 10)

	return userInfo, nil
}

func (s *OauthService) handleGoogle(code string) (model.UserInfo, error) {
	var userInfo model.UserInfo
	loginErr := &constant.LoginByGoogleFailedErr

	// 通过code获取accessToken
	tokenReqUrl := "https://oauth2.googleapis.com/token"
	payload := map[string]string{
		"client_id":     config.GlobalConfig.Oauth.Google.ClientID,
		"client_secret": config.GlobalConfig.Oauth.Google.ClientSecret,
		"code":          code,
		"grant_type":    "authorization_code",
		"redirect_uri":  config.GlobalConfig.Oauth.Google.RedirectUri,
	}
	bodyData, _ := json.Marshal(payload)

	tokenReq, _ := http.NewRequest(http.MethodPost, tokenReqUrl, bytes.NewBuffer(bodyData))
	tokenReq.Header.Set("Content-Type", "application/json")
	tokenResp, err := s.httpClient.Do(tokenReq)
	if err != nil {
		return userInfo, err
	}
	defer tokenResp.Body.Close()

	var googleTokenResp resp.GoogleTokenResp
	err = json.NewDecoder(tokenResp.Body).Decode(&googleTokenResp)
	if err != nil {
		return userInfo, err
	}
	if googleTokenResp.AccessToken == "" {
		return userInfo, loginErr
	}

	// 通过accessToken获取用户信息
	userReq, _ := http.NewRequest("GET", "https://www.googleapis.com/oauth2/v2/userinfo", nil)
	userReq.Header.Set("Authorization", "Bearer "+googleTokenResp.AccessToken)
	userResp, err := s.httpClient.Do(userReq)
	if err != nil {
		return userInfo, err
	}
	defer userResp.Body.Close()

	var googleUserResp resp.GoogleUserResp
	err = json.NewDecoder(userResp.Body).Decode(&googleUserResp)
	if err != nil {
		return userInfo, err
	}

	userInfo.Nickname = googleUserResp.Name
	userInfo.AvatarUrl = googleUserResp.Picture
	userInfo.OpenID = googleUserResp.ID

	return userInfo, nil
}
