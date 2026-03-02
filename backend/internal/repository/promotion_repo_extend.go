package repository

import "ozon-manager/internal/model"

// FindPromotionActionsByIDs 按数据库主键 ID 批量查询活动
func (r *PromotionRepository) FindPromotionActionsByIDs(shopID uint, ids []uint) ([]model.PromotionAction, error) {
	var pas []model.PromotionAction
	err := r.db.Where("shop_id = ? AND id IN ?", shopID, ids).Find(&pas).Error
	return pas, err
}
