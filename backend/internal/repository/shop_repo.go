package repository

import (
	"gorm.io/gorm"
	"ozon-manager/internal/model"
)

type ShopRepository struct {
	db *gorm.DB
}

func NewShopRepository(db *gorm.DB) *ShopRepository {
	return &ShopRepository{db: db}
}

// FindByID 根据ID查找店铺
func (r *ShopRepository) FindByID(id uint) (*model.Shop, error) {
	var shop model.Shop
	err := r.db.First(&shop, id).Error
	if err != nil {
		return nil, err
	}
	return &shop, nil
}

// FindByClientID 根据ClientID查找店铺
func (r *ShopRepository) FindByClientID(clientID string) (*model.Shop, error) {
	var shop model.Shop
	err := r.db.Where("client_id = ?", clientID).First(&shop).Error
	if err != nil {
		return nil, err
	}
	return &shop, nil
}

// FindAll 获取所有店铺
func (r *ShopRepository) FindAll() ([]model.Shop, error) {
	var shops []model.Shop
	err := r.db.Find(&shops).Error
	return shops, err
}

// FindActive 获取所有启用的店铺
func (r *ShopRepository) FindActive() ([]model.Shop, error) {
	var shops []model.Shop
	err := r.db.Where("is_active = ?", true).Find(&shops).Error
	return shops, err
}

// FindByIDs 根据ID列表查找店铺
func (r *ShopRepository) FindByIDs(ids []uint) ([]model.Shop, error) {
	var shops []model.Shop
	err := r.db.Where("id IN ?", ids).Find(&shops).Error
	return shops, err
}

// FindByUserID 获取用户可访问的店铺
func (r *ShopRepository) FindByUserID(userID uint) ([]model.Shop, error) {
	var shops []model.Shop
	err := r.db.Joins("JOIN user_shops ON user_shops.shop_id = shops.id").
		Where("user_shops.user_id = ?", userID).
		Find(&shops).Error
	return shops, err
}

// Create 创建店铺
func (r *ShopRepository) Create(shop *model.Shop) error {
	return r.db.Create(shop).Error
}

// Update 更新店铺
func (r *ShopRepository) Update(shop *model.Shop) error {
	return r.db.Save(shop).Error
}

// Delete 删除店铺
func (r *ShopRepository) Delete(id uint) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		// 删除用户-店铺关联
		if err := tx.Where("shop_id = ?", id).Delete(&model.UserShop{}).Error; err != nil {
			return err
		}
		// 删除店铺
		return tx.Delete(&model.Shop{}, id).Error
	})
}

// GetWithCredentials 获取店铺（包含API凭证）
func (r *ShopRepository) GetWithCredentials(id uint) (*model.Shop, error) {
	var shop model.Shop
	err := r.db.First(&shop, id).Error
	if err != nil {
		return nil, err
	}
	return &shop, nil
}
