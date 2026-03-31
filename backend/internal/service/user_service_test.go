package service

import (
	"strings"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"ozon-manager/internal/model"
	"ozon-manager/internal/repository"
	"ozon-manager/pkg/hash"
)

func openUserServiceTestDB(t *testing.T) *gorm.DB {
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

func createUserServiceTestUser(t *testing.T, db *gorm.DB, username, role, status string, ownerID *uint, password string) *model.User {
	t.Helper()

	passwordHash, err := HashPassword(password)
	if err != nil {
		t.Fatalf("HashPassword(): %v", err)
	}

	user := &model.User{
		Username:     username,
		PasswordHash: passwordHash,
		DisplayName:  username,
		Role:         role,
		Status:       status,
		OwnerID:      ownerID,
	}
	if err := db.Create(user).Error; err != nil {
		t.Fatalf("db.Create(user): %v", err)
	}

	return user
}

func seedUserRefreshToken(t *testing.T, repo *repository.RefreshTokenRepository, userID uint, suffix string) {
	t.Helper()

	if err := repo.Create(&model.UserRefreshToken{
		UserID:    userID,
		TokenHash: "token-hash-" + suffix,
		FamilyID:  "family-" + suffix,
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}); err != nil {
		t.Fatalf("repo.Create(refresh_token): %v", err)
	}
}

func countActiveRefreshTokens(t *testing.T, db *gorm.DB, userID uint) int64 {
	t.Helper()

	var count int64
	if err := db.Model(&model.UserRefreshToken{}).
		Where("user_id = ? AND revoked_at IS NULL", userID).
		Count(&count).Error; err != nil {
		t.Fatalf("count active refresh tokens: %v", err)
	}

	return count
}

func TestChangePasswordRevokesAllRefreshTokens(t *testing.T) {
	db := openUserServiceTestDB(t)
	userRepo := repository.NewUserRepository(db)
	shopRepo := repository.NewShopRepository(db)
	refreshRepo := repository.NewRefreshTokenRepository(db)
	userService := NewUserService(userRepo, shopRepo, refreshRepo)

	oldPassword := hash.SHA256Hash("old-password")
	user := createUserServiceTestUser(t, db, "alice", model.RoleStaff, "active", nil, oldPassword)
	seedUserRefreshToken(t, refreshRepo, user.ID, "a")
	seedUserRefreshToken(t, refreshRepo, user.ID, "b")

	if err := userService.ChangePassword(user.ID, oldPassword, hash.SHA256Hash("new-password")); err != nil {
		t.Fatalf("ChangePassword(): %v", err)
	}

	if count := countActiveRefreshTokens(t, db, user.ID); count != 0 {
		t.Fatalf("active refresh token count = %d, want 0", count)
	}
}

func TestUpdateShopAdminStatusDisabledRevokesAllRefreshTokens(t *testing.T) {
	db := openUserServiceTestDB(t)
	userRepo := repository.NewUserRepository(db)
	shopRepo := repository.NewShopRepository(db)
	refreshRepo := repository.NewRefreshTokenRepository(db)
	userService := NewUserService(userRepo, shopRepo, refreshRepo)

	user := createUserServiceTestUser(t, db, "shop-admin", model.RoleShopAdmin, "active", nil, hash.SHA256Hash("password"))
	seedUserRefreshToken(t, refreshRepo, user.ID, "shop-admin-disable")

	if err := userService.UpdateShopAdminStatus(user.ID, "disabled"); err != nil {
		t.Fatalf("UpdateShopAdminStatus(): %v", err)
	}

	if count := countActiveRefreshTokens(t, db, user.ID); count != 0 {
		t.Fatalf("active refresh token count = %d, want 0", count)
	}
}

func TestResetShopAdminPasswordRevokesAllRefreshTokens(t *testing.T) {
	db := openUserServiceTestDB(t)
	userRepo := repository.NewUserRepository(db)
	shopRepo := repository.NewShopRepository(db)
	refreshRepo := repository.NewRefreshTokenRepository(db)
	userService := NewUserService(userRepo, shopRepo, refreshRepo)

	user := createUserServiceTestUser(t, db, "shop-admin-reset", model.RoleShopAdmin, "active", nil, hash.SHA256Hash("password"))
	seedUserRefreshToken(t, refreshRepo, user.ID, "shop-admin-reset")

	if err := userService.ResetShopAdminPassword(user.ID, hash.SHA256Hash("new-password")); err != nil {
		t.Fatalf("ResetShopAdminPassword(): %v", err)
	}

	if count := countActiveRefreshTokens(t, db, user.ID); count != 0 {
		t.Fatalf("active refresh token count = %d, want 0", count)
	}
}

func TestUpdateStaffStatusDisabledRevokesAllRefreshTokens(t *testing.T) {
	db := openUserServiceTestDB(t)
	userRepo := repository.NewUserRepository(db)
	shopRepo := repository.NewShopRepository(db)
	refreshRepo := repository.NewRefreshTokenRepository(db)
	userService := NewUserService(userRepo, shopRepo, refreshRepo)

	owner := createUserServiceTestUser(t, db, "owner", model.RoleShopAdmin, "active", nil, hash.SHA256Hash("password"))
	staff := createUserServiceTestUser(t, db, "staff", model.RoleStaff, "active", &owner.ID, hash.SHA256Hash("password"))
	seedUserRefreshToken(t, refreshRepo, staff.ID, "staff-disable")

	if err := userService.UpdateStaffStatus(staff.ID, "disabled", owner.ID); err != nil {
		t.Fatalf("UpdateStaffStatus(): %v", err)
	}

	if count := countActiveRefreshTokens(t, db, staff.ID); count != 0 {
		t.Fatalf("active refresh token count = %d, want 0", count)
	}
}

func TestResetStaffPasswordRevokesAllRefreshTokens(t *testing.T) {
	db := openUserServiceTestDB(t)
	userRepo := repository.NewUserRepository(db)
	shopRepo := repository.NewShopRepository(db)
	refreshRepo := repository.NewRefreshTokenRepository(db)
	userService := NewUserService(userRepo, shopRepo, refreshRepo)

	owner := createUserServiceTestUser(t, db, "owner-reset", model.RoleShopAdmin, "active", nil, hash.SHA256Hash("password"))
	staff := createUserServiceTestUser(t, db, "staff-reset", model.RoleStaff, "active", &owner.ID, hash.SHA256Hash("password"))
	seedUserRefreshToken(t, refreshRepo, staff.ID, "staff-reset")

	if err := userService.ResetStaffPassword(staff.ID, hash.SHA256Hash("new-password"), owner.ID); err != nil {
		t.Fatalf("ResetStaffPassword(): %v", err)
	}

	if count := countActiveRefreshTokens(t, db, staff.ID); count != 0 {
		t.Fatalf("active refresh token count = %d, want 0", count)
	}
}
