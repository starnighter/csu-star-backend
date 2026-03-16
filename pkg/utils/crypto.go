package utils

import (
	"crypto/rand"
	"csu-star-backend/config"
	"errors"
	"log"
	"math/big"
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

func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(bytes), err
}

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

func GenerateNickname() string {
	csuPrefix := "csu_"
	timestamp := time.Now().UnixMilli()
	timePrefix := base62Encode(timestamp)

	remainLen := 16 - len(timePrefix)
	if remainLen <= 0 {
		return csuPrefix + timePrefix[:16]
	}

	randomSuffix := make([]byte, remainLen)
	for i := range randomSuffix {
		n, err := rand.Int(rand.Reader, big.NewInt(62))
		if err != nil {
			panic(err)
		}
		randomSuffix[i] = charset[n.Int64()]
	}

	return csuPrefix + timePrefix + string(randomSuffix)
}

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
