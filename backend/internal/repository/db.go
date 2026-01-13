package repository

import (
	"fmt"

	"ozon-manager/internal/config"
	"ozon-manager/internal/model"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// InitDB initializes the database connection.
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

// AutoMigrate runs GORM auto migrations.
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

// EnsureOwnerColumns adds owner_id columns and indexes when the schema is missing them.
func EnsureOwnerColumns(db *gorm.DB) error {
	if err := db.Exec("ALTER TABLE users ADD COLUMN IF NOT EXISTS owner_id INTEGER").Error; err != nil {
		return fmt.Errorf("failed to add users.owner_id: %w", err)
	}
	if err := db.Exec("CREATE INDEX IF NOT EXISTS idx_users_owner_id ON users(owner_id)").Error; err != nil {
		return fmt.Errorf("failed to create users.owner_id index: %w", err)
	}
	if err := db.Exec("ALTER TABLE shops ADD COLUMN IF NOT EXISTS owner_id INTEGER").Error; err != nil {
		return fmt.Errorf("failed to add shops.owner_id: %w", err)
	}
	if err := db.Exec("CREATE INDEX IF NOT EXISTS idx_shops_owner_id ON shops(owner_id)").Error; err != nil {
		return fmt.Errorf("failed to create shops.owner_id index: %w", err)
	}

	return nil
}

// CreateSuperAdminUser creates the default super admin account if missing.
func CreateSuperAdminUser(db *gorm.DB) error {
	var count int64
	db.Model(&model.User{}).Where("role = ?", model.RoleSuperAdmin).Count(&count)

	if count == 0 {
		// Default password: admin123
		//passwordHash := "$2a$10$N9qo8uLOickgx2ZMRZoMy.MqrqBuBk0F.Gc7YMG.T9D.Z2OVOQHMu"
		passwordHash := "$2a$10$ylb8XwllNQUWAlciq5nxiev6eFJk4FqSQmU2XI04Pg9qi2rb178wq"

		superAdmin := &model.User{
			Username:     "super_admin",
			PasswordHash: passwordHash,
			DisplayName:  "系统管理员",
			Role:         model.RoleSuperAdmin,
			Status:       "active",
		}

		return db.Create(superAdmin).Error
	}

	return nil
}
