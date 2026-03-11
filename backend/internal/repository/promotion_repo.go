package repository

import (
	"fmt"
	"strconv"
	"time"

	"gorm.io/gorm"
	"ozon-manager/internal/model"
)

type PromotionRepository struct {
	db *gorm.DB
}

func NewPromotionRepository(db *gorm.DB) *PromotionRepository {
	return &PromotionRepository{db: db}
}

// === LossProduct ===

func (r *PromotionRepository) CreateLossProduct(lp *model.LossProduct) error {
	return r.db.Create(lp).Error
}

func (r *PromotionRepository) FindLossProductByID(id uint) (*model.LossProduct, error) {
	var lp model.LossProduct
	err := r.db.Preload("Product").First(&lp, id).Error
	if err != nil {
		return nil, err
	}
	return &lp, nil
}

func (r *PromotionRepository) FindLossProductsByIDs(ids []uint) ([]model.LossProduct, error) {
	var lps []model.LossProduct
	err := r.db.Preload("Product").Where("id IN ?", ids).Find(&lps).Error
	return lps, err
}

func (r *PromotionRepository) FindUnprocessedLossProducts(shopID uint) ([]model.LossProduct, error) {
	var lps []model.LossProduct
	err := r.db.Joins("JOIN products ON products.id = loss_products.product_id").
		Where("products.shop_id = ? AND loss_products.processed_at IS NULL", shopID).
		Preload("Product").
		Find(&lps).Error
	return lps, err
}

func (r *PromotionRepository) UpdateLossProductProcessed(id uint) error {
	now := time.Now()
	return r.db.Model(&model.LossProduct{}).Where("id = ?", id).Updates(map[string]interface{}{
		"price_updated":      true,
		"promotion_exited":   true,
		"promotion_rejoined": true,
		"processed_at":       &now,
	}).Error
}

func (r *PromotionRepository) UpdateLossProductStep(id uint, field string, value bool) error {
	return r.db.Model(&model.LossProduct{}).Where("id = ?", id).Update(field, value).Error
}

// === PromotedProduct ===

func (r *PromotionRepository) CreatePromotedProduct(pp *model.PromotedProduct) error {
	return r.db.Create(pp).Error
}

func (r *PromotionRepository) FindPromotedProductsByProductID(productID uint) ([]model.PromotedProduct, error) {
	var pps []model.PromotedProduct
	err := r.db.Where("product_id = ? AND status = ?", productID, "active").Find(&pps).Error
	return pps, err
}

func (r *PromotionRepository) FindActivePromotedProducts(shopID uint) ([]model.PromotedProduct, error) {
	var pps []model.PromotedProduct
	err := r.db.Joins("JOIN products ON products.id = promoted_products.product_id").
		Where("products.shop_id = ? AND promoted_products.status = ?", shopID, "active").
		Preload("Product").
		Find(&pps).Error
	return pps, err
}

func (r *PromotionRepository) ExitPromotion(productID uint, promotionType string) error {
	now := time.Now()
	return r.db.Model(&model.PromotedProduct{}).
		Where("product_id = ? AND promotion_type = ? AND status = ?", productID, promotionType, "active").
		Updates(map[string]interface{}{
			"status":    "exited",
			"exited_at": &now,
		}).Error
}

func (r *PromotionRepository) ExitAllPromotions(productID uint) error {
	now := time.Now()
	return r.db.Model(&model.PromotedProduct{}).
		Where("product_id = ? AND status = ?", productID, "active").
		Updates(map[string]interface{}{
			"status":    "exited",
			"exited_at": &now,
		}).Error
}

func (r *PromotionRepository) CountByPromotionType(shopID uint, promotionType string) (int64, error) {
	var count int64
	err := r.db.Model(&model.PromotedProduct{}).
		Joins("JOIN products ON products.id = promoted_products.product_id").
		Where("products.shop_id = ? AND promoted_products.promotion_type = ? AND promoted_products.status = ?", shopID, promotionType, "active").
		Count(&count).Error
	return count, err
}

// === PromotionAction ===

func (r *PromotionRepository) CreatePromotionAction(pa *model.PromotionAction) error {
	if pa.Source == "" {
		pa.Source = "official"
	}
	if pa.SourceActionID == "" {
		pa.SourceActionID = strconv.FormatInt(pa.ActionID, 10)
	}
	return r.db.Create(pa).Error
}

func (r *PromotionRepository) FindPromotionActionByActionID(shopID uint, actionID int64) (*model.PromotionAction, error) {
	var pa model.PromotionAction
	err := r.db.Where("shop_id = ? AND source = ? AND action_id = ?", shopID, "official", actionID).First(&pa).Error
	if err == gorm.ErrRecordNotFound {
		err = r.db.Where("shop_id = ? AND action_id = ?", shopID, actionID).First(&pa).Error
	}
	if err != nil {
		return nil, err
	}
	return &pa, nil
}

