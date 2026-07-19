// Package middleware — JWT 鉴权 + Claims 定义。
//
// Token 策略（规范 V1.0 §6.3）：
//   - Access Token:  2 小时有效
//   - Refresh Token: 7 天有效
//   - 密钥首次启动随机生成 32 字节，持久化到 settings 表
package middleware

import (
	"errors"
	"time"

	"github.com/CainGao/trademind/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

// Claims JWT 声明。
type Claims struct {
	UserID   uint             `json:"user_id"`
	Username string           `json:"username"`
	Role     models.UserRole  `json:"role"`
	jwt.RegisteredClaims
}

// JWTManager 管理 Token 签发与解析。
type JWTManager struct {
	secret        []byte
	accessTTL     time.Duration // 默认 2h
	refreshTTL    time.Duration // 默认 7d
	issuer        string
}

// NewJWTManager 创建 JWT 管理器。secret 必须 ≥32 字节（规范 §6.3）。
func NewJWTManager(secret []byte) *JWTManager {
	if len(secret) < 32 {
		panic("JWT secret 必须 ≥32 字节")
	}
	return &JWTManager{
		secret:     secret,
		accessTTL:  2 * time.Hour,
		refreshTTL: 7 * 24 * time.Hour,
		issuer:     "trademind",
	}
}

// GenerateAccessToken 签发 Access Token（2h）。
func (m *JWTManager) GenerateAccessToken(u *models.User) (string, error) {
	return m.sign(u, m.accessTTL, "access")
}

// GenerateRefreshToken 签发 Refresh Token（7d）。
func (m *JWTManager) GenerateRefreshToken(u *models.User) (string, error) {
	return m.sign(u, m.refreshTTL, "refresh")
}

func (m *JWTManager) sign(u *models.User, ttl time.Duration, tokenType string) (string, error) {
	now := time.Now()
	claims := Claims{
		UserID:   u.ID,
		Username: u.Username,
		Role:     u.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    m.issuer,
			Subject:   u.Username,
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
		},
	}
	// 用 Audience 字段区分 token 类型（access / refresh）
	claims.RegisteredClaims.Audience = []string{tokenType}
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return tok.SignedString(m.secret)
}

// Parse 解析并校验 Token。
func (m *JWTManager) Parse(tokenStr string) (*Claims, error) {
	tok, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return m.secret, nil
	}, jwt.WithIssuer(m.issuer))
	if err != nil {
		return nil, err
	}
	claims, ok := tok.Claims.(*Claims)
	if !ok || !tok.Valid {
		return nil, errors.New("invalid token")
	}
	return claims, nil
}

// ===== Gin 中间件 =====

// JWT Gin 中间件：从 Authorization: Bearer <token> 解析用户。
// 失败返回 401（规范 §3.4: 错误码 2001/2002）。
func JWT(mgr *JWTManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		auth := c.GetHeader("Authorization")
		if len(auth) < 8 || auth[:7] != "Bearer " {
			c.AbortWithStatusJSON(401, gin.H{"code": 2001, "message": "未提供认证令牌"})
			return
		}
		tokenStr := auth[7:]
		claims, err := mgr.Parse(tokenStr)
		if err != nil {
			c.AbortWithStatusJSON(401, gin.H{"code": 2002, "message": "令牌无效或已过期"})
			return
		}
		// 注入到上下文，后续 handler 用 c.MustGet("user_id")
		c.Set("user_id", claims.UserID)
		c.Set("username", claims.Username)
		c.Set("role", claims.Role)
		c.Next()
	}
}

// RequireRole 角色守卫（可选）。仅允许指定角色通过。
//
// 用法：
//
//	r.GET("/admin/users", middleware.JWT(jwtMgr), middleware.RequireRole(models.RoleAdmin), h.List)
func RequireRole(roles ...models.UserRole) gin.HandlerFunc {
	allowed := make(map[models.UserRole]bool, len(roles))
	for _, r := range roles {
		allowed[r] = true
	}
	return func(c *gin.Context) {
		role, ok := c.Get("role")
		if !ok || !allowed[role.(models.UserRole)] {
			c.AbortWithStatusJSON(403, gin.H{"code": 2003, "message": "无权限"})
			return
		}
		c.Next()
	}
}
