package service

import (
	"errors"

	"golang.org/x/crypto/bcrypt"
	"ozon-manager/internal/dto"
	"ozon-manager/internal/repository"
	"ozon-manager/pkg/jwt"
)

var (
	ErrInvalidCredentials = errors.New("用户名或密码错误")
	ErrUserDisabled       = errors.New("账号已被禁用")
	ErrUserNotFound       = errors.New("用户不存在")
)

type AuthService struct {
	userRepo *repository.UserRepository
	shopRepo *repository.ShopRepository
}

func NewAuthService(userRepo *repository.UserRepository, shopRepo *repository.ShopRepository) *AuthService {
	return &AuthService{
		userRepo: userRepo,
		shopRepo: shopRepo,
	}
}

// Login 用户登录
func (s *AuthService) Login(req *dto.LoginRequest) (*dto.LoginResponse, error) {
	// 查找用户
	user, err := s.userRepo.FindByUsername(req.Username)
	if err != nil {
		return nil, ErrInvalidCredentials
	}

	// 验证密码
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		return nil, ErrInvalidCredentials
	}

	// 检查账号状态
	if !user.IsActive() {
		return nil, ErrUserDisabled
	}

	// 生成JWT令牌
	token, err := jwt.GenerateToken(user.ID, user.Username, user.DisplayName, user.Role)
	if err != nil {
		return nil, err
	}

	// 更新最后登录时间
	s.userRepo.UpdateLastLogin(user.ID)

	// 构建响应
	shops := make([]dto.ShopInfo, 0)
	for _, shop := range user.Shops {
		shops = append(shops, dto.ShopInfo{
			ID:   shop.ID,
			Name: shop.Name,
		})
	}

	return &dto.LoginResponse{
		Token: token,
		User: dto.UserInfo{
			ID:          user.ID,
			Username:    user.Username,
			DisplayName: user.DisplayName,
			Role:        user.Role,
			Shops:       shops,
		},
	}, nil
}

// GetCurrentUser 获取当前用户信息
func (s *AuthService) GetCurrentUser(userID uint) (*dto.UserInfo, error) {
	user, err := s.userRepo.FindByID(userID)
	if err != nil {
		return nil, ErrUserNotFound
	}

	// 获取用户可访问的店铺
	var shops []dto.ShopInfo
	if user.IsAdmin() {
		// 管理员可访问所有店铺
		allShops, _ := s.shopRepo.FindAll()
		for _, shop := range allShops {
			shops = append(shops, dto.ShopInfo{
				ID:   shop.ID,
				Name: shop.Name,
			})
		}
	} else {
		for _, shop := range user.Shops {
			shops = append(shops, dto.ShopInfo{
				ID:   shop.ID,
				Name: shop.Name,
			})
		}
	}

	return &dto.UserInfo{
		ID:          user.ID,
		Username:    user.Username,
		DisplayName: user.DisplayName,
		Role:        user.Role,
		Shops:       shops,
	}, nil
}

// HashPassword 生成密码哈希
func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(bytes), err
}

// CheckPassword 验证密码
func CheckPassword(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}
