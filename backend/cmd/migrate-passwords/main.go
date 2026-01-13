package main

import (
	"fmt"
	"log"

	"golang.org/x/crypto/bcrypt"
	"ozon-manager/internal/config"
	"ozon-manager/internal/repository"
	"ozon-manager/pkg/hash"
)

// 开发环境密码迁移工具
// 将测试账户的密码重新哈希为 BCrypt(SHA256(原密码)) 格式

func main() {
	// 加载配置
	cfg, err := config.LoadConfig("config/config.yaml")
	if err != nil {
		log.Fatal("加载配置失败:", err)
	}

	// 连接数据库
	db, err := repository.InitDB(&cfg.Database)
	if err != nil {
		log.Fatal("数据库连接失败:", err)
	}

	fmt.Println("🔐 开始迁移测试账户密码...")
	fmt.Println("新格式: BCrypt(SHA256(原密码))")
	fmt.Println()

	// 测试账户及其明文密码
	testAccounts := map[string]string{
		"super_admin": "admin123",
	}

	successCount := 0
	for username, plainPassword := range testAccounts {
		// 1. SHA-256 预哈希 (模拟前端行为)
		sha256Hash := hash.SHA256Hash(plainPassword)

		// 2. BCrypt 二次哈希 (后端存储格式)
		bcryptHash, err := bcrypt.GenerateFromPassword([]byte(sha256Hash), bcrypt.DefaultCost)
		if err != nil {
			log.Printf("❌ %s 密码哈希失败: %v", username, err)
			continue
		}

		// 3. 更新数据库
		result := db.Exec("UPDATE users SET password_hash = ? WHERE username = ?", string(bcryptHash), username)
		if result.Error != nil {
			log.Printf("❌ %s 数据库更新失败: %v", username, result.Error)
			continue
		}

		if result.RowsAffected == 0 {
			log.Printf("⚠️  %s: 用户不存在,跳过", username)
			continue
		}

		fmt.Printf("✅ %s: 密码已迁移\n", username)
		fmt.Printf("   原密码: %s\n", plainPassword)
		fmt.Printf("   SHA-256: %s...\n", sha256Hash[:16])
		fmt.Printf("   BCrypt: %s...\n\n", string(bcryptHash)[:29])
		successCount++
	}

	fmt.Printf("🎉 迁移完成: %d/%d 个账户成功\n", successCount, len(testAccounts))

	if successCount < len(testAccounts) {
		fmt.Println("\n提示: 部分账户未成功迁移,请检查数据库中是否存在这些用户")
	}
}
