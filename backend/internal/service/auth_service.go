package service

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"time"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
	"ozon-manager/internal/config"
	"ozon-manager/internal/dto"
	"ozon-manager/internal/model"
	"ozon-manager/internal/repository"
	"ozon-manager/pkg/hash"
	"ozon-manager/pkg/jwt"
)

var (
	ErrInvalidCredentials   = errors.New("用户名或密码错误")
	ErrUserDisabled         = errors.New("账号已被禁用")
	ErrUserNotFound         = errors.New("用户不存在")
	ErrRefreshTokenRequired = errors.New("缺少 refresh token")
	ErrInvalidRefreshToken  = errors.New("refresh token 无效或已过期")
)

const (
	refreshTokenBytes  = 32
	refreshFamilyBytes = 16
)

type AuthSessionMeta struct {
	UserAgent string
	IPAddress string
}

type AuthSession struct {
	Response     *dto.LoginResponse
	RefreshToken string
}

type AuthService struct {
	userRepo    *repository.UserRepository
	shopRepo    *repository.ShopRepository
	refreshRepo *repository.RefreshTokenRepository
}

func NewAuthService(userRepo *repository.UserRepository, shopRepo *repository.ShopRepository, refreshRepo *repository.RefreshTokenRepository) *AuthService {
	return &AuthService{
		userRepo:    userRepo,
		shopRepo:    shopRepo,
		refreshRepo: refreshRepo,
	}
}

// Login 用户登录
func (s *AuthService) Login(req *dto.LoginRequest, meta AuthSessionMeta) (*AuthSession, error) {
	// 查找用户
	user, err := s.userRepo.FindByUsername(req.Username)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrInvalidCredentials
		}
		return nil, err
	}

	// 验证密码
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		return nil, ErrInvalidCredentials
	}

	// 检查账号状态
	if !user.IsActive() {
		return nil, ErrUserDisabled
	}

	session, err := s.issueSession(user, meta, "")
	if err != nil {
		return nil, err
	}

	// 更新最后登录时间
	s.userRepo.UpdateLastLogin(user.ID)

	return session, nil
}

func (s *AuthService) Refresh(refreshToken string, meta AuthSessionMeta) (*AuthSession, error) {
	if refreshToken == "" {
		return nil, ErrRefreshTokenRequired
	}

	tokenHash := hash.SHA256Hash(refreshToken)
	storedToken, err := s.refreshRepo.FindByTokenHash(tokenHash)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrInvalidRefreshToken
		}
		return nil, err
	}

	if storedToken.RevokedAt != nil {
		if storedToken.FamilyID != "" {
			if revokeErr := s.refreshRepo.RevokeFamily(storedToken.FamilyID, "refresh_token_reuse_detected"); revokeErr != nil {
				return nil, revokeErr
			}
		}
		return nil, ErrInvalidRefreshToken
	}

	now := time.Now()
	if !storedToken.ExpiresAt.After(now) {
		if err := s.refreshRepo.RevokeByID(storedToken.ID, "refresh_token_expired"); err != nil {
			return nil, err
		}
		return nil, ErrInvalidRefreshToken
	}

	user, err := s.userRepo.FindByID(storedToken.UserID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrInvalidRefreshToken
		}
		return nil, err
	}
	if !user.IsActive() {
		if err := s.refreshRepo.RevokeAllByUserID(user.ID, "user_disabled"); err != nil {
			return nil, err
		}
		return nil, ErrUserDisabled
	}

	accessToken, tokenExpiresAt, err := s.generateAccessToken(user)
	if err != nil {
		return nil, err
	}

	newRefreshToken, refreshRecord, err := s.buildRefreshTokenRecord(user.ID, meta, storedToken.FamilyID)
	if err != nil {
		return nil, err
	}

	if err := s.refreshRepo.Rotate(storedToken.ID, "rotated", refreshRecord); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			if revokeErr := s.refreshRepo.RevokeFamily(storedToken.FamilyID, "refresh_token_reuse_detected"); revokeErr != nil {
				return nil, revokeErr
			}
			return nil, ErrInvalidRefreshToken
		}
		return nil, err
	}

	response, err := s.buildLoginResponse(user, accessToken, tokenExpiresAt)
	if err != nil {
		return nil, err
	}

	return &AuthSession{
		Response:     response,
		RefreshToken: newRefreshToken,
	}, nil
}

