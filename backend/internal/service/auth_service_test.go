package service

import (
	"strings"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"ozon-manager/internal/config"
	"ozon-manager/internal/dto"
	"ozon-manager/internal/model"
	"ozon-manager/internal/repository"
	"ozon-manager/pkg/hash"
)

func setAuthServiceTestConfig(t *testing.T) {
	t.Helper()

	previous := config.GlobalConfig
	config.GlobalConfig = &config.Config{
		JWT: config.JWTConfig{
			Secret:              "service-test-secret",
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

func openAuthServiceTestDB(t *testing.T) *gorm.DB {
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

func createAuthServiceTestUser(t *testing.T, db *gorm.DB, username, role, status, passwordHash string) *model.User {
	t.Helper()

	user := &model.User{
		Username:     username,
		PasswordHash: passwordHash,
		DisplayName:  username,
		Role:         role,
		Status:       status,
	}

	if err := db.Create(user).Error; err != nil {
		t.Fatalf("db.Create(user): %v", err)
	}

	return user
}

func TestLoginIssuesAccessTokenAndRefreshToken(t *testing.T) {
	setAuthServiceTestConfig(t)
	db := openAuthServiceTestDB(t)

	userRepo := repository.NewUserRepository(db)
	shopRepo := repository.NewShopRepository(db)
	refreshRepo := repository.NewRefreshTokenRepository(db)
	authService := NewAuthService(userRepo, shopRepo, refreshRepo)

	clientPassword := hash.SHA256Hash("password123")
	storedPassword, err := HashPassword(clientPassword)
	if err != nil {
		t.Fatalf("HashPassword(): %v", err)
	}

	user := createAuthServiceTestUser(t, db, "alice", model.RoleShopAdmin, "active", storedPassword)

	session, err := authService.Login(&dto.LoginRequest{
		Username: "alice",
		Password: clientPassword,
	}, AuthSessionMeta{
		UserAgent: "service-test",
		IPAddress: "127.0.0.1",
	})
	if err != nil {
		t.Fatalf("Login(): %v", err)
	}

	if session.RefreshToken == "" {
		t.Fatal("Login() expected refresh token to be issued")
	}
	if session.Response.Token == "" {
		t.Fatal("Login() expected access token to be issued")
	}
	if session.Response.TokenExpiresAt.IsZero() {
		t.Fatal("Login() expected token_expires_at to be populated")
	}

	stored, err := refreshRepo.FindActiveByTokenHash(hash.SHA256Hash(session.RefreshToken))
	if err != nil {
		t.Fatalf("FindActiveByTokenHash(): %v", err)
	}
	if stored.UserID != user.ID {
		t.Fatalf("stored refresh token user_id = %d, want %d", stored.UserID, user.ID)
	}
	if stored.FamilyID == "" {
		t.Fatal("stored refresh token expected family_id to be populated")
	}
}

func TestRefreshRotatesRefreshToken(t *testing.T) {
	setAuthServiceTestConfig(t)
	db := openAuthServiceTestDB(t)

	userRepo := repository.NewUserRepository(db)
	shopRepo := repository.NewShopRepository(db)
	refreshRepo := repository.NewRefreshTokenRepository(db)
	authService := NewAuthService(userRepo, shopRepo, refreshRepo)

	clientPassword := hash.SHA256Hash("password123")
	storedPassword, err := HashPassword(clientPassword)
	if err != nil {
		t.Fatalf("HashPassword(): %v", err)
	}

	createAuthServiceTestUser(t, db, "bob", model.RoleStaff, "active", storedPassword)

	initial, err := authService.Login(&dto.LoginRequest{
		Username: "bob",
		Password: clientPassword,
	}, AuthSessionMeta{
		UserAgent: "service-test",
		IPAddress: "127.0.0.1",
	})
	if err != nil {
		t.Fatalf("Login(): %v", err)
	}

	refreshed, err := authService.Refresh(initial.RefreshToken, AuthSessionMeta{
		UserAgent: "service-test-refresh",
		IPAddress: "127.0.0.2",
	})
	if err != nil {
		t.Fatalf("Refresh(): %v", err)
	}

	if refreshed.RefreshToken == "" {
		t.Fatal("Refresh() expected new refresh token")
	}
	if refreshed.RefreshToken == initial.RefreshToken {
		t.Fatal("Refresh() expected refresh token to rotate")
	}

	var previous model.UserRefreshToken
	if err := db.Where("token_hash = ?", hash.SHA256Hash(initial.RefreshToken)).First(&previous).Error; err != nil {
		t.Fatalf("query previous refresh token: %v", err)
	}
	if previous.RevokedAt == nil {
		t.Fatal("Refresh() expected previous refresh token to be revoked")
	}

	current, err := refreshRepo.FindActiveByTokenHash(hash.SHA256Hash(refreshed.RefreshToken))
	if err != nil {
		t.Fatalf("FindActiveByTokenHash(new): %v", err)
	}
	if current.FamilyID != previous.FamilyID {
		t.Fatalf("refresh token family_id = %q, want %q", current.FamilyID, previous.FamilyID)
	}
}
