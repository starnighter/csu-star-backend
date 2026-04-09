package service

import (
	"context"
	"csu-star-backend/config"
	"csu-star-backend/internal/constant"
	"csu-star-backend/internal/model"
	"csu-star-backend/internal/repo"
	"csu-star-backend/internal/resp"
	"csu-star-backend/pkg/utils"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/github"
	"golang.org/x/oauth2/google"
	"gorm.io/gorm"
)

type OauthService struct {
	userRepo    repo.UserRepository
	httpClient  *http.Client
	securitySvc *SecurityService
}

func NewOauthService(ur repo.UserRepository, c *http.Client) *OauthService {
	return &OauthService{userRepo: ur, httpClient: c}
}

func (s *OauthService) SetSecurityService(securitySvc *SecurityService) {
	s.securitySvc = securitySvc
}

func (s *OauthService) OauthLogin(provider model.OauthProvider, code string, codeVerifier string) (*model.Users, string, string, error) {
	userInfo, err := s.fetchProviderUserInfo(provider, code, codeVerifier)
	if err != nil {
		return nil, "", "", err
	}

	user, err := s.userRepo.FindOrCreateOauthUser(provider, &userInfo)
	if err != nil {
		return nil, "", "", err
	}
	if s.securitySvc != nil {
		checkedUser, _, accessErr := s.securitySvc.EnforceUserAccess(user.ID)
		if accessErr != nil {
			if errors.Is(accessErr, ErrSecurityUserBanned) {
				return nil, "", "", &constant.UserBannedErr
			}
			return nil, "", "", accessErr
		}
		if checkedUser != nil {
			user = checkedUser
		}
	} else if user.Status == model.UserStatusBanned {
		return nil, "", "", &constant.UserBannedErr
	}

	accessToken, refreshToken, err := utils.GenerateTokenPair(user.ID, string(user.Role))
	return user, accessToken, refreshToken, err
}

func (s *OauthService) OauthBind(userID int64, provider model.OauthProvider, code string, codeVerifier string) (*model.UserOauthBinding, error) {
	userInfo, err := s.fetchProviderUserInfo(provider, code, codeVerifier)
	if err != nil {
		return nil, err
	}

	existingBinding, err := s.userRepo.FindUserOauthBinding(provider, userInfo.OpenID)
	if err == nil {
		if existingBinding.UserID == userID {
			return existingBinding, nil
		}
		return nil, &constant.OauthHasBeenBoundErr
	}
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	userOauthBing, err := s.userRepo.CreateUserOauthBinding(userID, provider, &userInfo)
	if err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			conflictBinding, findErr := s.userRepo.FindUserOauthBinding(provider, userInfo.OpenID)
			if findErr == nil && conflictBinding.UserID == userID {
				return conflictBinding, nil
			}
			return nil, &constant.OauthHasBeenBoundErr
		}
		return nil, err
	}

	return userOauthBing, nil
}

