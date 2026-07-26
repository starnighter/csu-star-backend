package utils

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"io"
	"strings"

	"csu-star-backend/config"
)

// encryptedPrefix 标记密文，便于把「已加密」和「历史明文」区分开，
// 从而允许在不停机的情况下逐条迁移。
const encryptedPrefix = "enc:v1:"

var errNoSecretKey = errors.New("未配置加密密钥")

// secretAEAD 用配置里的密钥派生 AES-256-GCM。
//
// 密钥优先取 security.secret_key；未配置时退回 jwt.secret——不理想，但
// 好过明文落库，且不需要在部署时多配一个值就能生效。
func secretAEAD() (cipher.AEAD, error) {
	cfg := config.GetConfig()
	if cfg == nil {
		return nil, errNoSecretKey
	}
	raw := strings.TrimSpace(cfg.Security.SecretKey)
	if raw == "" {
		raw = strings.TrimSpace(cfg.JWT.Secret)
	}
	if raw == "" {
		return nil, errNoSecretKey
	}

	key := sha256.Sum256([]byte(raw))
	block, err := aes.NewCipher(key[:])
	if err != nil {
		return nil, err
	}
	return cipher.NewGCM(block)
}

// EncryptSecret 加密敏感配置值（如 SMTP 密码）。
// 空串原样返回，方便「不修改该字段」的更新语义。
func EncryptSecret(plaintext string) (string, error) {
	if plaintext == "" {
		return "", nil
	}
	aead, err := secretAEAD()
	if err != nil {
		return "", err
	}

	nonce := make([]byte, aead.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	sealed := aead.Seal(nonce, nonce, []byte(plaintext), nil)
	return encryptedPrefix + base64.StdEncoding.EncodeToString(sealed), nil
}

// DecryptSecret 解密 EncryptSecret 的产物。
// 没有密文前缀的值按历史明文原样返回，保证老数据仍可用。
func DecryptSecret(stored string) (string, error) {
	if stored == "" {
		return "", nil
	}
	if !strings.HasPrefix(stored, encryptedPrefix) {
		return stored, nil
	}

	sealed, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(stored, encryptedPrefix))
	if err != nil {
		return "", err
	}
	aead, err := secretAEAD()
	if err != nil {
		return "", err
	}
	if len(sealed) < aead.NonceSize() {
		return "", errors.New("密文长度不合法")
	}

	nonce, body := sealed[:aead.NonceSize()], sealed[aead.NonceSize():]
	plaintext, err := aead.Open(nil, nonce, body, nil)
	if err != nil {
		return "", err
	}
	return string(plaintext), nil
}

// MaskSecret 返回可安全回传给前端的脱敏形式。
func MaskSecret(stored string) string {
	if stored == "" {
		return ""
	}
	return "********"
}
