// Package handler — 认证 HTTP 处理器。
//
// 规范 V1.0 §3.1: RESTful 路由
// 规范 V1.0 §3.2: 统一响应格式
// 规范 V1.0 §3.5: binding 校验
// 规范 V1.0 §4.3: Handler 统一错误处理
package handler

import (
	"errors"
	"fmt"

	"github.com/CainGao/trademind/internal/middleware"
	"github.com/CainGao/trademind/internal/models"
	"github.com/CainGao/trademind/internal/pkg/response"
	"github.com/CainGao/trademind/internal/service"
	"github.com/gin-gonic/gin"
)

// AuthHandler 认证处理器。
type AuthHandler struct {
	svc     *service.AuthService
	limiter *middleware.LoginLimiter
}

// NewAuthHandler 创建处理器。
func NewAuthHandler(svc *service.AuthService, limiter *middleware.LoginLimiter) *AuthHandler {
	return &AuthHandler{svc: svc, limiter: limiter}
}

// RegisterRoutes 注册公开认证路由（仅 login/refresh）。
// ⚠️ register 不在此处：未认证请求绝不能创建账号（尤其 admin），见 RegisterAdminRoutes。
func (h *AuthHandler) RegisterRoutes(r *gin.RouterGroup) {
	r.POST("/login", h.Login)
	r.POST("/refresh", h.Refresh)
}

// RegisterAdminRoutes 注册受保护的用户管理路由。
// 调用方必须挂在 JWT + RequireRole(RoleAdmin) 中间件之后（router.go 装配）。
// POST /api/auth/register（gotcha #82：原公开注册 + 可指定 role=admin = 越权漏洞）
func (h *AuthHandler) RegisterAdminRoutes(r *gin.RouterGroup) {
	r.POST("/register", h.Register)
}

// Login 登录。
// POST /api/auth/login
//
//	{"username":"admin","password":"xxx"}
//	→ {"code":0,"data":{"access_token":"...","refresh_token":"...","user":{...}}}
func (h *AuthHandler) Login(c *gin.Context) {
	var input service.LoginInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.BadRequest(c, "参数错误: "+err.Error())
		return
	}

	// 防暴力破解：检查是否被限流
	if !h.limiter.Allow(input.Username) {
		remaining := h.limiter.LockoutRemaining(input.Username)
		mins := int(remaining.Minutes())
		if mins < 1 {
			mins = 1
		}
		c.Header("Retry-After", fmt.Sprintf("%d", int(remaining.Seconds())))
		response.TooManyRequests(c, fmt.Sprintf("登录尝试过多，账号已被临时锁定，请 %d 分钟后再试", mins))
		return
	}

	pair, err := h.svc.Login(input, c.ClientIP())
	if err != nil {
		if errors.Is(err, service.ErrInvalidCredentials) {
			h.limiter.RecordFailure(input.Username) // 记录失败
			response.Unauthorized(c, err.Error())
			return
		}
		if errors.Is(err, service.ErrUserInactive) {
			response.Forbidden(c, err.Error())
			return
		}
		response.InternalError(c, "登录失败")
		return
	}

	// 登录成功 → 清除失败计数
	h.limiter.RecordSuccess(input.Username)
	response.Success(c, pair)
}

// Register 注册新用户（首次启动向导或管理员后台调用）。
// POST /api/auth/register
//
//	{"username":"alice","password":"xxx","nickname":"小美","role":"staff"}
func (h *AuthHandler) Register(c *gin.Context) {
	var input service.RegisterInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.BadRequest(c, "参数错误: "+err.Error())
		return
	}
	user, err := h.svc.Register(input)
	if err != nil {
		if errors.Is(err, service.ErrUsernameTaken) {
			response.Conflict(c, err.Error())
			return
		}
		if errors.Is(err, service.ErrWeakPassword) {
			response.BadRequest(c, err.Error())
			return
		}
		response.InternalError(c, "注册失败")
		return
	}
	response.Created(c, user)
}

// Refresh 刷新 Token。
// POST /api/auth/refresh
//
//	{"refresh_token":"..."}
//	→ 新的 access_token + refresh_token
func (h *AuthHandler) Refresh(c *gin.Context) {
	var input service.RefreshInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.BadRequest(c, "参数错误: "+err.Error())
		return
	}
	pair, err := h.svc.Refresh(input)
	if err != nil {
		response.Unauthorized(c, err.Error())
		return
	}
	response.Success(c, pair)
}

// Me 当前用户信息（需 JWT）。
// GET /api/auth/me
func (h *AuthHandler) Me(c *gin.Context) {
	userID := c.MustGet("user_id").(uint)
	username := c.MustGet("username").(string)
	role := c.MustGet("role").(models.UserRole)
	response.Success(c, gin.H{
		"user_id":  userID,
		"username": username,
		"role":     role,
	})
}
