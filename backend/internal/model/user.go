package model

import (
	"time"
)

// User 用户表
type User struct {
	ID           uint       `gorm:"primaryKey" json:"id"`
	Username     string     `gorm:"size:50;uniqueIndex;not null" json:"username"`
	PasswordHash string     `gorm:"size:255;not null" json:"-"`
	DisplayName  string     `gorm:"size:100;not null" json:"display_name"`
	Role         string     `gorm:"size:20;not null;default:staff" json:"role"` // admin / staff
	Status       string     `gorm:"size:20;not null;default:active" json:"status"` // active / disabled
	LastLoginAt  *time.Time `json:"last_login_at"`
	CreatedBy    *uint      `json:"created_by"`
	CreatedAt    time.Time  `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt    time.Time  `gorm:"autoUpdateTime" json:"updated_at"`

	// 关联
	Shops []Shop `gorm:"many2many:user_shops;" json:"shops,omitempty"`
}

func (User) TableName() string {
	return "users"
}

// IsAdmin 判断是否是管理员
func (u *User) IsAdmin() bool {
	return u.Role == "admin"
}

// IsActive 判断账号是否启用
func (u *User) IsActive() bool {
	return u.Status == "active"
}
