// Package handler — 认证 HTTP 处理器。
//
// 规范 V1.0 §3.1: RESTful 路由
// 规范 V1.0 §3.2: 统一响应格式
// 规范 V1.0 §3.5: binding 校验
// 规范 V1.0 §4.3: Handler 统一错误处理
package handler

import (
	"errors"

	"github.com/CainGao/trademind/internal/models"
	"github.com/CainGao/trademind/internal/pkg/response"
	"github.com/CainGao/trademind/internal/service"
	"github.com/gin-gonic/gin"
)

// AuthHandler 认证处理器。
type AuthHandler struct {
	svc *service.AuthService
}

// NewAuthHandler 创建处理器。
func NewAuthHandler(svc *service.AuthService) *AuthHandler {
	return &AuthHandler{svc: svc}
}

// RegisterRoutes 注册 /auth/* 路由（公开）。
func (h *AuthHandler) RegisterRoutes(r *gin.RouterGroup) {
	r.POST("/login", h.Login)
	r.POST("/register", h.Register)
	r.POST("/refresh", h.Refresh)
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
	pair, err := h.svc.Login(input, c.ClientIP())
	if err != nil {
		if errors.Is(err, service.ErrInvalidCredentials) {
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
			response.BadRequest(c, err.Error()) // TODO: 改 409 Conflict
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
