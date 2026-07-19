// Package handler — 首次启动向导 HTTP 处理器。
//
// 路由（规范 §3.1）：
//   GET  /api/setup/status         查询首启状态（公开，用于前端守卫）
//   POST /api/setup/company        保存企业信息（需管理员登录）
//   POST /api/setup/scenario       选择业务场景
//   POST /api/setup/ai-key         配置 AI Key
//   POST /api/setup/change-password 修改默认密码
//   POST /api/setup/complete       标记完成
//
// 安全：所有写操作要求管理员登录。status 可匿名查（让前端知道要不要跳 setup）。

package handler

import (
	"errors"

	"github.com/CainGao/trademind/internal/pkg/response"
	"github.com/CainGao/trademind/internal/service"
	"github.com/gin-gonic/gin"
)

// SetupHandler 首次启动向导处理器。
type SetupHandler struct {
	svc *service.SetupService
}

// NewSetupHandler 创建处理器。
func NewSetupHandler(svc *service.SetupService) *SetupHandler {
	return &SetupHandler{svc: svc}
}

// RegisterRoutes 在 /api/setup 注册路由（写操作，需 admin 登录）。
// 注：status 由调用方在公开组注册；此方法只注册写操作。
func (h *SetupHandler) RegisterRoutes(r *gin.RouterGroup) {
	r.POST("/setup/company", h.SaveCompany)
	r.POST("/setup/scenario", h.SelectScenario)
	r.POST("/setup/ai-key", h.SaveAIKeys)
	r.POST("/setup/change-password", h.ChangePassword)
	r.POST("/setup/complete", h.Complete)
}

// Status 查询首启状态。
// GET /api/setup/status
func (h *SetupHandler) Status(c *gin.Context) {
	status, err := h.svc.GetStatus()
	if err != nil {
		response.InternalError(c, "读取状态失败")
		return
	}
	response.Success(c, status)
}

// SaveCompany 保存企业信息。
// POST /api/setup/company
func (h *SetupHandler) SaveCompany(c *gin.Context) {
	var input service.SaveCompanyInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.BadRequest(c, "参数错误: "+err.Error())
		return
	}
	if err := h.svc.SaveCompany(input); err != nil {
		response.InternalError(c, "保存企业信息失败: "+err.Error())
		return
	}
	response.Success(c, gin.H{"ok": true})
}

// SelectScenario 选择业务场景。
// POST /api/setup/scenario
func (h *SetupHandler) SelectScenario(c *gin.Context) {
	var input service.SelectScenarioInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.BadRequest(c, "参数错误: "+err.Error())
		return
	}
	if err := h.svc.SelectScenario(input); err != nil {
		response.InternalError(c, "保存场景失败: "+err.Error())
		return
	}
	response.Success(c, gin.H{"ok": true, "scenario": input.Scenario})
}

// SaveAIKeys 配置 AI Key。
// POST /api/setup/ai-key
func (h *SetupHandler) SaveAIKeys(c *gin.Context) {
	var input service.AIKeyInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.BadRequest(c, "参数错误: "+err.Error())
		return
	}
	if err := h.svc.SaveAIKeys(input); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	response.Success(c, gin.H{"ok": true})
}

// ChangePassword 修改管理员密码。
// POST /api/setup/change-password
func (h *SetupHandler) ChangePassword(c *gin.Context) {
	var input service.ChangePasswordInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.BadRequest(c, "参数错误: "+err.Error())
		return
	}
	userID := c.MustGet("user_id").(uint)
	if err := h.svc.ChangePassword(userID, input); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	response.Success(c, gin.H{"ok": true})
}

// Complete 标记首启完成。
// POST /api/setup/complete
func (h *SetupHandler) Complete(c *gin.Context) {
	if err := h.svc.Complete(); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	response.Success(c, gin.H{"ok": true, "completed": true})
}

// isSetupRequired 辅助函数：判断响应是否因 setup 未完成。
// 供中间件调用，强制跳转 setup。
var ErrSetupRequired = errors.New("setup required")
