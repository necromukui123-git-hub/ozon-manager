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

func TestInitDatabaseIncludesUserRefreshTokensTable(t *testing.T) {
	content, err := os.ReadFile("init_database.sql")
	if err != nil {
		t.Fatalf("read init_database.sql: %v", err)
	}

	sql := string(content)
	if !strings.Contains(sql, "CREATE TABLE IF NOT EXISTS user_refresh_tokens (") {
		t.Fatal("user_refresh_tokens table is missing in init_database.sql")
	}

	if !strings.Contains(sql, "token_hash") {
		t.Fatal("user_refresh_tokens.token_hash column is missing in init_database.sql")
	}

	if !strings.Contains(sql, "CREATE UNIQUE INDEX IF NOT EXISTS idx_user_refresh_tokens_token_hash ON user_refresh_tokens(token_hash);") {
		t.Fatal("user_refresh_tokens.token_hash unique index is missing in init_database.sql")
	}

	if !strings.Contains(sql, "CREATE INDEX IF NOT EXISTS idx_user_refresh_tokens_family_id ON user_refresh_tokens(family_id);") {
		t.Fatal("user_refresh_tokens.family_id index is missing in init_database.sql")
	}
}

func TestInitDatabaseIncludesAutoPromotionDateRangeColumns(t *testing.T) {
	content, err := os.ReadFile("init_database.sql")
	if err != nil {
		t.Fatalf("read init_database.sql: %v", err)
	}

	sql := string(content)
	configPattern := regexp.MustCompile(`(?s)CREATE TABLE IF NOT EXISTS auto_promotion_configs \((.*?)\);`)
	configMatch := configPattern.FindStringSubmatch(sql)
	if len(configMatch) != 2 {
		t.Fatal("failed to locate auto_promotion_configs table definition in init_database.sql")
	}
	if !strings.Contains(configMatch[1], "target_date_end     DATE") {
		t.Fatal("auto_promotion_configs.target_date_end column is missing in init_database.sql")
	}

	runPattern := regexp.MustCompile(`(?s)CREATE TABLE IF NOT EXISTS auto_promotion_runs \((.*?)\);`)
	runMatch := runPattern.FindStringSubmatch(sql)
	if len(runMatch) != 2 {
		t.Fatal("failed to locate auto_promotion_runs table definition in init_database.sql")
	}
	if !strings.Contains(runMatch[1], "target_date_end     DATE NOT NULL") {
		t.Fatal("auto_promotion_runs.target_date_end column is missing in init_database.sql")
	}
}
