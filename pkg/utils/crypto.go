package utils

import (
	"crypto/rand"
	"crypto/sha256"
	"csu-star-backend/config"
	"encoding/hex"
	"errors"
	"io"
	"log"
	"math/big"
	"mime/multipart"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

const charset = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"

type CustomClaims struct {
	UserID   int64  `json:"user_id"`
	UserRole string `json:"user_role"`
	Type     string `json:"type"`
	jwt.RegisteredClaims
}

// GenerateTokenPair 生成 accessToken 以及 refreshToken
func GenerateTokenPair(userID int64, userRole string) (string, string, error) {
	secret := []byte(config.GlobalConfig.JWT.Secret)

	accessClaims := CustomClaims{
		UserID:   userID,
		UserRole: userRole,
		Type:     "access",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Duration(config.GlobalConfig.JWT.AccessExpiration) * time.Second)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	accessToken, err := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims).SignedString(secret)
	if err != nil {
		log.Fatalf("生成访问令牌失败：%v", err)
		return "", "", err
	}

	refreshClaims := CustomClaims{
		UserID:   userID,
		UserRole: userRole,
		Type:     "refresh",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Duration(config.GlobalConfig.JWT.RefreshExpiration) * time.Second)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	refreshToken, err := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims).SignedString(secret)
	if err != nil {
		log.Fatalf("生成刷新令牌失败：%v", err)
		return "", "", err
	}

	return accessToken, refreshToken, nil
}

// ParseToken 解析 JWT Token
func ParseToken(tokenStr string) (*CustomClaims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &CustomClaims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(config.GlobalConfig.JWT.Secret), nil
	})
	if err != nil {
		log.Printf("解析令牌失败：%v", err)
		return nil, err
	}

	if claims, ok := token.Claims.(*CustomClaims); ok && token.Valid {
		return claims, nil
	}
	return nil, errors.New("无效的令牌")
}

// HashPassword 生成密码加密后的 Hash 值
func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(bytes), err
}

// CheckPasswordHash 校验密码和 Hash 值
func CheckPasswordHash(password, hash string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}

// base62Encode 将 int64 转换为 Base62 字符串
func base62Encode(n int64) string {
	if n == 0 {
		return string(charset[0])
	}
	var result []byte
	base := int64(62)
	for n > 0 {
		rem := n % base
		result = append([]byte{charset[rem]}, result...)
		n = n / base
	}
	return string(result)
}

// GenerateNickname 生成随机唯一用户昵称
func GenerateNickname() (string, error) {
	csuPrefix := "csu_"
	timestamp := time.Now().UnixMilli()
	timePrefix := base62Encode(timestamp)

	remainLen := 16 - len(timePrefix)
	if remainLen <= 0 {
		return csuPrefix + timePrefix[:16], nil
	}

	randomSuffix := make([]byte, remainLen)
	for i := range randomSuffix {
		n, err := rand.Int(rand.Reader, big.NewInt(62))
		if err != nil {
			return "", err
		}
		randomSuffix[i] = charset[n.Int64()]
	}

	return csuPrefix + timePrefix + string(randomSuffix), nil
}

// GenerateCaptcha 生成指定长度的验证码
func GenerateCaptcha(length int) (string, error) {
	digits := "0123456789"
	captcha := make([]byte, length)
	for i := range captcha {
		num, err := rand.Int(rand.Reader, big.NewInt(int64(len(digits))))
		if err != nil {
			return "", err
		}
		captcha[i] = digits[num.Int64()]
	}
	return string(captcha), nil
}

// CalculateFileHash 计算文件的 SHA256 哈希值
func CalculateFileHash(f multipart.File) (string, error) {
	// 无论读取是否成功，函数退出时自动文件指针复位
	defer f.Seek(0, io.SeekStart)

	hasher := sha256.New()
	if _, err := io.Copy(hasher, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(hasher.Sum(nil)), nil
}

// DetectMimeType 探测文件的真实 MIME 类型
func DetectMimeType(f multipart.File) (string, error) {
	defer f.Seek(0, io.SeekStart)

	buffer := make([]byte, 512)
	n, err := f.Read(buffer)
	if err != nil && err != io.EOF {
		return "", err
	}
	// buffer[:n] 防止文件小于 512 字节时读取到空数据
	return http.DetectContentType(buffer[:n]), nil
}