func (s *AuthService) Logout(refreshToken string) error {
	if refreshToken == "" {
		return nil
	}

	storedToken, err := s.refreshRepo.FindByTokenHash(hash.SHA256Hash(refreshToken))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		return err
	}

	if storedToken.RevokedAt != nil {
		return nil
	}

	return s.refreshRepo.RevokeByID(storedToken.ID, "logout")
}

// GetCurrentUser 获取当前用户信息
func (s *AuthService) GetCurrentUser(userID uint) (*dto.UserInfo, error) {
	user, err := s.userRepo.FindByID(userID)
	if err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}
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
		Status:      user.Status,
		Shops:       shops,
	}, nil
}

func (s *AuthService) issueSession(user *model.User, meta AuthSessionMeta, familyID string) (*AuthSession, error) {
	accessToken, tokenExpiresAt, err := s.generateAccessToken(user)
	if err != nil {
		return nil, err
	}

	refreshToken, refreshRecord, err := s.buildRefreshTokenRecord(user.ID, meta, familyID)
	if err != nil {
		return nil, err
	}
	if err := s.refreshRepo.Create(refreshRecord); err != nil {
		return nil, err
	}

	response, err := s.buildLoginResponse(user, accessToken, tokenExpiresAt)
	if err != nil {
		return nil, err
	}

	return &AuthSession{
		Response:     response,
		RefreshToken: refreshToken,
	}, nil
}

func (s *AuthService) generateAccessToken(user *model.User) (string, time.Time, error) {
	expiresAt := time.Now().Add(accessTokenTTL(config.GetConfig().JWT))
	token, err := jwt.GenerateAccessToken(user.ID, user.Username, user.DisplayName, user.Role)
	if err != nil {
		return "", time.Time{}, err
	}
	return token, expiresAt, nil
}

func (s *AuthService) buildRefreshTokenRecord(userID uint, meta AuthSessionMeta, familyID string) (string, *model.UserRefreshToken, error) {
	if familyID == "" {
		var err error
		familyID, err = generateOpaqueToken(refreshFamilyBytes)
		if err != nil {
			return "", nil, err
		}
	}

	token, err := generateOpaqueToken(refreshTokenBytes)
	if err != nil {
		return "", nil, err
	}

	now := time.Now()
	return token, &model.UserRefreshToken{
		UserID:    userID,
		TokenHash: hash.SHA256Hash(token),
		FamilyID:  familyID,
		UserAgent: meta.UserAgent,
		IPAddress: meta.IPAddress,
		IssuedAt:  now,
		ExpiresAt: now.Add(refreshTokenTTL(config.GetConfig().JWT)),
	}, nil
}

func (s *AuthService) buildLoginResponse(user *model.User, accessToken string, tokenExpiresAt time.Time) (*dto.LoginResponse, error) {
	userInfo, err := s.GetCurrentUser(user.ID)
	if err != nil {
		return nil, err
	}

	return &dto.LoginResponse{
		Token:          accessToken,
		TokenExpiresAt: tokenExpiresAt,
		User:           *userInfo,
	}, nil
}

func generateOpaqueToken(size int) (string, error) {
	bytes := make([]byte, size)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(bytes), nil
}

func accessTokenTTL(cfg config.JWTConfig) time.Duration {
	if cfg.AccessExpireMinutes > 0 {
		return time.Duration(cfg.AccessExpireMinutes) * time.Minute
	}
	if cfg.ExpireHours > 0 {
		return time.Duration(cfg.ExpireHours) * time.Hour
	}
	return 24 * time.Hour
}

func refreshTokenTTL(cfg config.JWTConfig) time.Duration {
	if cfg.RefreshExpireHours > 0 {
		return time.Duration(cfg.RefreshExpireHours) * time.Hour
	}
	if cfg.ExpireHours > 0 {
		return time.Duration(cfg.ExpireHours) * time.Hour
	}
	return 7 * 24 * time.Hour
}

// HashPassword 对前端已做 SHA-256 的密码字符串生成 bcrypt 哈希
func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(bytes), err
}

// CheckPassword 校验前端已做 SHA-256 的密码字符串与存储哈希是否匹配
func CheckPassword(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}
