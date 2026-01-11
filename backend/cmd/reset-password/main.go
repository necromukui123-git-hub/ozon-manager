package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"ozon-manager/internal/config"
)

func main() {
	// 命令行参数
	username := flag.String("username", "", "要重置密码的用户名")
	password := flag.String("password", "", "新密码（至少6位）")
	configPath := flag.String("config", "config/config.yaml", "配置文件路径")
	flag.Parse()

	// 验证参数
	if *username == "" {
		fmt.Println("错误: 请提供用户名 (-username)")
		fmt.Println("用法: go run cmd/reset-password/main.go -username <用户名> -password <新密码>")
		os.Exit(1)
	}

	if *password == "" {
		fmt.Println("错误: 请提供新密码 (-password)")
		fmt.Println("用法: go run cmd/reset-password/main.go -username <用户名> -password <新密码>")
		os.Exit(1)
	}

	if len(*password) < 6 {
		fmt.Println("错误: 密码长度至少为6位")
		os.Exit(1)
	}

	// 加载配置
	cfg, err := config.LoadConfig(*configPath)
	if err != nil {
		log.Fatalf("加载配置失败: %v", err)
	}

	// 连接数据库
	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%d sslmode=disable",
		cfg.Database.Host,
		cfg.Database.User,
		cfg.Database.Password,
		cfg.Database.DBName,
		cfg.Database.Port,
	)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("连接数据库失败: %v", err)
	}

	// 检查用户是否存在
	var count int64
	db.Table("users").Where("username = ?", *username).Count(&count)
	if count == 0 {
		fmt.Printf("错误: 用户 '%s' 不存在\n", *username)
		os.Exit(1)
	}

	// 生成密码哈希
	passwordHash, err := bcrypt.GenerateFromPassword([]byte(*password), bcrypt.DefaultCost)
	if err != nil {
		log.Fatalf("生成密码哈希失败: %v", err)
	}

	// 更新密码
	result := db.Table("users").Where("username = ?", *username).Update("password_hash", string(passwordHash))
	if result.Error != nil {
		log.Fatalf("更新密码失败: %v", result.Error)
	}

	if result.RowsAffected == 0 {
		fmt.Println("警告: 没有更新任何记录")
		os.Exit(1)
	}

	fmt.Printf("成功: 用户 '%s' 的密码已重置\n", *username)
}
