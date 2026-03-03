package service

import (
	"errors"
	"strconv"
	"strings"

	"ozon-manager/internal/dto"
	"ozon-manager/internal/model"
	"ozon-manager/internal/repository"
)

var (
	ErrShopNotFound         = errors.New("店铺不存在")
	ErrClientIDExists       = errors.New("Client ID已存在")
	ErrNoAccessToShop       = errors.New("无权访问该店铺")
	ErrShopNotBelongToYou   = errors.New("该店铺不属于您")
	ErrActiveClientIDExists = errors.New("已存在使用该 Client ID 的可用店铺")
	ErrInvalidClientID      = errors.New("Client ID必须是正整数")
	ErrInvalidEngineMode    = errors.New("执行引擎模式无效")
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
			ID:                  shop.ID,
			Name:                shop.Name,
			ExecutionEngineMode: normalizeExecutionEngineModeOrDefault(shop.ExecutionEngineMode),
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
	normalizedClientID, err := normalizeShopClientID(req.ClientID)
	if err != nil {
		return nil, err
	}

	// 检查ClientID是否已存在
	existing, _ := s.shopRepo.FindByClientID(normalizedClientID)
	if existing != nil {
		return nil, ErrClientIDExists
	}

	shop := &model.Shop{
		Name:                req.Name,
		ClientID:            normalizedClientID,
		ApiKey:              req.ApiKey,
		IsActive:            true,
		ExecutionEngineMode: model.ShopExecutionEngineAuto,
	}

	if err := s.shopRepo.Create(shop); err != nil {
		return nil, err
	}

	return &dto.ShopInfo{
		ID:                  shop.ID,
		Name:                shop.Name,
		ExecutionEngineMode: normalizeExecutionEngineModeOrDefault(shop.ExecutionEngineMode),
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
		normalizedClientID, normalizeErr := normalizeShopClientID(req.ClientID)
		if normalizeErr != nil {
			return normalizeErr
		}

		// 检查新ClientID是否已被其他店铺使用
		existing, _ := s.shopRepo.FindByClientID(normalizedClientID)
		if existing != nil && existing.ID != shopID {
			return ErrClientIDExists
		}
		shop.ClientID = normalizedClientID
	}
	if req.ApiKey != "" {
		shop.ApiKey = req.ApiKey
	}
	if req.IsActive != nil {
		shop.IsActive = *req.IsActive
	}
	if req.ExecutionEngineMode != "" {
		engineMode, normalizeErr := normalizeExecutionEngineMode(req.ExecutionEngineMode)
		if normalizeErr != nil {
			return normalizeErr
		}
		shop.ExecutionEngineMode = engineMode
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

// ========== 店铺管理员功能 ==========

// CreateMyShop 店铺管理员创建自己的店铺
func (s *ShopService) CreateMyShop(req *dto.CreateShopRequest, ownerID uint) (*dto.ShopInfo, error) {
	normalizedClientID, err := normalizeShopClientID(req.ClientID)
	if err != nil {
		return nil, err
	}

	// 检查是否存在同 ClientID 且为可用的店铺
	activeShop, _ := s.shopRepo.FindActiveByClientID(normalizedClientID)
	if activeShop != nil {
		return nil, ErrActiveClientIDExists
	}

	shop := &model.Shop{
		Name:                req.Name,
		ClientID:            normalizedClientID,
		ApiKey:              req.ApiKey,
		IsActive:            true,
		ExecutionEngineMode: model.ShopExecutionEngineAuto,
		OwnerID:             ownerID,
	}

	if err := s.shopRepo.Create(shop); err != nil {
		return nil, err
	}

	return &dto.ShopInfo{
		ID:                  shop.ID,
		Name:                shop.Name,
		IsActive:            shop.IsActive,
		ExecutionEngineMode: normalizeExecutionEngineModeOrDefault(shop.ExecutionEngineMode),
	}, nil
}

// GetMyShops 获取店铺管理员自己的店铺
func (s *ShopService) GetMyShops(ownerID uint) ([]dto.ShopInfo, error) {
	shops, err := s.shopRepo.FindByOwnerID(ownerID)
	if err != nil {
		return nil, err
	}

	result := make([]dto.ShopInfo, 0, len(shops))
	for _, shop := range shops {
		result = append(result, dto.ShopInfo{
			ID:                  shop.ID,
			Name:                shop.Name,
			IsActive:            shop.IsActive,
			ExecutionEngineMode: normalizeExecutionEngineModeOrDefault(shop.ExecutionEngineMode),
		})
	}

	return result, nil
}

// UpdateMyShop 店铺管理员更新自己的店铺
func (s *ShopService) UpdateMyShop(shopID uint, req *dto.UpdateShopRequest, ownerID uint) error {
	shop, err := s.shopRepo.FindByID(shopID)
	if err != nil {
		return ErrShopNotFound
	}

	// 验证店铺属于当前店铺管理员
	if shop.OwnerID != ownerID {
		return ErrShopNotBelongToYou
	}

	if req.Name != "" {
		shop.Name = req.Name
	}
	if req.ClientID != "" {
		normalizedClientID, normalizeErr := normalizeShopClientID(req.ClientID)
		if normalizeErr != nil {
			return normalizeErr
		}

		// 检查新ClientID是否已被其他店铺使用
		existing, _ := s.shopRepo.FindByClientID(normalizedClientID)
		if existing != nil && existing.ID != shopID {
			return ErrClientIDExists
		}
		shop.ClientID = normalizedClientID
	}
	if req.ApiKey != "" {
		shop.ApiKey = req.ApiKey
	}
	if req.IsActive != nil {
		shop.IsActive = *req.IsActive
	}
	if req.ExecutionEngineMode != "" {
		engineMode, normalizeErr := normalizeExecutionEngineMode(req.ExecutionEngineMode)
		if normalizeErr != nil {
			return normalizeErr
		}
		shop.ExecutionEngineMode = engineMode
	}

	return s.shopRepo.Update(shop)
}

// DeleteMyShop 店铺管理员删除自己的店铺
func (s *ShopService) DeleteMyShop(shopID uint, ownerID uint) error {
	shop, err := s.shopRepo.FindByID(shopID)
	if err != nil {
		return ErrShopNotFound
	}

	// 验证店铺属于当前店铺管理员
	if shop.OwnerID != ownerID {
		return ErrShopNotBelongToYou
	}

	return s.shopRepo.Delete(shopID)
}

// IsShopOwner 检查用户是否是店铺的所有者
func (s *ShopService) IsShopOwner(userID, shopID uint) bool {
	return s.shopRepo.IsOwner(userID, shopID)
}

// ========== 三层角色权限检查 ==========

func (s *ShopService) CheckUserAccessByRole(userID, shopID uint, role string) error {
	switch role {
	case model.RoleSuperAdmin:
		// 系统管理员可以查看所有店铺（只读）
		return nil
	case model.RoleShopAdmin:
		// 店铺管理员只能访问自己的店铺
		if s.shopRepo.IsOwner(userID, shopID) {
			return nil
		}
		return ErrNoAccessToShop
	case model.RoleStaff:
		// 员工只能访问被分配的店铺
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
	default:
		return ErrNoAccessToShop
	}
}

// GetAccessibleShopsByRole 根据角色获取用户可访问的店铺
func (s *ShopService) GetAccessibleShopsByRole(userID uint, role string) ([]dto.ShopInfo, error) {
	var shops []model.Shop
	var err error

	switch role {
	case model.RoleSuperAdmin:
		shops, err = s.shopRepo.FindAll()
	case model.RoleShopAdmin:
		shops, err = s.shopRepo.FindByOwnerID(userID)
	case model.RoleStaff:
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
			ID:                  shop.ID,
			Name:                shop.Name,
			IsActive:            shop.IsActive,
			ExecutionEngineMode: normalizeExecutionEngineModeOrDefault(shop.ExecutionEngineMode),
		})
	}

	return result, nil
}

