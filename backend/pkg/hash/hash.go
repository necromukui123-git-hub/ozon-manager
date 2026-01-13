package hash

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
)

// SHA256Hash 计算 SHA-256 哈希
func SHA256Hash(input string) string {
	h := sha256.New()
	h.Write([]byte(input))
	return hex.EncodeToString(h.Sum(nil))
}

// SHA256HashWithSalt 带盐值的 SHA-256 哈希
func SHA256HashWithSalt(password, username string) string {
	salt := strings.ToLower(username)
	return SHA256Hash(password + salt)
}
