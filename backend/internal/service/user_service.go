package service

import (
	"errors"

	"ozon-manager/internal/dto"
	"ozon-manager/internal/model"
	"ozon-manager/internal/repository"
)

var (
	ErrUsernameExists         = errors.New("用户名已存在")
	ErrCannotModifyAdmin      = errors.New("不能修改管理员账号")
	ErrWrongPassword          = errors.New("原密码错误")
	ErrNotShopAdmin           = errors.New("不是店铺管理员")
	ErrStaffNotBelongToYou    = errors.New("该员工不属于您")
	ErrCannotModifySuperAdmin = errors.New("不能修改系统管理员账号")
)

type UserService struct {
	userRepo *repository.UserRepository
	shopRepo *repository.ShopRepository
}

func NewUserService(userRepo *repository.UserRepository, shopRepo *repository.ShopRepository) *UserService {
	return &UserService{
		userRepo: userRepo,
		shopRepo: shopRepo,
	}
}

// GetAllUsers 获取所有用户（员工）
func (s *UserService) GetAllUsers() ([]dto.UserInfo, error) {
	users, err := s.userRepo.FindStaff()
	if err != nil {
		return nil, err
	}

	result := make([]dto.UserInfo, 0, len(users))
	for _, user := range users {
		shops := make([]dto.ShopInfo, 0)
		for _, shop := range user.Shops {
			shops = append(shops, dto.ShopInfo{
				ID:   shop.ID,
				Name: shop.Name,
			})
		}
		result = append(result, dto.UserInfo{
			ID:          user.ID,
			Username:    user.Username,
			DisplayName: user.DisplayName,
			Role:        user.Role,
			Shops:       shops,
		})
	}

	return result, nil
}

// CreateUser 创建用户
func (s *UserService) CreateUser(req *dto.CreateUserRequest, createdBy uint) (*dto.UserInfo, error) {
	// 检查用户名是否已存在
	existing, _ := s.userRepo.FindByUsername(req.Username)
	if existing != nil {
		return nil, ErrUsernameExists
	}

	// 生成密码哈希
	passwordHash, err := HashPassword(req.Password)
	if err != nil {
		return nil, err
	}

	// 创建用户
	user := &model.User{
		Username:     req.Username,
		PasswordHash: passwordHash,
		DisplayName:  req.DisplayName,
		Role:         "staff",
		Status:       "active",
		CreatedBy:    &createdBy,
	}

	if err := s.userRepo.Create(user); err != nil {
		return nil, err
	}

	// 分配店铺
	if len(req.ShopIDs) > 0 {
		if err := s.userRepo.UpdateShops(user.ID, req.ShopIDs); err != nil {
			return nil, err
		}
	}

	// 获取完整用户信息
	user, _ = s.userRepo.FindByID(user.ID)

	shops := make([]dto.ShopInfo, 0)
	for _, shop := range user.Shops {
		shops = append(shops, dto.ShopInfo{
			ID:   shop.ID,
			Name: shop.Name,
		})
	}

	return &dto.UserInfo{
		ID:          user.ID,
		Username:    user.Username,
		DisplayName: user.DisplayName,
		Role:        user.Role,
		Shops:       shops,
	}, nil
}

// UpdateUserStatus 更新用户状态
func (s *UserService) UpdateUserStatus(userID uint, status string) error {
	user, err := s.userRepo.FindByID(userID)
	if err != nil {
		return ErrUserNotFound
	}

	// 不能修改管理员账号
	if user.IsAdmin() {
		return ErrCannotModifyAdmin
	}

	return s.userRepo.UpdateStatus(userID, status)
}

// UpdateUserPassword 重置用户密码
func (s *UserService) UpdateUserPassword(userID uint, newPassword string) error {
	user, err := s.userRepo.FindByID(userID)
	if err != nil {
		return ErrUserNotFound
	}

	// 不能修改管理员账号
	if user.IsAdmin() {
		return ErrCannotModifyAdmin
	}

	passwordHash, err := HashPassword(newPassword)
	if err != nil {
		return err
	}

	return s.userRepo.UpdatePassword(userID, passwordHash)
}

// UpdateUserShops 更新用户可访问的店铺
func (s *UserService) UpdateUserShops(userID uint, shopIDs []uint) error {
	user, err := s.userRepo.FindByID(userID)
	if err != nil {
		return ErrUserNotFound
	}

	// 不能修改管理员账号
	if user.IsAdmin() {
		return ErrCannotModifyAdmin
	}

	return s.userRepo.UpdateShops(userID, shopIDs)
}

