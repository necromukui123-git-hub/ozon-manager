package repository

import (
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

// CreateLossProduct 创建亏损商品记录
func (r *PromotionRepository) CreateLossProduct(lp *model.LossProduct) error {
	return r.db.Create(lp).Error
}

// FindLossProductByID 根据ID查找亏损商品
func (r *PromotionRepository) FindLossProductByID(id uint) (*model.LossProduct, error) {
	var lp model.LossProduct
	err := r.db.Preload("Product").First(&lp, id).Error
	if err != nil {
		return nil, err
	}
	return &lp, nil
}

// FindLossProductsByIDs 根据ID列表查找亏损商品
func (r *PromotionRepository) FindLossProductsByIDs(ids []uint) ([]model.LossProduct, error) {
	var lps []model.LossProduct
	err := r.db.Preload("Product").Where("id IN ?", ids).Find(&lps).Error
	return lps, err
}

// FindUnprocessedLossProducts 查找未处理的亏损商品
func (r *PromotionRepository) FindUnprocessedLossProducts(shopID uint) ([]model.LossProduct, error) {
	var lps []model.LossProduct
	err := r.db.Joins("JOIN products ON products.id = loss_products.product_id").
		Where("products.shop_id = ? AND loss_products.processed_at IS NULL", shopID).
		Preload("Product").
		Find(&lps).Error
	return lps, err
}

// UpdateLossProductProcessed 标记亏损商品已处理
func (r *PromotionRepository) UpdateLossProductProcessed(id uint) error {
	now := time.Now()
	return r.db.Model(&model.LossProduct{}).Where("id = ?", id).Updates(map[string]interface{}{
		"price_updated":      true,
		"promotion_exited":   true,
		"promotion_rejoined": true,
		"processed_at":       &now,
	}).Error
}

// UpdateLossProductStep 更新亏损商品处理步骤
func (r *PromotionRepository) UpdateLossProductStep(id uint, field string, value bool) error {
	return r.db.Model(&model.LossProduct{}).Where("id = ?", id).Update(field, value).Error
}

// === PromotedProduct ===

// CreatePromotedProduct 创建已推广商品记录
func (r *PromotionRepository) CreatePromotedProduct(pp *model.PromotedProduct) error {
	return r.db.Create(pp).Error
}

// FindPromotedProductsByProductID 根据商品ID查找推广记录
func (r *PromotionRepository) FindPromotedProductsByProductID(productID uint) ([]model.PromotedProduct, error) {
	var pps []model.PromotedProduct
	err := r.db.Where("product_id = ? AND status = ?", productID, "active").Find(&pps).Error
	return pps, err
}

// FindActivePromotedProducts 查找店铺所有活跃的推广商品
func (r *PromotionRepository) FindActivePromotedProducts(shopID uint) ([]model.PromotedProduct, error) {
	var pps []model.PromotedProduct
	err := r.db.Joins("JOIN products ON products.id = promoted_products.product_id").
		Where("products.shop_id = ? AND promoted_products.status = ?", shopID, "active").
		Preload("Product").
		Find(&pps).Error
	return pps, err
}

// ExitPromotion 退出促销活动
func (r *PromotionRepository) ExitPromotion(productID uint, promotionType string) error {
	now := time.Now()
	return r.db.Model(&model.PromotedProduct{}).
		Where("product_id = ? AND promotion_type = ? AND status = ?", productID, promotionType, "active").
		Updates(map[string]interface{}{
			"status":    "exited",
			"exited_at": &now,
		}).Error
}

// ExitAllPromotions 退出所有促销活动
func (r *PromotionRepository) ExitAllPromotions(productID uint) error {
	now := time.Now()
	return r.db.Model(&model.PromotedProduct{}).
		Where("product_id = ? AND status = ?", productID, "active").
		Updates(map[string]interface{}{
			"status":    "exited",
			"exited_at": &now,
		}).Error
}

// CountByPromotionType 统计某类型促销的商品数量
func (r *PromotionRepository) CountByPromotionType(shopID uint, promotionType string) (int64, error) {
	var count int64
	err := r.db.Model(&model.PromotedProduct{}).
		Joins("JOIN products ON products.id = promoted_products.product_id").
		Where("products.shop_id = ? AND promoted_products.promotion_type = ? AND promoted_products.status = ?",
			shopID, promotionType, "active").
		Count(&count).Error
	return count, err
}

// === PromotionAction ===

// CreatePromotionAction 创建促销活动
func (r *PromotionRepository) CreatePromotionAction(pa *model.PromotionAction) error {
	return r.db.Create(pa).Error
}

// FindPromotionActionByActionID 根据ActionID查找促销活动
func (r *PromotionRepository) FindPromotionActionByActionID(shopID uint, actionID int64) (*model.PromotionAction, error) {
	var pa model.PromotionAction
	err := r.db.Where("shop_id = ? AND action_id = ?", shopID, actionID).First(&pa).Error
	if err != nil {
		return nil, err
	}
	return &pa, nil
}

// FindPromotionActionsByShopID 获取店铺所有促销活动
func (r *PromotionRepository) FindPromotionActionsByShopID(shopID uint) ([]model.PromotionAction, error) {
	var pas []model.PromotionAction
	err := r.db.Where("shop_id = ?", shopID).Find(&pas).Error
	return pas, err
}

// FindElasticBoostAction 获取弹性提升活动
func (r *PromotionRepository) FindElasticBoostAction(shopID uint) (*model.PromotionAction, error) {
	var pa model.PromotionAction
	err := r.db.Where("shop_id = ? AND is_elastic_boost = ?", shopID, true).First(&pa).Error
	if err != nil {
		return nil, err
	}
	return &pa, nil
}

// FindDiscount28Action 获取28%折扣活动
func (r *PromotionRepository) FindDiscount28Action(shopID uint) (*model.PromotionAction, error) {
	var pa model.PromotionAction
	err := r.db.Where("shop_id = ? AND is_discount_28 = ?", shopID, true).First(&pa).Error
	if err != nil {
		return nil, err
	}
	return &pa, nil
}

// UpsertPromotionAction 创建或更新促销活动
func (r *PromotionRepository) UpsertPromotionAction(pa *model.PromotionAction) error {
	return r.db.Where("shop_id = ? AND action_id = ?", pa.ShopID, pa.ActionID).
		Assign(pa).
		FirstOrCreate(pa).Error
}

// FindPromotionActionByID 根据数据库ID查找促销活动
func (r *PromotionRepository) FindPromotionActionByID(id uint) (*model.PromotionAction, error) {
	var pa model.PromotionAction
	err := r.db.First(&pa, id).Error
	if err != nil {
		return nil, err
	}
	return &pa, nil
}

// DeletePromotionAction 删除促销活动
func (r *PromotionRepository) DeletePromotionAction(id uint) error {
	return r.db.Delete(&model.PromotionAction{}, id).Error
}

// FindPromotionActionsByActionIDs 根据ActionID列表查找促销活动
func (r *PromotionRepository) FindPromotionActionsByActionIDs(shopID uint, actionIDs []int64) ([]model.PromotionAction, error) {
	var pas []model.PromotionAction
	err := r.db.Where("shop_id = ? AND action_id IN ?", shopID, actionIDs).Find(&pas).Error
	return pas, err
}

// FindActivePromotionActions 查找活跃的促销活动
func (r *PromotionRepository) FindActivePromotionActions(shopID uint) ([]model.PromotionAction, error) {
	var pas []model.PromotionAction
	err := r.db.Where("shop_id = ? AND status = ?", shopID, "active").Order("created_at DESC").Find(&pas).Error
	return pas, err
}

// UpdatePromotionActionStatus 更新促销活动状态
func (r *PromotionRepository) UpdatePromotionActionStatus(id uint, status string) error {
	return r.db.Model(&model.PromotionAction{}).Where("id = ?", id).Update("status", status).Error
}