func (r *PromotionRepository) FindPromotionActionsByShopID(shopID uint) ([]model.PromotionAction, error) {
	var pas []model.PromotionAction
	err := r.db.Where("shop_id = ?", shopID).Order("sort_order ASC, id ASC").Find(&pas).Error
	return pas, err
}

func (r *PromotionRepository) UpsertPromotionAction(pa *model.PromotionAction) error {
	if pa.Source == "" {
		pa.Source = "official"
	}
	if pa.SourceActionID == "" {
		pa.SourceActionID = strconv.FormatInt(pa.ActionID, 10)
	}

	var existing model.PromotionAction
	err := r.db.Where("shop_id = ? AND source = ? AND source_action_id = ?", pa.ShopID, pa.Source, pa.SourceActionID).First(&existing).Error

	if err == gorm.ErrRecordNotFound {
		var maxSortOrder int
		r.db.Model(&model.PromotionAction{}).Where("shop_id = ?", pa.ShopID).Select("COALESCE(MAX(sort_order), -1)").Scan(&maxSortOrder)
		pa.SortOrder = maxSortOrder + 1
		return r.db.Create(pa).Error
	}
	if err != nil {
		return err
	}

	pa.ID = existing.ID
	pa.DisplayName = existing.DisplayName
	pa.SortOrder = existing.SortOrder
	return r.db.Save(pa).Error
}

func (r *PromotionRepository) FindPromotionActionByID(id uint) (*model.PromotionAction, error) {
	var pa model.PromotionAction
	err := r.db.First(&pa, id).Error
	if err != nil {
		return nil, err
	}
	return &pa, nil
}

func (r *PromotionRepository) FindPromotionActionByIDAndShop(id uint, shopID uint) (*model.PromotionAction, error) {
	var pa model.PromotionAction
	err := r.db.Where("id = ? AND shop_id = ?", id, shopID).First(&pa).Error
	if err != nil {
		return nil, err
	}
	return &pa, nil
}

func (r *PromotionRepository) DeletePromotionAction(id uint) error {
	return r.db.Delete(&model.PromotionAction{}, id).Error
}

func (r *PromotionRepository) FindPromotionActionsByActionIDs(shopID uint, actionIDs []int64) ([]model.PromotionAction, error) {
	var pas []model.PromotionAction
	err := r.db.Where("shop_id = ? AND source = ? AND action_id IN ?", shopID, "official", actionIDs).Find(&pas).Error
	return pas, err
}

func (r *PromotionRepository) FindActivePromotionActions(shopID uint) ([]model.PromotionAction, error) {
	var pas []model.PromotionAction
	err := r.db.Where("shop_id = ? AND status = ?", shopID, "active").Order("sort_order ASC, id ASC").Find(&pas).Error
	return pas, err
}

func (r *PromotionRepository) UpdatePromotionActionStatus(id uint, status string) error {
	return r.db.Model(&model.PromotionAction{}).Where("id = ?", id).Update("status", status).Error
}

func (r *PromotionRepository) UpdatePromotionActionDisplayName(id uint, displayName string) error {
	return r.db.Model(&model.PromotionAction{}).Where("id = ?", id).Update("display_name", displayName).Error
}

