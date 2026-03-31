package jwt

import (
	"testing"
	"time"

	golangjwt "github.com/golang-jwt/jwt/v5"
	"ozon-manager/internal/config"
)

func setTestConfig(t *testing.T, accessExpireMinutes int) {
	t.Helper()

	previous := config.GlobalConfig
	config.GlobalConfig = &config.Config{
		JWT: config.JWTConfig{
			Secret:              "test-secret",
			AccessExpireMinutes: accessExpireMinutes,
		},
	}

	t.Cleanup(func() {
		config.GlobalConfig = previous
	})
}

func TestGenerateAccessTokenUsesAccessTTLAndType(t *testing.T) {
	setTestConfig(t, 15)

	start := time.Now()
	tokenString, err := GenerateAccessToken(42, "alice", "Alice", "shop_admin")
	if err != nil {
		t.Fatalf("GenerateAccessToken() unexpected error: %v", err)
	}

	token, err := golangjwt.Parse(tokenString, func(token *golangjwt.Token) (interface{}, error) {
		return []byte(config.GlobalConfig.JWT.Secret), nil
	})
	if err != nil {
		t.Fatalf("Parse() unexpected error: %v", err)
	}

	claims, ok := token.Claims.(golangjwt.MapClaims)
	if !ok {
		t.Fatalf("token claims type = %T, want MapClaims", token.Claims)
	}

	if got := claims["token_type"]; got != "access" {
		t.Fatalf("token_type = %v, want access", got)
	}

	expUnix, ok := claims["exp"].(float64)
	if !ok {
		t.Fatalf("exp claim type = %T, want float64", claims["exp"])
	}

	ttl := time.Unix(int64(expUnix), 0).Sub(start)
	if ttl < 14*time.Minute || ttl > 16*time.Minute {
		t.Fatalf("token ttl = %v, want about 15m", ttl)
	}
}

func TestParseTokenRejectsRefreshTokenType(t *testing.T) {
	setTestConfig(t, 15)

	now := time.Now()
	tokenString, err := golangjwt.NewWithClaims(golangjwt.SigningMethodHS256, golangjwt.MapClaims{
		"user_id":      7,
		"username":     "bob",
		"display_name": "Bob",
		"role":         "staff",
		"token_type":   "refresh",
		"iss":          "ozon-manager",
		"iat":          now.Unix(),
		"nbf":          now.Unix(),
		"exp":          now.Add(time.Hour).Unix(),
	}).SignedString([]byte(config.GlobalConfig.JWT.Secret))
	if err != nil {
		t.Fatalf("SignedString() unexpected error: %v", err)
	}

	_, err = ParseToken(tokenString)
	if err != ErrInvalidToken {
		t.Fatalf("ParseToken() error = %v, want %v", err, ErrInvalidToken)
	}
}