// GetUserByID 获取用户信息
func (s *UserService) GetUserByID(userID uint) (*dto.UserInfo, error) {
	user, err := s.userRepo.FindByID(userID)
	if err != nil {
		return nil, ErrUserNotFound
	}

	shops := make([]dto.ShopInfo, 0)
	for _, shop := range user.Shops {
		shops = append(shops, dto.ShopInfo{
			ID:   shop.ID,
			Name: shop.Name,
		})
	}

	return &dto.UserInfo{
		ID:          user.ID,
		Username:    user.Username,
		DisplayName: user.DisplayName,
		Role:        user.Role,
		Shops:       shops,
	}, nil
}

// CanAccessShop 检查用户是否可以访问店铺
func (s *UserService) CanAccessShop(userID, shopID uint) (bool, error) {
	user, err := s.userRepo.FindByID(userID)
	if err != nil {
		return false, err
	}

	// 管理员可以访问所有店铺
	if user.IsAdmin() {
		return true, nil
	}

	// 检查员工是否有权限
	shopIDs, err := s.userRepo.GetUserShopIDs(userID)
	if err != nil {
		return false, err
	}

	for _, id := range shopIDs {
		if id == shopID {
			return true, nil
		}
	}

	return false, nil
}

// ChangePassword 用户修改自己的密码
func (s *UserService) ChangePassword(userID uint, oldPassword, newPassword string) error {
	user, err := s.userRepo.FindByID(userID)
	if err != nil {
		return ErrUserNotFound
	}

	// 验证旧密码
	if !CheckPassword(oldPassword, user.PasswordHash) {
		return ErrWrongPassword
	}

	// 生成新密码哈希
	passwordHash, err := HashPassword(newPassword)
	if err != nil {
		return err
	}

	return s.userRepo.UpdatePassword(userID, passwordHash)
}

// ========== 系统管理员功能 ==========

// GetAllShopAdmins 获取所有店铺管理员（系统管理员调用）
func (s *UserService) GetAllShopAdmins() ([]dto.ShopAdminInfo, error) {
	users, err := s.userRepo.FindShopAdmins()
	if err != nil {
		return nil, err
	}

	result := make([]dto.ShopAdminInfo, 0, len(users))
	for _, user := range users {
		// 获取店铺数量
		shopCount, _ := s.shopRepo.CountByOwnerID(user.ID)
		// 获取员工数量
		staffCount, _ := s.userRepo.CountByOwnerID(user.ID)

		result = append(result, dto.ShopAdminInfo{
			ID:          user.ID,
			Username:    user.Username,
			DisplayName: user.DisplayName,
			Status:      user.Status,
			ShopCount:   shopCount,
			StaffCount:  staffCount,
			CreatedAt:   user.CreatedAt,
			LastLoginAt: user.LastLoginAt,
		})
	}

	return result, nil
}

// GetShopAdminDetail 获取店铺管理员详情（系统管理员调用）
func (s *UserService) GetShopAdminDetail(shopAdminID uint) (*dto.ShopAdminDetail, error) {
	user, err := s.userRepo.FindShopAdminWithDetails(shopAdminID)
	if err != nil {
		return nil, ErrUserNotFound
	}

	// 获取店铺列表
	shops, _ := s.shopRepo.FindByOwnerID(shopAdminID)
	shopInfos := make([]dto.ShopInfo, 0, len(shops))
	for _, shop := range shops {
		shopInfos = append(shopInfos, dto.ShopInfo{
			ID:       shop.ID,
			Name:     shop.Name,
			IsActive: shop.IsActive,
		})
	}

	// 获取员工列表
	staffInfos := make([]dto.UserInfo, 0, len(user.Staff))
	for _, staff := range user.Staff {
		staffShops := make([]dto.ShopInfo, 0)
		for _, shop := range staff.Shops {
			staffShops = append(staffShops, dto.ShopInfo{
				ID:   shop.ID,
				Name: shop.Name,
			})
		}
		staffInfos = append(staffInfos, dto.UserInfo{
			ID:          staff.ID,
			Username:    staff.Username,
			DisplayName: staff.DisplayName,
			Role:        staff.Role,
			Status:      staff.Status,
			Shops:       staffShops,
		})
	}

	return &dto.ShopAdminDetail{
		ID:          user.ID,
		Username:    user.Username,
		DisplayName: user.DisplayName,
		Status:      user.Status,
		CreatedAt:   user.CreatedAt,
		LastLoginAt: user.LastLoginAt,
		Shops:       shopInfos,
		Staff:       staffInfos,
	}, nil
}

