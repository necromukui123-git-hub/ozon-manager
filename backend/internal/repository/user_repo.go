package repository

import (
	"gorm.io/gorm"
	"ozon-manager/internal/model"
)

type UserRepository struct {
	db *gorm.DB
}

func NewUserRepository(db *gorm.DB) *UserRepository {
	return &UserRepository{db: db}
}

// FindByID 根据ID查找用户
func (r *UserRepository) FindByID(id uint) (*model.User, error) {
	var user model.User
	err := r.db.Preload("Shops").First(&user, id).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// FindByUsername 根据用户名查找用户
func (r *UserRepository) FindByUsername(username string) (*model.User, error) {
	var user model.User
	err := r.db.Preload("Shops").Where("username = ?", username).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// FindAll 获取所有用户
func (r *UserRepository) FindAll() ([]model.User, error) {
	var users []model.User
	err := r.db.Preload("Shops").Find(&users).Error
	return users, err
}

// FindStaff 获取所有员工（非管理员）
func (r *UserRepository) FindStaff() ([]model.User, error) {
	var users []model.User
	err := r.db.Preload("Shops").Where("role = ?", "staff").Find(&users).Error
	return users, err
}

// Create 创建用户
func (r *UserRepository) Create(user *model.User) error {
	return r.db.Create(user).Error
}

// Update 更新用户
func (r *UserRepository) Update(user *model.User) error {
	return r.db.Save(user).Error
}

// UpdateStatus 更新用户状态
func (r *UserRepository) UpdateStatus(id uint, status string) error {
	return r.db.Model(&model.User{}).Where("id = ?", id).Update("status", status).Error
}

// UpdatePassword 更新用户密码
func (r *UserRepository) UpdatePassword(id uint, passwordHash string) error {
	return r.db.Model(&model.User{}).Where("id = ?", id).Update("password_hash", passwordHash).Error
}

// UpdateLastLogin 更新最后登录时间
func (r *UserRepository) UpdateLastLogin(id uint) error {
	return r.db.Model(&model.User{}).Where("id = ?", id).Update("last_login_at", gorm.Expr("NOW()")).Error
}

// UpdateShops 更新用户可访问的店铺
func (r *UserRepository) UpdateShops(userID uint, shopIDs []uint) error {
	// 先删除现有关联
	if err := r.db.Where("user_id = ?", userID).Delete(&model.UserShop{}).Error; err != nil {
		return err
	}

	// 添加新的关联
	for _, shopID := range shopIDs {
		userShop := model.UserShop{
			UserID: userID,
			ShopID: shopID,
		}
		if err := r.db.Create(&userShop).Error; err != nil {
			return err
		}
	}

	return nil
}

// GetUserShopIDs 获取用户可访问的店铺ID列表
func (r *UserRepository) GetUserShopIDs(userID uint) ([]uint, error) {
	var shopIDs []uint
	err := r.db.Model(&model.UserShop{}).Where("user_id = ?", userID).Pluck("shop_id", &shopIDs).Error
	return shopIDs, err
}

// Delete 删除用户
func (r *UserRepository) Delete(id uint) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		// 删除用户-店铺关联
		if err := tx.Where("user_id = ?", id).Delete(&model.UserShop{}).Error; err != nil {
			return err
		}
		// 删除用户
		return tx.Delete(&model.User{}, id).Error
	})
}

// FindShopAdmins 获取所有店铺管理员
func (r *UserRepository) FindShopAdmins() ([]model.User, error) {
	var users []model.User
	err := r.db.Where("role = ?", model.RoleShopAdmin).Find(&users).Error
	return users, err
}

// FindByOwnerID 获取某个店铺管理员下的所有员工
func (r *UserRepository) FindByOwnerID(ownerID uint) ([]model.User, error) {
	var users []model.User
	err := r.db.Preload("Shops").Where("owner_id = ?", ownerID).Find(&users).Error
	return users, err
}

// FindByRole 按角色查找用户
func (r *UserRepository) FindByRole(role string) ([]model.User, error) {
	var users []model.User
	err := r.db.Preload("Shops").Where("role = ?", role).Find(&users).Error
	return users, err
}

// FindShopAdminWithDetails 获取店铺管理员详情（包含其店铺和员工）
func (r *UserRepository) FindShopAdminWithDetails(id uint) (*model.User, error) {
	var user model.User
	err := r.db.Preload("Staff").Preload("Staff.Shops").First(&user, id).Error
	if err != nil {
		return nil, err
	}
	if user.Role != model.RoleShopAdmin {
		return nil, gorm.ErrRecordNotFound
	}
	return &user, nil
}

// CountByRole 统计各角色用户数量
func (r *UserRepository) CountByRole(role string) (int64, error) {
	var count int64
	err := r.db.Model(&model.User{}).Where("role = ?", role).Count(&count).Error
	return count, err
}

// CountByOwnerID 统计某个店铺管理员下的员工数量
func (r *UserRepository) CountByOwnerID(ownerID uint) (int64, error) {
	var count int64
	err := r.db.Model(&model.User{}).Where("owner_id = ?", ownerID).Count(&count).Error
	return count, err
}