func (r *PromotionRepository) UpdatePromotionActionsSortOrder(shopID uint, sortOrders map[uint]int) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		for id, sortOrder := range sortOrders {
			if err := tx.Model(&model.PromotionAction{}).
				Where("id = ? AND shop_id = ?", id, shopID).
				Update("sort_order", sortOrder).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (r *PromotionRepository) ReplaceActionProducts(action *model.PromotionAction, products []model.PromotionActionProduct) error {
	now := time.Now()
	for index := range products {
		products[index].PromotionActionID = action.ID
		products[index].ShopID = action.ShopID
		products[index].LastSyncedAt = &now
	}

	return r.db.Transaction(func(tx *gorm.DB) error {
		for index := range products {
			product := products[index]
			var existing model.PromotionActionProduct
			err := tx.Where("promotion_action_id = ? AND source_sku = ?", action.ID, product.SourceSKU).First(&existing).Error
			if err == gorm.ErrRecordNotFound {
				if createErr := tx.Create(&product).Error; createErr != nil {
					return createErr
				}
				continue
			}
			if err != nil {
				return err
			}

			product.ID = existing.ID
			if saveErr := tx.Save(&product).Error; saveErr != nil {
				return saveErr
			}
		}

		sourceSKUs := make([]string, 0, len(products))
		for _, product := range products {
			sourceSKUs = append(sourceSKUs, product.SourceSKU)
		}

		if len(sourceSKUs) > 0 {
			if err := tx.Where("promotion_action_id = ? AND source_sku NOT IN ?", action.ID, sourceSKUs).Delete(&model.PromotionActionProduct{}).Error; err != nil {
				return err
			}
		} else {
			if err := tx.Where("promotion_action_id = ?", action.ID).Delete(&model.PromotionActionProduct{}).Error; err != nil {
				return err
			}
		}

		if err := tx.Model(&model.PromotionAction{}).Where("id = ?", action.ID).Updates(map[string]interface{}{
			"last_products_synced_at": &now,
			"updated_at":              &now,
		}).Error; err != nil {
			return fmt.Errorf("failed to update action sync time: %w", err)
		}

		return nil
	})
}

func (r *PromotionRepository) ListActionProducts(shopID uint, promotionActionID uint, page int, pageSize int, keyword string, status string) ([]model.PromotionActionProduct, int64, error) {
	var items []model.PromotionActionProduct
	var total int64

	query := r.db.Model(&model.PromotionActionProduct{}).Where("shop_id = ? AND promotion_action_id = ?", shopID, promotionActionID)
	if keyword != "" {
		pattern := "%" + keyword + "%"
		query = query.Where("source_sku ILIKE ? OR offer_id ILIKE ? OR platform_sku ILIKE ? OR name_cn ILIKE ? OR name_origin ILIKE ? OR name ILIKE ?",
			pattern, pattern, pattern, pattern, pattern, pattern)
	}
	if status != "" && status != "all" {
		query = query.Where("status = ?", status)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	err := query.Order("discount_percent DESC, id ASC").Offset(offset).Limit(pageSize).Find(&items).Error
	if err != nil {
		return nil, 0, err
	}

	return items, total, nil
}

func (r *PromotionRepository) ReplaceActionCandidates(action *model.PromotionAction, candidates []model.PromotionActionCandidate) error {
	now := time.Now()
	for index := range candidates {
		candidates[index].PromotionActionID = action.ID
		candidates[index].ShopID = action.ShopID
		candidates[index].LastSyncedAt = &now
	}

	return r.db.Transaction(func(tx *gorm.DB) error {
		for index := range candidates {
			candidate := candidates[index]
			var existing model.PromotionActionCandidate
			err := tx.Where("promotion_action_id = ? AND source_sku = ?", action.ID, candidate.SourceSKU).First(&existing).Error
			if err == gorm.ErrRecordNotFound {
				if createErr := tx.Create(&candidate).Error; createErr != nil {
					return createErr
				}
				continue
			}
			if err != nil {
				return err
			}

			candidate.ID = existing.ID
			if saveErr := tx.Save(&candidate).Error; saveErr != nil {
				return saveErr
			}
		}

		sourceSKUs := make([]string, 0, len(candidates))
		for _, candidate := range candidates {
			sourceSKUs = append(sourceSKUs, candidate.SourceSKU)
		}

		if len(sourceSKUs) > 0 {
			if err := tx.Where("promotion_action_id = ? AND source_sku NOT IN ?", action.ID, sourceSKUs).Delete(&model.PromotionActionCandidate{}).Error; err != nil {
				return err
			}
		} else {
			if err := tx.Where("promotion_action_id = ?", action.ID).Delete(&model.PromotionActionCandidate{}).Error; err != nil {
				return err
			}
		}

		return nil
	})
}

func (r *PromotionRepository) ListActionCandidatesByActionIDsAndSourceSKUs(shopID uint, actionIDs []uint, sourceSKUs []string) ([]model.PromotionActionCandidate, error) {
	items := make([]model.PromotionActionCandidate, 0)
	if len(actionIDs) == 0 || len(sourceSKUs) == 0 {
		return items, nil
	}

	err := r.db.Where("shop_id = ? AND promotion_action_id IN ? AND source_sku IN ?", shopID, actionIDs, sourceSKUs).
		Find(&items).Error
	return items, err
}

func (r *PromotionRepository) ListActionProductsByActionIDsAndSourceSKUs(shopID uint, actionIDs []uint, sourceSKUs []string) ([]model.PromotionActionProduct, error) {
	items := make([]model.PromotionActionProduct, 0)
	if len(actionIDs) == 0 || len(sourceSKUs) == 0 {
		return items, nil
	}

	err := r.db.Where("shop_id = ? AND promotion_action_id IN ? AND source_sku IN ?", shopID, actionIDs, sourceSKUs).
		Find(&items).Error
	return items, err
}
