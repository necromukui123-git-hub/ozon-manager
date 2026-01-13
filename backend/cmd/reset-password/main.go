package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"ozon-manager/internal/config"
	"ozon-manager/pkg/hash"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func main() {
	username := flag.String("username", "", "username to reset")
	password := flag.String("password", "", "new password (min 6 chars)")
	configPath := flag.String("config", "config/config.yaml", "config file path")
	flag.Parse()

	if *username == "" {
		fmt.Println("error: missing -username")
		fmt.Println("usage: go run cmd/reset-password/main.go -username <user> -password <new_password>")
		os.Exit(1)
	}

	if *password == "" {
		fmt.Println("error: missing -password")
		fmt.Println("usage: go run cmd/reset-password/main.go -username <user> -password <new_password>")
		os.Exit(1)
	}

	if len(*password) < 6 {
		fmt.Println("error: password must be at least 6 characters")
		os.Exit(1)
	}

	cfg, err := config.LoadConfig(*configPath)
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	dsn := fmt.Sprintf(
		"host=%s user=%s password=%s dbname=%s port=%d sslmode=disable",
		cfg.Database.Host,
		cfg.Database.User,
		cfg.Database.Password,
		cfg.Database.DBName,
		cfg.Database.Port,
	)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("failed to connect database: %v", err)
	}

	var count int64
	db.Table("users").Where("username = ?", *username).Count(&count)
	if count == 0 {
		fmt.Printf("error: user '%s' not found", *username)
		os.Exit(1)
	}

	// Match login flow: bcrypt(sha256(password))
	sha := hash.SHA256Hash(*password)
	passwordHash, err := bcrypt.GenerateFromPassword([]byte(sha), bcrypt.DefaultCost)
	if err != nil {
		log.Fatalf("failed to generate password hash: %v", err)
	}

	result := db.Table("users").Where("username = ?", *username).Update("password_hash", string(passwordHash))
	if result.Error != nil {
		log.Fatalf("failed to update password: %v", result.Error)
	}

	if result.RowsAffected == 0 {
		fmt.Println("warning: no rows updated")
		os.Exit(1)
	}

	fmt.Printf("success: password reset for '%s'", *username)
}
