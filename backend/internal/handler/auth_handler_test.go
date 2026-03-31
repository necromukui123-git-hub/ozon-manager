package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"ozon-manager/internal/config"
	"ozon-manager/internal/dto"
	"ozon-manager/internal/model"
	"ozon-manager/internal/repository"
	"ozon-manager/internal/service"
	"ozon-manager/pkg/hash"
)

func setAuthHandlerTestConfig(t *testing.T) {
	t.Helper()

	previous := config.GlobalConfig
	config.GlobalConfig = &config.Config{
		JWT: config.JWTConfig{
			Secret:              "handler-test-secret",
			AccessExpireMinutes: 15,
			RefreshExpireHours:  24,
			RefreshCookieName:   "ozon_refresh_token",
			RefreshCookieSecure: false,
		},
	}

	t.Cleanup(func() {
		config.GlobalConfig = previous
	})
}

func openAuthHandlerTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	dsn := "file:" + strings.ReplaceAll(t.Name(), "/", "_") + "?mode=memory&cache=shared"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("gorm.Open(): %v", err)
	}

	if err := db.AutoMigrate(&model.User{}, &model.Shop{}, &model.UserShop{}, &model.UserRefreshToken{}); err != nil {
		t.Fatalf("AutoMigrate(): %v", err)
	}

	return db
}

func createAuthHandlerTestUser(t *testing.T, db *gorm.DB, username string) string {
	t.Helper()

	clientPassword := hash.SHA256Hash("password123")
	storedPassword, err := service.HashPassword(clientPassword)
	if err != nil {
		t.Fatalf("HashPassword(): %v", err)
	}

	user := &model.User{
		Username:     username,
		PasswordHash: storedPassword,
		DisplayName:  username,
		Role:         model.RoleShopAdmin,
		Status:       "active",
	}
	if err := db.Create(user).Error; err != nil {
		t.Fatalf("db.Create(user): %v", err)
	}

	return clientPassword
}

func newAuthHandlerTestRouter(t *testing.T, db *gorm.DB) *gin.Engine {
	t.Helper()

	gin.SetMode(gin.TestMode)
	userRepo := repository.NewUserRepository(db)
	shopRepo := repository.NewShopRepository(db)
	refreshRepo := repository.NewRefreshTokenRepository(db)
	authService := service.NewAuthService(userRepo, shopRepo, refreshRepo)
	authHandler := NewAuthHandler(authService)

	router := gin.New()
	router.POST("/api/v1/auth/login", authHandler.Login)
	router.POST("/api/v1/auth/refresh", authHandler.Refresh)
	router.POST("/api/v1/auth/logout", authHandler.Logout)
	return router
}

func TestLoginSetsRefreshCookie(t *testing.T) {
	setAuthHandlerTestConfig(t)
	db := openAuthHandlerTestDB(t)
	password := createAuthHandlerTestUser(t, db, "alice")
	router := newAuthHandlerTestRouter(t, db)

	body, err := json.Marshal(dto.LoginRequest{
		Username: "alice",
		Password: password,
	})
	if err != nil {
		t.Fatalf("json.Marshal(): %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body=%s", recorder.Code, http.StatusOK, recorder.Body.String())
	}

	cookies := recorder.Result().Cookies()
	if len(cookies) == 0 {
		t.Fatal("login expected refresh cookie to be set")
	}
	if cookies[0].Name != "ozon_refresh_token" {
		t.Fatalf("cookie name = %q, want ozon_refresh_token", cookies[0].Name)
	}
	if !cookies[0].HttpOnly {
		t.Fatal("refresh cookie expected HttpOnly=true")
	}
	if !bytes.Contains(recorder.Body.Bytes(), []byte("token_expires_at")) {
		t.Fatal("login response expected token_expires_at")
	}
}

func TestRefreshRotatesCookie(t *testing.T) {
	setAuthHandlerTestConfig(t)
	db := openAuthHandlerTestDB(t)
	password := createAuthHandlerTestUser(t, db, "bob")
	router := newAuthHandlerTestRouter(t, db)

	loginBody, err := json.Marshal(dto.LoginRequest{
		Username: "bob",
		Password: password,
	})
	if err != nil {
		t.Fatalf("json.Marshal(login): %v", err)
	}

	loginReq := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(loginBody))
	loginReq.Header.Set("Content-Type", "application/json")
	loginRecorder := httptest.NewRecorder()
	router.ServeHTTP(loginRecorder, loginReq)
	if loginRecorder.Code != http.StatusOK {
		t.Fatalf("login status = %d, body=%s", loginRecorder.Code, loginRecorder.Body.String())
	}

	loginCookie := loginRecorder.Result().Cookies()[0]

	refreshReq := httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", nil)
	refreshReq.AddCookie(loginCookie)
	refreshRecorder := httptest.NewRecorder()
	router.ServeHTTP(refreshRecorder, refreshReq)

	if refreshRecorder.Code != http.StatusOK {
		t.Fatalf("refresh status = %d, want %d, body=%s", refreshRecorder.Code, http.StatusOK, refreshRecorder.Body.String())
	}

	refreshCookies := refreshRecorder.Result().Cookies()
	if len(refreshCookies) == 0 {
		t.Fatal("refresh expected rotated cookie")
	}
	if refreshCookies[0].Value == loginCookie.Value {
		t.Fatal("refresh expected cookie value to rotate")
	}
}

func TestLogoutRevokesRefreshTokenWithoutAccessToken(t *testing.T) {
	setAuthHandlerTestConfig(t)
	db := openAuthHandlerTestDB(t)
	password := createAuthHandlerTestUser(t, db, "carol")
	router := newAuthHandlerTestRouter(t, db)

	loginBody, err := json.Marshal(dto.LoginRequest{
		Username: "carol",
		Password: password,
	})
	if err != nil {
		t.Fatalf("json.Marshal(login): %v", err)
	}

	loginReq := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(loginBody))
	loginReq.Header.Set("Content-Type", "application/json")
	loginRecorder := httptest.NewRecorder()
	router.ServeHTTP(loginRecorder, loginReq)
	if loginRecorder.Code != http.StatusOK {
		t.Fatalf("login status = %d, body=%s", loginRecorder.Code, loginRecorder.Body.String())
	}

	loginCookie := loginRecorder.Result().Cookies()[0]

	logoutReq := httptest.NewRequest(http.MethodPost, "/api/v1/auth/logout", nil)
	logoutReq.AddCookie(loginCookie)
	logoutRecorder := httptest.NewRecorder()
	router.ServeHTTP(logoutRecorder, logoutReq)

	if logoutRecorder.Code != http.StatusOK {
		t.Fatalf("logout status = %d, want %d, body=%s", logoutRecorder.Code, http.StatusOK, logoutRecorder.Body.String())
	}

	var stored model.UserRefreshToken
	if err := db.Where("token_hash = ?", hash.SHA256Hash(loginCookie.Value)).First(&stored).Error; err != nil {
		t.Fatalf("query refresh token after logout: %v", err)
	}
	if stored.RevokedAt == nil {
		t.Fatal("logout expected refresh token to be revoked")
	}
}
