package repository

import (
	"fmt"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"ozon-manager/internal/config"
	"ozon-manager/internal/model"
)

// InitDB 初始化数据库连接
func InitDB(cfg *config.DatabaseConfig) (*gorm.DB, error) {
	dsn := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.DBName, cfg.SSLMode,
	)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect database: %w", err)
	}

	return db, nil
}

// AutoMigrate 自动迁移数据库表结构
func AutoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(
		&model.User{},
		&model.Shop{},
		&model.UserShop{},
		&model.Product{},
		&model.LossProduct{},
		&model.PromotedProduct{},
		&model.PromotionAction{},
		&model.OperationLog{},
	)
}

// CreateAdminUser 创建默认管理员账号
func CreateAdminUser(db *gorm.DB) error {
	var count int64
	db.Model(&model.User{}).Where("role = ?", "admin").Count(&count)

	if count == 0 {
		// 默认密码: admin123
		passwordHash := "$2a$10$N9qo8uLOickgx2ZMRZoMy.MqrqBuBk0F.Gc7YMG.T9D.Z2OVOQHMu"

		admin := &model.User{
			Username:     "admin",
			PasswordHash: passwordHash,
			DisplayName:  "系统管理员",
			Role:         "admin",
			Status:       "active",
		}

		return db.Create(admin).Error
	}

	return nil
}