// CreateShopAdmin 创建店铺管理员（系统管理员调用）
func (s *UserService) CreateShopAdmin(req *dto.CreateShopAdminRequest) (*dto.ShopAdminInfo, error) {
	// 检查用户名是否已存在
	existing, _ := s.userRepo.FindByUsername(req.Username)
	if existing != nil {
		return nil, ErrUsernameExists
	}

	// 生成密码哈希
	passwordHash, err := HashPassword(req.Password)
	if err != nil {
		return nil, err
	}

	// 创建用户
	user := &model.User{
		Username:     req.Username,
		PasswordHash: passwordHash,
		DisplayName:  req.DisplayName,
		Role:         model.RoleShopAdmin,
		Status:       "active",
	}

	if err := s.userRepo.Create(user); err != nil {
		return nil, err
	}

	return &dto.ShopAdminInfo{
		ID:          user.ID,
		Username:    user.Username,
		DisplayName: user.DisplayName,
		Status:      user.Status,
		ShopCount:   0,
		StaffCount:  0,
		CreatedAt:   user.CreatedAt,
	}, nil
}

// UpdateShopAdminStatus 更新店铺管理员状态（系统管理员调用）
func (s *UserService) UpdateShopAdminStatus(shopAdminID uint, status string) error {
	user, err := s.userRepo.FindByID(shopAdminID)
	if err != nil {
		return ErrUserNotFound
	}

	// 不能修改系统管理员
	if user.IsSuperAdmin() {
		return ErrCannotModifySuperAdmin
	}

	// 必须是店铺管理员
	if !user.IsShopAdmin() {
		return ErrNotShopAdmin
	}

	return s.userRepo.UpdateStatus(shopAdminID, status)
}

// ResetShopAdminPassword 重置店铺管理员密码（系统管理员调用）
func (s *UserService) ResetShopAdminPassword(shopAdminID uint, newPassword string) error {
	user, err := s.userRepo.FindByID(shopAdminID)
	if err != nil {
		return ErrUserNotFound
	}

	// 不能修改系统管理员
	if user.IsSuperAdmin() {
		return ErrCannotModifySuperAdmin
	}

	// 必须是店铺管理员
	if !user.IsShopAdmin() {
		return ErrNotShopAdmin
	}

	passwordHash, err := HashPassword(newPassword)
	if err != nil {
		return err
	}

	return s.userRepo.UpdatePassword(shopAdminID, passwordHash)
}

// DeleteShopAdmin 删除店铺管理员（系统管理员调用）
func (s *UserService) DeleteShopAdmin(shopAdminID uint) error {
	user, err := s.userRepo.FindByID(shopAdminID)
	if err != nil {
		return ErrUserNotFound
	}

	// 不能删除系统管理员
	if user.IsSuperAdmin() {
		return ErrCannotModifySuperAdmin
	}

	// 必须是店铺管理员
	if !user.IsShopAdmin() {
		return ErrNotShopAdmin
	}

	return s.userRepo.Delete(shopAdminID)
}

// ========== 店铺管理员功能 ==========

// GetMyStaff 获取自己的员工列表（店铺管理员调用）
func (s *UserService) GetMyStaff(ownerID uint) ([]dto.UserInfo, error) {
	users, err := s.userRepo.FindByOwnerID(ownerID)
	if err != nil {
		return nil, err
	}

	result := make([]dto.UserInfo, 0, len(users))
	for _, user := range users {
		shops := make([]dto.ShopInfo, 0)
		for _, shop := range user.Shops {
			shops = append(shops, dto.ShopInfo{
				ID:   shop.ID,
				Name: shop.Name,
			})
		}
		result = append(result, dto.UserInfo{
			ID:          user.ID,
			Username:    user.Username,
			DisplayName: user.DisplayName,
			Role:        user.Role,
			Status:      user.Status,
			Shops:       shops,
		})
	}

	return result, nil
}

