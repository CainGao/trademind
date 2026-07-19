// Package handler — Chrome 插件对接 HTTP 处理器。
//
// 路由（全部需 JWT + 首启完成）：
//   POST /api/extension/collect           采集商品入库
//   POST /api/extension/behavior          上报单条行为
//   POST /api/extension/behavior/batch    批量上报行为
//   GET  /api/extension/status            连接状态（插件轮询）
//
// 鉴权策略：插件和前端共用同一套账号体系（/api/auth/login 拿 token），
// 携带 Authorization: Bearer <token> 访问。插件不需要单独的注册流程。
package handler

import (
	"github.com/CainGao/trademind/internal/models"
	"github.com/CainGao/trademind/internal/pkg/response"
	"github.com/CainGao/trademind/internal/service"
	"github.com/gin-gonic/gin"
)

type ExtensionHandler struct {
	svc *service.ExtensionService
}

func NewExtensionHandler(svc *service.ExtensionService) *ExtensionHandler {
	return &ExtensionHandler{svc: svc}
}

func (h *ExtensionHandler) RegisterRoutes(r *gin.RouterGroup) {
	r.POST("/extension/collect", h.Collect)
	r.POST("/extension/behavior", h.ReportBehavior)
	r.POST("/extension/behavior/batch", h.ReportBehaviorBatch)
	r.GET("/extension/status", h.Status)
}

// Collect 接收插件采集的商品。
// POST /api/extension/collect
func (h *ExtensionHandler) Collect(c *gin.Context) {
	var in service.CollectProductInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.BadRequest(c, "参数错误: "+err.Error())
		return
	}
	userID := c.MustGet("user_id").(uint)
	result, err := h.svc.CollectProduct(userID, in)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	if result.IsNewProduct {
		response.Created(c, result)
	} else {
		response.Success(c, result)
	}
}

// ReportBehavior 上报单条行为。
// POST /api/extension/behavior
func (h *ExtensionHandler) ReportBehavior(c *gin.Context) {
	var in service.BehaviorInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.BadRequest(c, "参数错误: "+err.Error())
		return
	}
	userID := c.MustGet("user_id").(uint)
	if err := h.svc.ReportBehavior(userID, in); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	response.Success(c, gin.H{"reported": true})
}

// ReportBehaviorBatch 批量上报行为。
// POST /api/extension/behavior/batch
//
//	{"events": [{...}, {...}]}
func (h *ExtensionHandler) ReportBehaviorBatch(c *gin.Context) {
	var body struct {
		Events []service.BehaviorInput `json:"events"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		response.BadRequest(c, "参数错误: "+err.Error())
		return
	}
	userID := c.MustGet("user_id").(uint)
	n, err := h.svc.ReportBehaviorBatch(userID, body.Events)
	if err != nil {
		response.InternalError(c, "批量写入失败")
		return
	}
	response.Success(c, gin.H{"reported": n})
}

// Status 连接状态检查。
// GET /api/extension/status
func (h *ExtensionHandler) Status(c *gin.Context) {
	username, _ := c.MustGet("username").(string)
	role, _ := c.MustGet("role").(models.UserRole)
	st := h.svc.Status(username, string(role))
	response.Success(c, st)
}
