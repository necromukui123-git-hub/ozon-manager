package repository

import (
	"strings"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"ozon-manager/internal/model"
)

func openRefreshTokenTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	dsn := "file:" + strings.ReplaceAll(t.Name(), "/", "_") + "?mode=memory&cache=shared"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("gorm.Open(): %v", err)
	}

	if err := db.AutoMigrate(&model.UserRefreshToken{}); err != nil {
		t.Fatalf("AutoMigrate(): %v", err)
	}

	return db
}

func TestRefreshTokenRepositoryCreateFindAndRevoke(t *testing.T) {
	db := openRefreshTokenTestDB(t)
	repo := NewRefreshTokenRepository(db)

	token := &model.UserRefreshToken{
		UserID:    7,
		TokenHash: "hash-1",
		FamilyID:  "family-1",
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}

	if err := repo.Create(token); err != nil {
		t.Fatalf("Create(): %v", err)
	}

	found, err := repo.FindActiveByTokenHash("hash-1")
	if err != nil {
		t.Fatalf("FindActiveByTokenHash(): %v", err)
	}
	if found.ID != token.ID {
		t.Fatalf("FindActiveByTokenHash() id = %d, want %d", found.ID, token.ID)
	}

	if err := repo.RevokeByID(token.ID, "logout"); err != nil {
		t.Fatalf("RevokeByID(): %v", err)
	}

	if _, err := repo.FindActiveByTokenHash("hash-1"); err == nil {
		t.Fatal("FindActiveByTokenHash() expected revoked token to be hidden")
	}
}