// CreateStaff 创建员工（店铺管理员调用）
func (s *UserService) CreateStaff(req *dto.CreateStaffRequest, ownerID uint) (*dto.UserInfo, error) {
	// 检查用户名是否已存在
	existing, _ := s.userRepo.FindByUsername(req.Username)
	if existing != nil {
		return nil, ErrUsernameExists
	}

	// 验证店铺都属于当前店铺管理员
	for _, shopID := range req.ShopIDs {
		if !s.shopRepo.IsOwner(ownerID, shopID) {
			return nil, ErrShopNotBelongToYou
		}
	}

	// 生成密码哈希
	passwordHash, err := HashPassword(req.Password)
	if err != nil {
		return nil, err
	}

	// 创建用户
	user := &model.User{
		Username:     req.Username,
		PasswordHash: passwordHash,
		DisplayName:  req.DisplayName,
		Role:         model.RoleStaff,
		Status:       "active",
		OwnerID:      &ownerID,
	}

	if err := s.userRepo.Create(user); err != nil {
		return nil, err
	}

	// 分配店铺
	if len(req.ShopIDs) > 0 {
		if err := s.userRepo.UpdateShops(user.ID, req.ShopIDs); err != nil {
			return nil, err
		}
	}

	// 获取完整用户信息
	user, _ = s.userRepo.FindByID(user.ID)

	shops := make([]dto.ShopInfo, 0)
	for _, shop := range user.Shops {
		shops = append(shops, dto.ShopInfo{
			ID:   shop.ID,
			Name: shop.Name,
		})
	}

	return &dto.UserInfo{
		ID:          user.ID,
		Username:    user.Username,
		DisplayName: user.DisplayName,
		Role:        user.Role,
		Shops:       shops,
	}, nil
}

// UpdateStaffStatus 更新员工状态（店铺管理员调用）
func (s *UserService) UpdateStaffStatus(staffID uint, status string, ownerID uint) error {
	user, err := s.userRepo.FindByID(staffID)
	if err != nil {
		return ErrUserNotFound
	}

	// 验证员工属于当前店铺管理员
	if user.OwnerID == nil || *user.OwnerID != ownerID {
		return ErrStaffNotBelongToYou
	}

	return s.userRepo.UpdateStatus(staffID, status)
}

// ResetStaffPassword 重置员工密码（店铺管理员调用）
func (s *UserService) ResetStaffPassword(staffID uint, newPassword string, ownerID uint) error {
	user, err := s.userRepo.FindByID(staffID)
	if err != nil {
		return ErrUserNotFound
	}

	// 验证员工属于当前店铺管理员
	if user.OwnerID == nil || *user.OwnerID != ownerID {
		return ErrStaffNotBelongToYou
	}

	passwordHash, err := HashPassword(newPassword)
	if err != nil {
		return err
	}

	return s.userRepo.UpdatePassword(staffID, passwordHash)
}

// UpdateStaffShops 更新员工可访问的店铺（店铺管理员调用）
func (s *UserService) UpdateStaffShops(staffID uint, shopIDs []uint, ownerID uint) error {
	user, err := s.userRepo.FindByID(staffID)
	if err != nil {
		return ErrUserNotFound
	}

	// 验证员工属于当前店铺管理员
	if user.OwnerID == nil || *user.OwnerID != ownerID {
		return ErrStaffNotBelongToYou
	}

	// 验证所有店铺都属于当前店铺管理员
	for _, shopID := range shopIDs {
		if !s.shopRepo.IsOwner(ownerID, shopID) {
			return ErrShopNotBelongToYou
		}
	}

	return s.userRepo.UpdateShops(staffID, shopIDs)
}

// DeleteStaff 删除员工（店铺管理员调用）
func (s *UserService) DeleteStaff(staffID uint, ownerID uint) error {
	user, err := s.userRepo.FindByID(staffID)
	if err != nil {
		return ErrUserNotFound
	}

	// 验证员工属于当前店铺管理员
	if user.OwnerID == nil || *user.OwnerID != ownerID {
		return ErrStaffNotBelongToYou
	}

	return s.userRepo.Delete(staffID)
}

// ========== 通用功能 ==========

// GetAccessibleShops 获取用户可访问的店铺
func (s *UserService) GetAccessibleShops(userID uint, role string) ([]dto.ShopInfo, error) {
	var shops []model.Shop
	var err error

	switch role {
	case model.RoleSuperAdmin:
		// 系统管理员可以看到所有店铺（只读）
		shops, err = s.shopRepo.FindAll()
	case model.RoleShopAdmin:
		// 店铺管理员看到自己的店铺
		shops, err = s.shopRepo.FindByOwnerID(userID)
	case model.RoleStaff:
		// 员工看到被分配的店铺
		shops, err = s.shopRepo.FindByUserID(userID)
	default:
		return nil, errors.New("未知角色")
	}

	if err != nil {
		return nil, err
	}

	result := make([]dto.ShopInfo, 0, len(shops))
	for _, shop := range shops {
		result = append(result, dto.ShopInfo{
			ID:       shop.ID,
			Name:     shop.Name,
			IsActive: shop.IsActive,
		})
	}

	return result, nil
}
