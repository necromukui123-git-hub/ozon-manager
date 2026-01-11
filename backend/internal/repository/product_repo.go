package repository

import (
	"gorm.io/gorm"
	"ozon-manager/internal/model"
)

type ProductRepository struct {
	db *gorm.DB
}

func NewProductRepository(db *gorm.DB) *ProductRepository {
	return &ProductRepository{db: db}
}

// FindByID 根据ID查找商品
func (r *ProductRepository) FindByID(id uint) (*model.Product, error) {
	var product model.Product
	err := r.db.Preload("PromotedProducts").First(&product, id).Error
	if err != nil {
		return nil, err
	}
	return &product, nil
}

// FindBySourceSKU 根据SourceSKU查找商品
func (r *ProductRepository) FindBySourceSKU(shopID uint, sourceSKU string) (*model.Product, error) {
	var product model.Product
	err := r.db.Where("shop_id = ? AND source_sku = ?", shopID, sourceSKU).First(&product).Error
	if err != nil {
		return nil, err
	}
	return &product, nil
}

// FindByOzonProductID 根据Ozon产品ID查找商品
func (r *ProductRepository) FindByOzonProductID(shopID uint, ozonProductID int64) (*model.Product, error) {
	var product model.Product
	err := r.db.Where("shop_id = ? AND ozon_product_id = ?", shopID, ozonProductID).First(&product).Error
	if err != nil {
		return nil, err
	}
	return &product, nil
}

// FindByShopID 获取店铺的所有商品
func (r *ProductRepository) FindByShopID(shopID uint) ([]model.Product, error) {
	var products []model.Product
	err := r.db.Where("shop_id = ?", shopID).Find(&products).Error
	return products, err
}

// FindWithFilters 带筛选条件的商品列表
func (r *ProductRepository) FindWithFilters(shopID uint, isLoss *bool, isPromoted *bool, keyword string, page, pageSize int) ([]model.Product, int64, error) {
	var products []model.Product
	var total int64

	query := r.db.Model(&model.Product{}).Where("shop_id = ?", shopID)

	if isLoss != nil {
		query = query.Where("is_loss = ?", *isLoss)
	}

	if isPromoted != nil {
		query = query.Where("is_promoted = ?", *isPromoted)
	}

	if keyword != "" {
		query = query.Where("name ILIKE ? OR source_sku ILIKE ?", "%"+keyword+"%", "%"+keyword+"%")
	}

	// 获取总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 分页查询
	offset := (page - 1) * pageSize
	err := query.Preload("PromotedProducts").
		Offset(offset).
		Limit(pageSize).
		Order("created_at DESC").
		Find(&products).Error

	return products, total, err
}

// FindEligible 查找符合促销条件的商品（排除亏损和已推广）
func (r *ProductRepository) FindEligible(shopID uint, excludeLoss, excludePromoted bool) ([]model.Product, error) {
	var products []model.Product
	query := r.db.Where("shop_id = ? AND status = ?", shopID, "active")

	if excludeLoss {
		query = query.Where("is_loss = ?", false)
	}

	if excludePromoted {
		query = query.Where("is_promoted = ?", false)
	}

	err := query.Find(&products).Error
	return products, err
}

// FindPromotable 查找可推广的商品
func (r *ProductRepository) FindPromotable(shopID uint) ([]model.Product, error) {
	return r.FindEligible(shopID, true, true)
}

// Create 创建商品
func (r *ProductRepository) Create(product *model.Product) error {
	return r.db.Create(product).Error
}

// CreateBatch 批量创建商品
func (r *ProductRepository) CreateBatch(products []model.Product) error {
	return r.db.CreateInBatches(products, 100).Error
}

// Update 更新商品
func (r *ProductRepository) Update(product *model.Product) error {
	return r.db.Save(product).Error
}

// UpdatePrice 更新商品价格
func (r *ProductRepository) UpdatePrice(id uint, price float64) error {
	return r.db.Model(&model.Product{}).Where("id = ?", id).Update("current_price", price).Error
}

// UpdateLossStatus 更新亏损状态
func (r *ProductRepository) UpdateLossStatus(id uint, isLoss bool) error {
	return r.db.Model(&model.Product{}).Where("id = ?", id).Update("is_loss", isLoss).Error
}

// UpdatePromotedStatus 更新推广状态
func (r *ProductRepository) UpdatePromotedStatus(id uint, isPromoted bool) error {
	return r.db.Model(&model.Product{}).Where("id = ?", id).Update("is_promoted", isPromoted).Error
}

// Upsert 创建或更新商品
func (r *ProductRepository) Upsert(product *model.Product) error {
	return r.db.Where("shop_id = ? AND ozon_product_id = ?", product.ShopID, product.OzonProductID).
		Assign(product).
		FirstOrCreate(product).Error
}

// CountByShopID 统计店铺商品数量
func (r *ProductRepository) CountByShopID(shopID uint) (int64, error) {
	var count int64
	err := r.db.Model(&model.Product{}).Where("shop_id = ?", shopID).Count(&count).Error
	return count, err
}

// CountLossByShopID 统计店铺亏损商品数量
func (r *ProductRepository) CountLossByShopID(shopID uint) (int64, error) {
	var count int64
	err := r.db.Model(&model.Product{}).Where("shop_id = ? AND is_loss = ?", shopID, true).Count(&count).Error
	return count, err
}

// CountPromotedByShopID 统计店铺已推广商品数量
func (r *ProductRepository) CountPromotedByShopID(shopID uint) (int64, error) {
	var count int64
	err := r.db.Model(&model.Product{}).Where("shop_id = ? AND is_promoted = ?", shopID, true).Count(&count).Error
	return count, err
}
