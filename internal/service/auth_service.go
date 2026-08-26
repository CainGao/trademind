// Package service — 认证业务逻辑（登录/注册/刷新 Token）。
//
// 规范 §6.1: 密码 bcrypt cost=12
// 规范 §6.3: Access 2h + Refresh 7d
// 规范 §6.7: 登录记审计日志
package service

import (
	"errors"
	"time"

	"github.com/CainGao/trademind/internal/database"
	"github.com/CainGao/trademind/internal/middleware"
	"github.com/CainGao/trademind/internal/models"
	"github.com/CainGao/trademind/internal/pkg/crypto"
	"github.com/CainGao/trademind/internal/repository"
	"gorm.io/gorm"
)

var (
	// ErrInvalidCredentials 用户名或密码错误。
	ErrInvalidCredentials = errors.New("用户名或密码错误")
	// ErrUsernameTaken 用户名已被占用。
	ErrUsernameTaken = errors.New("用户名已被占用")
	// ErrUserInactive 用户已禁用。
	ErrUserInactive = errors.New("账号已被禁用")
	// ErrWeakPassword 密码命中弱密码黑名单（gotcha #88）。
	ErrWeakPassword = errors.New("密码过于简单（命中常见弱密码黑名单），请更换")
)

// AuthService 认证业务。
type AuthService struct {
	userRepo   *repository.UserRepo
	auditRepo  *repository.AuditLogRepo
	jwt        *middleware.JWTManager
}

// NewAuthService 创建认证服务。
func NewAuthService(userRepo *repository.UserRepo, auditRepo *repository.AuditLogRepo, jwt *middleware.JWTManager) *AuthService {
	return &AuthService{userRepo: userRepo, auditRepo: auditRepo, jwt: jwt}
}

// LoginInput 登录入参。
type LoginInput struct {
	Username string `json:"username" binding:"required,min=2,max=50"`
	Password string `json:"password" binding:"required,min=6,max=64"`
}

// TokenPair 登录返回的 Token 对。
type TokenPair struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	ExpiresAt    time.Time `json:"expires_at"` // access token 过期时间
	User         UserInfo  `json:"user"`
}

// UserInfo 返回给前端的用户信息（不含密码 hash）。
type UserInfo struct {
	ID       uint            `json:"id"`
	Username string          `json:"username"`
	Nickname string          `json:"nickname"`
	Role     models.UserRole `json:"role"`
	Avatar   string          `json:"avatar"`
	// MustChangePassword 登录时检测到 admin 仍在使用 seed 默认密码时为 true。
	// 前端据此显示改密提醒横幅（gotcha #88 默认密码治理）。
	MustChangePassword bool `json:"must_change_password"`
}

// Login 校验账号密码，签发 Token 对。
func (s *AuthService) Login(input LoginInput, ip string) (*TokenPair, error) {
	u, err := s.userRepo.GetByUsername(input.Username)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrInvalidCredentials
		}
		return nil, err
	}
	if u.Status == models.UserStatusInactive {
		return nil, ErrUserInactive
	}
	if err := crypto.CheckPassword(u.PasswordHash, input.Password); err != nil {
		// 记录失败审计
		s.auditRepo.Log(u.ID, "login_failed", "user", &u.ID, "密码错误", ip)
		return nil, ErrInvalidCredentials
	}

	// 签发 Token
	access, err := s.jwt.GenerateAccessToken(u)
	if err != nil {
		return nil, err
	}
	refresh, err := s.jwt.GenerateRefreshToken(u)
	if err != nil {
		return nil, err
	}

	// 更新最后登录时间
	now := time.Now()
	s.userRepo.UpdateLastLogin(u.ID, now)
	// 登录审计
	s.auditRepo.Log(u.ID, "login", "user", &u.ID, "登录成功", ip)

	// 默认密码检测（gotcha #88）：admin 仍用 seed 默认密码时置标志，
	// 前端登录后显示改密提醒横幅。登录时明文在手，一次字符串比较零成本。
	mustChange := u.Username == "admin" && input.Password == database.DefaultAdminPassword
	if mustChange {
		s.auditRepo.Log(u.ID, "weak_password_warning", "user", &u.ID,
			"管理员仍在使用默认密码", ip)
	}

	return &TokenPair{
		AccessToken:  access,
		RefreshToken: refresh,
		ExpiresAt:    now.Add(2 * time.Hour),
		User: UserInfo{
			ID: u.ID, Username: u.Username, Nickname: u.Nickname,
			Role: u.Role, Avatar: u.Avatar,
			MustChangePassword: mustChange,
		},
	}, nil
}

// RegisterInput 注册入参。
type RegisterInput struct {
	Username string `json:"username" binding:"required,min=2,max=50"`
	Password string `json:"password" binding:"required,min=6,max=64"`
	Nickname string `json:"nickname" binding:"omitempty,max=50"`
	Role     models.UserRole `json:"role" binding:"omitempty,oneof=admin boss sourcing sales operator staff"`
}

// Register 创建新用户。
func (s *AuthService) Register(input RegisterInput) (*UserInfo, error) {
	// 检查重名
	if existing, _ := s.userRepo.GetByUsername(input.Username); existing != nil {
		return nil, ErrUsernameTaken
	}
	// 弱密码黑名单拦截（gotcha #88）
	if isWeakPassword(input.Password) {
		return nil, ErrWeakPassword
	}

	hash, err := crypto.HashPassword(input.Password)
	if err != nil {
		return nil, err
	}

	role := input.Role
	if role == "" {
		role = models.RoleStaff
	}

	u := &models.User{
		Username:     input.Username,
		PasswordHash: hash,
		Nickname:     input.Nickname,
		Role:         role,
		Status:       models.UserStatusActive,
	}
	if err := s.userRepo.Create(u); err != nil {
		return nil, err
	}

	return &UserInfo{
		ID: u.ID, Username: u.Username, Nickname: u.Nickname,
		Role: u.Role,
	}, nil
}

// RefreshInput 刷新 Token 入参。
type RefreshInput struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

// Refresh 用 Refresh Token 换新的 Access Token。
func (s *AuthService) Refresh(input RefreshInput) (*TokenPair, error) {
	claims, err := s.jwt.Parse(input.RefreshToken)
	if err != nil {
		return nil, errors.New("refresh token 无效或已过期")
	}
	// 校验是 refresh 类型
	if len(claims.RegisteredClaims.Audience) == 0 || claims.RegisteredClaims.Audience[0] != "refresh" {
		return nil, errors.New("token 类型错误")
	}
	u, err := s.userRepo.GetByID(claims.UserID)
	if err != nil {
		return nil, err
	}
	if u.Status == models.UserStatusInactive {
		return nil, ErrUserInactive
	}

	access, err := s.jwt.GenerateAccessToken(u)
	if err != nil {
		return nil, err
	}
	refresh, err := s.jwt.GenerateRefreshToken(u)
	if err != nil {
		return nil, err
	}
	now := time.Now()
	return &TokenPair{
		AccessToken:  access,
		RefreshToken: refresh,
		ExpiresAt:    now.Add(2 * time.Hour),
		User: UserInfo{
			ID: u.ID, Username: u.Username, Nickname: u.Nickname,
			Role: u.Role, Avatar: u.Avatar,
		},
	}, nil
}
