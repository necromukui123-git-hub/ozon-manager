package model

import (
	"time"
)

// 角色常量
const (
	RoleSuperAdmin = "super_admin" // 系统管理员
	RoleShopAdmin  = "shop_admin"  // 店铺管理员
	RoleStaff      = "staff"       // 员工
)

// User 用户表
type User struct {
	ID           uint       `gorm:"primaryKey" json:"id"`
	Username     string     `gorm:"size:50;uniqueIndex;not null" json:"username"`
	PasswordHash string     `gorm:"size:255;not null" json:"-"`
	DisplayName  string     `gorm:"size:100;not null" json:"display_name"`
	Role         string     `gorm:"size:20;not null;default:staff" json:"role"` // super_admin / shop_admin / staff
	Status       string     `gorm:"size:20;not null;default:active" json:"status"` // active / disabled
	LastLoginAt  *time.Time `json:"last_login_at"`
	OwnerID      *uint      `gorm:"index" json:"owner_id"` // 所属店铺管理员ID（仅 staff 有值）
	CreatedBy    *uint      `json:"created_by"`
	CreatedAt    time.Time  `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt    time.Time  `gorm:"autoUpdateTime" json:"updated_at"`

	// 关联
	Shops []Shop `gorm:"many2many:user_shops;" json:"shops,omitempty"`
	Owner *User  `gorm:"foreignKey:OwnerID" json:"owner,omitempty"`  // 所属管理员
	Staff []User `gorm:"foreignKey:OwnerID" json:"staff,omitempty"`  // 下属员工（仅 shop_admin 使用）
}

func (User) TableName() string {
	return "users"
}

// IsSuperAdmin 判断是否是系统管理员
func (u *User) IsSuperAdmin() bool {
	return u.Role == RoleSuperAdmin
}

// IsShopAdmin 判断是否是店铺管理员
func (u *User) IsShopAdmin() bool {
	return u.Role == RoleShopAdmin
}

// IsStaff 判断是否是员工
func (u *User) IsStaff() bool {
	return u.Role == RoleStaff
}

// IsAdmin 判断是否是管理员（兼容旧代码，super_admin 和 shop_admin 都算管理员）
func (u *User) IsAdmin() bool {
	return u.Role == RoleSuperAdmin || u.Role == RoleShopAdmin
}

// CanOperateBusiness 判断是否可以执行业务操作（shop_admin 和 staff）
func (u *User) CanOperateBusiness() bool {
	return u.Role == RoleShopAdmin || u.Role == RoleStaff
}

// IsActive 判断账号是否启用
func (u *User) IsActive() bool {
	return u.Status == "active"
}
