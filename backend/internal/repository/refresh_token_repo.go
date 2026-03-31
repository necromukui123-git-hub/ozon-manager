package repository

import (
	"time"

	"gorm.io/gorm"
	"ozon-manager/internal/model"
)

type RefreshTokenRepository struct {
	db *gorm.DB
}

func NewRefreshTokenRepository(db *gorm.DB) *RefreshTokenRepository {
	return &RefreshTokenRepository{db: db}
}

func (r *RefreshTokenRepository) Create(token *model.UserRefreshToken) error {
	if token.IssuedAt.IsZero() {
		token.IssuedAt = time.Now()
	}
	return r.db.Create(token).Error
}

func (r *RefreshTokenRepository) FindActiveByTokenHash(tokenHash string) (*model.UserRefreshToken, error) {
	var token model.UserRefreshToken
	err := r.db.
		Where("token_hash = ?", tokenHash).
		Where("revoked_at IS NULL").
		Where("expires_at > ?", time.Now()).
		First(&token).Error
	if err != nil {
		return nil, err
	}
	return &token, nil
}

func (r *RefreshTokenRepository) FindByTokenHash(tokenHash string) (*model.UserRefreshToken, error) {
	var token model.UserRefreshToken
	err := r.db.Where("token_hash = ?", tokenHash).First(&token).Error
	if err != nil {
		return nil, err
	}
	return &token, nil
}

func (r *RefreshTokenRepository) RevokeByID(id uint, reason string) error {
	return r.RevokeByIDWithReplacement(id, reason, nil)
}

func (r *RefreshTokenRepository) RevokeByIDWithReplacement(id uint, reason string, replacedByTokenID *uint) error {
	now := time.Now()
	updates := map[string]interface{}{
		"revoked_at":    &now,
		"revoke_reason": reason,
	}
	if replacedByTokenID != nil {
		updates["replaced_by_token_id"] = *replacedByTokenID
	}
	return r.db.Model(&model.UserRefreshToken{}).
		Where("id = ? AND revoked_at IS NULL", id).
		Updates(updates).Error
}

func (r *RefreshTokenRepository) Rotate(previousID uint, reason string, newToken *model.UserRefreshToken) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(newToken).Error; err != nil {
			return err
		}

		now := time.Now()
		result := tx.Model(&model.UserRefreshToken{}).
			Where("id = ? AND revoked_at IS NULL", previousID).
			Updates(map[string]interface{}{
				"revoked_at":           &now,
				"revoke_reason":        reason,
				"replaced_by_token_id": newToken.ID,
				"last_used_at":         &now,
			})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return gorm.ErrRecordNotFound
		}

		return nil
	})
}

func (r *RefreshTokenRepository) RevokeFamily(familyID, reason string) error {
	now := time.Now()
	return r.db.Model(&model.UserRefreshToken{}).
		Where("family_id = ? AND revoked_at IS NULL", familyID).
		Updates(map[string]interface{}{
			"revoked_at":    &now,
			"revoke_reason": reason,
		}).Error
}

func (r *RefreshTokenRepository) RevokeAllByUserID(userID uint, reason string) error {
	now := time.Now()
	return r.db.Model(&model.UserRefreshToken{}).
		Where("user_id = ? AND revoked_at IS NULL", userID).
		Updates(map[string]interface{}{
			"revoked_at":    &now,
			"revoke_reason": reason,
		}).Error
}

func (r *RefreshTokenRepository) TouchLastUsedAt(id uint, usedAt time.Time) error {
	return r.db.Model(&model.UserRefreshToken{}).
		Where("id = ?", id).
		Update("last_used_at", usedAt).Error
}
