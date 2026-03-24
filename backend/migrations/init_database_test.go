package migrations_test

import (
	"os"
	"regexp"
	"strings"
	"testing"

	"golang.org/x/crypto/bcrypt"
	"ozon-manager/pkg/hash"
)

func TestInitDatabaseSuperAdminPasswordMatchesLoginContract(t *testing.T) {
	content, err := os.ReadFile("init_database.sql")
	if err != nil {
		t.Fatalf("read init_database.sql: %v", err)
	}

	pattern := regexp.MustCompile(`VALUES \('super_admin', '([^']+)'`)
	match := pattern.FindStringSubmatch(string(content))
	if len(match) != 2 {
		t.Fatal("failed to find super_admin password hash in init_database.sql")
	}

	storedHash := match[1]
	clientHash := hash.SHA256Hash("admin123")

	if err := bcrypt.CompareHashAndPassword([]byte(storedHash), []byte(clientHash)); err != nil {
		t.Fatalf("super_admin hash does not match bcrypt(SHA256(admin123)): %v", err)
	}

	if err := bcrypt.CompareHashAndPassword([]byte(storedHash), []byte("admin123")); err == nil {
		t.Fatal("super_admin hash should not match raw admin123; login sends SHA-256 first")
	}
}

func TestInitDatabaseUsersTableIncludesOwnerID(t *testing.T) {
	content, err := os.ReadFile("init_database.sql")
	if err != nil {
		t.Fatalf("read init_database.sql: %v", err)
	}

	sql := string(content)
	usersTablePattern := regexp.MustCompile(`(?s)CREATE TABLE IF NOT EXISTS users \((.*?)\);`)
	usersTableMatch := usersTablePattern.FindStringSubmatch(sql)
	if len(usersTableMatch) != 2 {
		t.Fatal("failed to locate users table definition in init_database.sql")
	}

	usersTableDDL := usersTableMatch[1]

	if !strings.Contains(usersTableDDL, "owner_id        INTEGER REFERENCES users(id)") {
		t.Fatal("users table is missing owner_id column in init_database.sql")
	}

	if !strings.Contains(sql, "CREATE INDEX IF NOT EXISTS idx_users_owner_id ON users(owner_id);") {
		t.Fatal("users.owner_id index is missing in init_database.sql")
	}
}