func (s *OauthService) fetchProviderUserInfo(provider model.OauthProvider, code string, codeVerifier string) (model.UserInfo, error) {
	normalizedProvider := model.OauthProvider(strings.ToLower(string(provider)))

	switch normalizedProvider {
	case model.OauthProviderQQ:
		return s.handleQQ(code)
	case model.OauthProviderWechat:
		return s.handleWechat(code)
	case model.OauthProviderGithub:
		return s.handleGitHub(code, codeVerifier)
	case model.OauthProviderGoogle:
		return s.handleGoogle(code, codeVerifier)
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
	tokenReq, err := http.NewRequest(http.MethodGet, tokenReqUrl, nil)
	if err != nil {
		return userInfo, err
	}

	tokenResp, err := s.httpClient.Do(tokenReq)
	if err != nil {
		return userInfo, err
	}
	defer tokenResp.Body.Close()

	tokenBody, err := io.ReadAll(tokenResp.Body)
	if err != nil {
		return userInfo, err
	}

	if tokenResp.StatusCode != http.StatusOK {
		return userInfo, loginErr
	}

	var qqTokenResp resp.QQTokenResp
	err = json.Unmarshal(tokenBody, &qqTokenResp)
	if err != nil {
		return userInfo, err
	}
	if qqTokenResp.AccessToken == "" || qqTokenResp.OpenID == "" {
		return userInfo, loginErr
	}

	// 通过accessToken获取userInfo
	userReqUrl := fmt.Sprintf(
		"https://graph.qq.com/user/get_user_info?access_token=%s&oauth_consumer_key=%s&openid=%s",
		qqTokenResp.AccessToken,
		config.GlobalConfig.Oauth.QQ.AppID,
		qqTokenResp.OpenID,
	)
	userReq, err := http.NewRequest(http.MethodGet, userReqUrl, nil)
	if err != nil {
		return userInfo, err
	}

	userResp, err := s.httpClient.Do(userReq)
	if err != nil {
		return userInfo, err
	}
	defer userResp.Body.Close()

	userBody, err := io.ReadAll(userResp.Body)
	if err != nil {
		return userInfo, err
	}

	if userResp.StatusCode != http.StatusOK {
		return userInfo, loginErr
	}

	var qqUserResp resp.QQUserResp
	err = json.Unmarshal(userBody, &qqUserResp)
	if err != nil {
		return userInfo, err
	}
	if qqUserResp.Ret != 0 || qqUserResp.Nickname == "" {
		return userInfo, loginErr
	}

	userInfo.Nickname = qqUserResp.Nickname
	userInfo.AvatarUrl = normalizeProviderAvatarURL(qqUserResp.FigureurlQQ2)
	userInfo.OpenID = qqTokenResp.OpenID

	return userInfo, nil
}

func normalizeProviderAvatarURL(raw string) string {
	raw = strings.TrimSpace(raw)
	if strings.HasPrefix(raw, "http://thirdqq.qlogo.cn/") {
		return "https://" + strings.TrimPrefix(raw, "http://")
	}
	return raw
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
	if err != nil {
		return userInfo, err
	}

	userInfo.Nickname = wechatUserResp.Nickname
	userInfo.AvatarUrl = avatarUrl
	userInfo.OpenID = wechatUserResp.OpenId
	userInfo.UnionId = wechatUserResp.Unionid

	return userInfo, nil
}

func (s *OauthService) handleGitHub(code string, codeVerifier string) (model.UserInfo, error) {
	var userInfo model.UserInfo
	loginErr := &constant.LoginByGitHubFailedErr

	conf := &oauth2.Config{
		ClientID:     config.GlobalConfig.Oauth.GitHub.ClientID,
		ClientSecret: config.GlobalConfig.Oauth.GitHub.ClientSecret,
		RedirectURL:  config.GlobalConfig.Oauth.GitHub.RedirectUri,
		Endpoint:     github.Endpoint,
	}

	var (
		token *oauth2.Token
		err   error
	)
	if codeVerifier != "" {
		token, err = conf.Exchange(context.Background(), code, oauth2.SetAuthURLParam("code_verifier", codeVerifier))
	} else {
		token, err = conf.Exchange(context.Background(), code)
	}
	if err != nil {
		return userInfo, loginErr
	}

	client := conf.Client(context.Background(), token)

	userResp, err := client.Get("https://api.github.com/user")
	if err != nil {
		return userInfo, loginErr
	}
	defer userResp.Body.Close()

	if userResp.StatusCode != http.StatusOK {
		return userInfo, loginErr
	}

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

func (s *OauthService) handleGoogle(code string, codeVerifier string) (model.UserInfo, error) {
	var userInfo model.UserInfo
	loginErr := &constant.LoginByGoogleFailedErr

	conf := &oauth2.Config{
		ClientID:     config.GlobalConfig.Oauth.Google.ClientID,
		ClientSecret: config.GlobalConfig.Oauth.Google.ClientSecret,
		RedirectURL:  config.GlobalConfig.Oauth.Google.RedirectUri,
		Scopes: []string{
			"https://www.googleapis.com/auth/userinfo.profile",
			"https://www.googleapis.com/auth/userinfo.email",
		},
		Endpoint: google.Endpoint,
	}

	var (
		token *oauth2.Token
		err   error
	)
	if codeVerifier != "" {
		token, err = conf.Exchange(context.Background(), code, oauth2.SetAuthURLParam("code_verifier", codeVerifier))
	} else {
		token, err = conf.Exchange(context.Background(), code)
	}
	if err != nil {
		fmt.Printf("[oauth][google] exchange failed code=%s has_verifier=%t err=%v\n", code, codeVerifier != "", err)
		return userInfo, loginErr
	}

	client := conf.Client(context.Background(), token)

	userResp, err := client.Get("https://www.googleapis.com/oauth2/v3/userinfo")
	if err != nil {
		fmt.Printf("[oauth][google] userinfo request failed err=%v\n", err)
		return userInfo, loginErr
	}
	defer userResp.Body.Close()

	if userResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(userResp.Body)
		fmt.Printf("[oauth][google] userinfo status=%d body=%s\n", userResp.StatusCode, string(body))
		return userInfo, loginErr
	}

	var googleUserResp resp.GoogleUserResp
	err = json.NewDecoder(userResp.Body).Decode(&googleUserResp)
	if err != nil {
		fmt.Printf("[oauth][google] userinfo decode failed err=%v\n", err)
		return userInfo, err
	}
	if googleUserResp.Sub == "" || googleUserResp.Name == "" {
		fmt.Printf("[oauth][google] userinfo fields invalid sub_empty=%t name_empty=%t\n", googleUserResp.Sub == "", googleUserResp.Name == "")
		return userInfo, loginErr
	}

	userInfo.Nickname = googleUserResp.Name
	userInfo.AvatarUrl = googleUserResp.Picture
	userInfo.OpenID = googleUserResp.Sub

	return userInfo, nil
}
