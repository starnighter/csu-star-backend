package resp

import (
	"encoding/json"
	"fmt"
)

type QQTokenResp struct {
	AccessToken  string `json:"access_token"`
	ExpiresIn    int    `json:"-"`
	RefreshToken string `json:"refresh_token"`
	OpenID       string `json:"openid"`
}

type qqTokenRespAlias struct {
	AccessToken  string          `json:"access_token"`
	ExpiresIn    json.RawMessage `json:"expires_in"`
	RefreshToken string          `json:"refresh_token"`
	OpenID       string          `json:"openid"`
}

func (r *QQTokenResp) UnmarshalJSON(data []byte) error {
	var aux qqTokenRespAlias
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	r.AccessToken = aux.AccessToken
	r.RefreshToken = aux.RefreshToken
	r.OpenID = aux.OpenID

	if len(aux.ExpiresIn) == 0 {
		r.ExpiresIn = 0
		return nil
	}

	if err := json.Unmarshal(aux.ExpiresIn, &r.ExpiresIn); err == nil {
		return nil
	}

	var expiresInStr string
	if err := json.Unmarshal(aux.ExpiresIn, &expiresInStr); err == nil {
		var parsed int
		if _, err = fmt.Sscanf(expiresInStr, "%d", &parsed); err == nil {
			r.ExpiresIn = parsed
			return nil
		}
	}

	return fmt.Errorf("invalid expires_in: %s", string(aux.ExpiresIn))
}
