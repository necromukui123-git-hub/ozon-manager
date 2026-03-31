package jwt

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"ozon-manager/internal/config"
)

var (
	ErrInvalidToken = errors.New("invalid token")
	ErrExpiredToken = errors.New("token has expired")
)

const (
	TokenTypeAccess = "access"
)

// Claims 自定义JWT声明
type Claims struct {
	UserID      uint   `json:"user_id"`
	Username    string `json:"username"`
	DisplayName string `json:"display_name"`
	Role        string `json:"role"`
	TokenType   string `json:"token_type"`
	jwt.RegisteredClaims
}

// GenerateAccessToken 生成 access token
func GenerateAccessToken(userID uint, username, displayName, role string) (string, error) {
	cfg := config.GetConfig()
	now := time.Now()

	claims := Claims{
		UserID:      userID,
		Username:    username,
		DisplayName: displayName,
		Role:        role,
		TokenType:   TokenTypeAccess,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(accessTokenTTL(cfg.JWT))),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			Issuer:    "ozon-manager",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(cfg.JWT.Secret))
}

// GenerateToken 兼容旧调用方，内部生成 access token
func GenerateToken(userID uint, username, displayName, role string) (string, error) {
	return GenerateAccessToken(userID, username, displayName, role)
}

// ParseToken 解析JWT令牌
func ParseToken(tokenString string) (*Claims, error) {
	cfg := config.GetConfig()

	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(cfg.JWT.Secret), nil
	})

	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrExpiredToken
		}
		return nil, ErrInvalidToken
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		if claims.TokenType != TokenTypeAccess {
			return nil, ErrInvalidToken
		}
		return claims, nil
	}

	return nil, ErrInvalidToken
}

func accessTokenTTL(cfg config.JWTConfig) time.Duration {
	if cfg.AccessExpireMinutes > 0 {
		return time.Duration(cfg.AccessExpireMinutes) * time.Minute
	}
	if cfg.ExpireHours > 0 {
		return time.Duration(cfg.ExpireHours) * time.Hour
	}
	return 24 * time.Hour
}
