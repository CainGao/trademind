// Package handler — Agent 任务管理 HTTP 端点。
//
// 端点：
//   - POST /api/agents/run?type=selection|sourcing   手动触发 Agent
//   - GET  /api/agents/runs                          Agent 运行历史
//   - GET  /api/agents/runs/:id                      单次运行详情
//   - GET  /api/agents/schedule                      当前定时配置
//   - PUT  /api/agents/schedule                      更新定时
package handler

import (
	"strconv"

	"github.com/CainGao/trademind/internal/models"
	"github.com/CainGao/trademind/internal/pkg/response"
	"github.com/CainGao/trademind/internal/service"
	"github.com/gin-gonic/gin"
)

// AgentHandler Agent 相关 HTTP 端点。
type AgentHandler struct {
	agentSvc *service.AgentService
	schedSvc *service.SchedulerService
}

// NewAgentHandler 构造。
func NewAgentHandler(agentSvc *service.AgentService, schedSvc *service.SchedulerService) *AgentHandler {
	return &AgentHandler{agentSvc: agentSvc, schedSvc: schedSvc}
}

// Run 手动触发 Agent。
// POST /api/agents/run?type=selection&days=14&provider=deepseek
func (h *AgentHandler) Run(c *gin.Context) {
	agentType := c.DefaultQuery("type", "selection")
	provider := service.AIProvider(c.DefaultQuery("provider", ""))
	days, _ := strconv.Atoi(c.DefaultQuery("days", "14"))

	switch agentType {
	case "selection":
		report, run, err := h.agentSvc.RunSelection(days, provider, models.TriggerUser)
		if err != nil {
			response.InternalError(c, "选品 Agent 执行失败: "+err.Error())
			return
		}
		response.Success(c, gin.H{"report": report, "run": run})

	case "sourcing":
		report, run, err := h.agentSvc.RunSourcing(provider, models.TriggerUser)
		if err != nil {
			response.InternalError(c, "采购 Agent 执行失败: "+err.Error())
			return
		}
		response.Success(c, gin.H{"report": report, "run": run})

	default:
		response.BadRequest(c, "不支持的 agent 类型: "+agentType+"（支持: selection, sourcing）")
	}
}

// AnalyzeProduct 商品分析 Agent（沿用 Week 4 已有逻辑）。
// POST /api/agents/analyze-product?product_id=1
func (h *AgentHandler) AnalyzeProduct(c *gin.Context) {
	productID, err := strconv.ParseUint(c.Query("product_id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "product_id 必填")
		return
	}
	provider := service.AIProvider(c.DefaultQuery("provider", ""))
	analysis, err := h.agentSvc.AnalyzeProduct(uint(productID), provider)
	if err != nil {
		response.InternalError(c, "分析失败: "+err.Error())
		return
	}
	response.Success(c, analysis)
}

// RunsRequest 运行历史查询参数。
type RunsRequest struct {
	Page     int    `form:"page"`
	PageSize int    `form:"page_size"`
	Type     string `form:"type"`
}

// ListRuns Agent 运行历史。
// GET /api/agents/runs?page=1&page_size=20&type=selection
func (h *AgentHandler) ListRuns(c *gin.Context) {
	var req RunsRequest
	_ = c.ShouldBindQuery(&req)
	if req.Page < 1 {
		req.Page = 1
	}
	if req.PageSize < 1 || req.PageSize > 100 {
		req.PageSize = 20
	}

	items, total, err := h.agentSvc.ListRuns(req.Page, req.PageSize, req.Type)
	if err != nil {
		response.InternalError(c, "查询失败: "+err.Error())
		return
	}
	response.SuccessPage(c, items, total, req.Page, req.PageSize)
}

// GetRun 单次运行详情。
// GET /api/agents/runs/:id
func (h *AgentHandler) GetRun(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "id 非法")
		return
	}
	run, err := h.agentSvc.GetRun(uint(id))
	if err != nil {
		response.NotFound(c, "运行记录不存在")
		return
	}
	response.Success(c, run)
}

// GetSchedule 当前定时配置。
// GET /api/agents/schedule
func (h *AgentHandler) GetSchedule(c *gin.Context) {
	response.Success(c, h.schedSvc.AllSchedule())
}

// UpdateScheduleReq 更新定时请求体。
type UpdateScheduleReq struct {
	AgentType string `json:"agent_type" binding:"required"`
	Cron      string `json:"cron" binding:"required"`
}

// UpdateSchedule 更新定时。
// PUT /api/agents/schedule
func (h *AgentHandler) UpdateSchedule(c *gin.Context) {
	var req UpdateScheduleReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误: "+err.Error())
		return
	}
	if err := h.schedSvc.UpdateSchedule(models.AgentType(req.AgentType), req.Cron); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	response.Success(c, h.schedSvc.AllSchedule())
}
