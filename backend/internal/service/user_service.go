package service

import (
	"errors"

	"ozon-manager/internal/dto"
	"ozon-manager/internal/model"
	"ozon-manager/internal/repository"
)

var (
	ErrUsernameExists = errors.New("用户名已存在")
	ErrCannotModifyAdmin = errors.New("不能修改管理员账号")
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
