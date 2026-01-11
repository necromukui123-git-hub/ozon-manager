package service

import (
	"errors"

	"ozon-manager/internal/dto"
	"ozon-manager/internal/model"
	"ozon-manager/internal/repository"
)

var (
	ErrShopNotFound    = errors.New("店铺不存在")
	ErrClientIDExists  = errors.New("Client ID已存在")
	ErrNoAccessToShop  = errors.New("无权访问该店铺")
)

type ShopService struct {
	shopRepo *repository.ShopRepository
	userRepo *repository.UserRepository
}

func NewShopService(shopRepo *repository.ShopRepository, userRepo *repository.UserRepository) *ShopService {
	return &ShopService{
		shopRepo: shopRepo,
		userRepo: userRepo,
	}
}

// GetAllShops 获取所有店铺
func (s *ShopService) GetAllShops() ([]dto.ShopInfo, error) {
	shops, err := s.shopRepo.FindAll()
	if err != nil {
		return nil, err
	}

	result := make([]dto.ShopInfo, 0, len(shops))
	for _, shop := range shops {
		result = append(result, dto.ShopInfo{
			ID:   shop.ID,
			Name: shop.Name,
		})
	}

	return result, nil
}

// GetUserAccessibleShops 获取用户可访问的店铺
func (s *ShopService) GetUserAccessibleShops(userID uint, isAdmin bool) ([]dto.ShopInfo, error) {
	var shops []model.Shop
	var err error

	if isAdmin {
		shops, err = s.shopRepo.FindAll()
	} else {
		shops, err = s.shopRepo.FindByUserID(userID)
	}

	if err != nil {
		return nil, err
	}

	result := make([]dto.ShopInfo, 0, len(shops))
	for _, shop := range shops {
		result = append(result, dto.ShopInfo{
			ID:   shop.ID,
			Name: shop.Name,
		})
	}

	return result, nil
}

// GetShopByID 获取店铺详情
func (s *ShopService) GetShopByID(shopID uint) (*model.Shop, error) {
	shop, err := s.shopRepo.FindByID(shopID)
	if err != nil {
		return nil, ErrShopNotFound
	}
	return shop, nil
}

// CreateShop 创建店铺
func (s *ShopService) CreateShop(req *dto.CreateShopRequest) (*dto.ShopInfo, error) {
	// 检查ClientID是否已存在
	existing, _ := s.shopRepo.FindByClientID(req.ClientID)
	if existing != nil {
		return nil, ErrClientIDExists
	}

	shop := &model.Shop{
		Name:     req.Name,
		ClientID: req.ClientID,
		ApiKey:   req.ApiKey,
		IsActive: true,
	}

	if err := s.shopRepo.Create(shop); err != nil {
		return nil, err
	}

	return &dto.ShopInfo{
		ID:   shop.ID,
		Name: shop.Name,
	}, nil
}

// UpdateShop 更新店铺
func (s *ShopService) UpdateShop(shopID uint, req *dto.UpdateShopRequest) error {
	shop, err := s.shopRepo.FindByID(shopID)
	if err != nil {
		return ErrShopNotFound
	}

	if req.Name != "" {
		shop.Name = req.Name
	}
	if req.ClientID != "" {
		// 检查新ClientID是否已被其他店铺使用
		existing, _ := s.shopRepo.FindByClientID(req.ClientID)
		if existing != nil && existing.ID != shopID {
			return ErrClientIDExists
		}
		shop.ClientID = req.ClientID
	}
	if req.ApiKey != "" {
		shop.ApiKey = req.ApiKey
	}
	if req.IsActive != nil {
		shop.IsActive = *req.IsActive
	}

	return s.shopRepo.Update(shop)
}

// DeleteShop 删除店铺
func (s *ShopService) DeleteShop(shopID uint) error {
	_, err := s.shopRepo.FindByID(shopID)
	if err != nil {
		return ErrShopNotFound
	}

	return s.shopRepo.Delete(shopID)
}

// GetShopCredentials 获取店铺API凭证
func (s *ShopService) GetShopCredentials(shopID uint) (clientID, apiKey string, err error) {
	shop, err := s.shopRepo.GetWithCredentials(shopID)
	if err != nil {
		return "", "", ErrShopNotFound
	}
	return shop.ClientID, shop.ApiKey, nil
}

// CheckUserAccess 检查用户是否有权访问店铺
func (s *ShopService) CheckUserAccess(userID, shopID uint, isAdmin bool) error {
	if isAdmin {
		return nil
	}

	shops, err := s.shopRepo.FindByUserID(userID)
	if err != nil {
		return err
	}

	for _, shop := range shops {
		if shop.ID == shopID {
			return nil
		}
	}

	return ErrNoAccessToShop
}