// GetSystemOverview 获取系统概览（系统管理员调用）
func (s *ShopService) GetSystemOverview() (*dto.SystemOverviewResponse, error) {
	shopAdminCount, _ := s.userRepo.CountByRole(model.RoleShopAdmin)
	shopCount, _ := s.shopRepo.CountAll()
	staffCount, _ := s.userRepo.CountByRole(model.RoleStaff)

	return &dto.SystemOverviewResponse{
		ShopAdminCount: shopAdminCount,
		ShopCount:      shopCount,
		StaffCount:     staffCount,
	}, nil
}

func (s *ShopService) GetMyShopExecutionEngine(shopID uint, ownerID uint) (*dto.ShopExecutionEngineResponse, error) {
	shop, err := s.shopRepo.FindByID(shopID)
	if err != nil {
		return nil, ErrShopNotFound
	}
	if shop.OwnerID != ownerID {
		return nil, ErrShopNotBelongToYou
	}

	return &dto.ShopExecutionEngineResponse{
		ShopID:              shop.ID,
		ExecutionEngineMode: normalizeExecutionEngineModeOrDefault(shop.ExecutionEngineMode),
	}, nil
}

func (s *ShopService) UpdateMyShopExecutionEngine(shopID uint, ownerID uint, mode string) (*dto.ShopExecutionEngineResponse, error) {
	shop, err := s.shopRepo.FindByID(shopID)
	if err != nil {
		return nil, ErrShopNotFound
	}
	if shop.OwnerID != ownerID {
		return nil, ErrShopNotBelongToYou
	}

	normalized, err := normalizeExecutionEngineMode(mode)
	if err != nil {
		return nil, err
	}

	if err := s.shopRepo.UpdateExecutionEngineMode(shopID, normalized); err != nil {
		return nil, err
	}

	return &dto.ShopExecutionEngineResponse{
		ShopID:              shop.ID,
		ExecutionEngineMode: normalized,
	}, nil
}

func normalizeShopClientID(clientID string) (string, error) {
	trimmed := strings.TrimSpace(clientID)
	if trimmed == "" {
		return "", ErrInvalidClientID
	}

	parsed, err := strconv.ParseUint(trimmed, 10, 64)
	if err != nil || parsed == 0 {
		return "", ErrInvalidClientID
	}

	return strconv.FormatUint(parsed, 10), nil
}

func normalizeExecutionEngineMode(mode string) (string, error) {
	normalized := normalizeExecutionEngineModeOrDefault(mode)
	switch normalized {
	case model.ShopExecutionEngineAuto, model.ShopExecutionEngineExtension, model.ShopExecutionEngineAgent:
		return normalized, nil
	default:
		return "", ErrInvalidEngineMode
	}
}

func normalizeExecutionEngineModeOrDefault(mode string) string {
	normalized := strings.TrimSpace(strings.ToLower(mode))
	if normalized == "" {
		return model.ShopExecutionEngineAuto
	}
	return normalized
}
