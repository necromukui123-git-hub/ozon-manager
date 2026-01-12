package model

import (
	"time"
)

// Shop 店铺表
type Shop struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Name      string    `gorm:"size:100;not null" json:"name"`
	ClientID  string    `gorm:"size:50;uniqueIndex;not null" json:"client_id"`
	ApiKey    string    `gorm:"size:200;not null" json:"-"` // 不返回给前端
	IsActive  bool      `gorm:"default:true" json:"is_active"`
	OwnerID   uint      `gorm:"not null;index" json:"owner_id"` // 店铺所属的店铺管理员ID
	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updated_at"`

	// 关联
	Owner *User  `gorm:"foreignKey:OwnerID" json:"owner,omitempty"` // 所属管理员
	Users []User `gorm:"many2many:user_shops;" json:"users,omitempty"`
}

func (Shop) TableName() string {
	return "shops"
}

// UserShop 用户-店铺关联表
type UserShop struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	UserID    uint      `gorm:"not null;uniqueIndex:idx_user_shop" json:"user_id"`
	ShopID    uint      `gorm:"not null;uniqueIndex:idx_user_shop" json:"shop_id"`
	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`
}

func (UserShop) TableName() string {
	return "user_shops"
}
