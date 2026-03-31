package model

import "time"

type UserRefreshToken struct {
	ID                uint       `gorm:"primaryKey" json:"id"`
	UserID            uint       `gorm:"not null;index" json:"user_id"`
	TokenHash         string     `gorm:"size:128;not null;uniqueIndex" json:"token_hash"`
	FamilyID          string     `gorm:"size:64;not null;index" json:"family_id"`
	UserAgent         string     `gorm:"type:text" json:"user_agent"`
	IPAddress         string     `gorm:"size:64" json:"ip_address"`
	IssuedAt          time.Time  `gorm:"not null" json:"issued_at"`
	ExpiresAt         time.Time  `gorm:"not null;index" json:"expires_at"`
	LastUsedAt        *time.Time `json:"last_used_at"`
	RevokedAt         *time.Time `gorm:"index" json:"revoked_at"`
	RevokeReason      string     `gorm:"size:100" json:"revoke_reason"`
	ReplacedByTokenID *uint      `json:"replaced_by_token_id"`
	CreatedAt         time.Time  `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt         time.Time  `gorm:"autoUpdateTime" json:"updated_at"`

	User *User `gorm:"foreignKey:UserID" json:"user,omitempty"`
}

func (UserRefreshToken) TableName() string {
	return "user_refresh_tokens"
}
