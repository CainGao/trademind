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

// Agent 输入文本长度上限（防御 AI context 溢出 + DB 膨胀，同 gotcha #61）。
const (
	maxEmailSubjectLen = 1 * 1024  // 邮件主题最大 1KB
	maxEmailContentLen = 50 * 1024 // 邮件正文最大 50KB
	maxReviewsLen      = 50 * 1024 // 评论文本最大 50KB
)

// validListingPlatforms OptimizeListing 允许的平台枚举（同 gotcha #55 枚举校验）。
var validListingPlatforms = map[string]bool{
	"amazon":  true,
	"shopify": true,
	"tiktok":  true,
	"temu":    true,
}

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

// ============================================================================
// Week 6 新增：B2B 专用 Agent（邮件/询盘/报价）+ B2C 专用 Agent（上架/评论）
// ============================================================================

// EmailAnalysisReq 邮件分析请求体。
type EmailAnalysisReq struct {
	Subject string `json:"subject" binding:"required"`
	Content string `json:"content" binding:"required"`
}

// AnalyzeEmail POST /api/agents/analyze-email
func (h *AgentHandler) AnalyzeEmail(c *gin.Context) {
	var req EmailAnalysisReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误: "+err.Error())
		return
	}
	if len(req.Subject) > maxEmailSubjectLen {
		response.BadRequest(c, "邮件主题不能超过 1KB")
		return
	}
	if len(req.Content) > maxEmailContentLen {
		response.BadRequest(c, "邮件正文不能超过 50KB")
		return
	}
	provider := service.AIProvider(c.DefaultQuery("provider", ""))
	analysis, run, err := h.agentSvc.AnalyzeEmail(req.Subject, req.Content, provider, models.TriggerUser)
	if err != nil {
		response.InternalError(c, "邮件分析失败: "+err.Error())
		return
	}
	response.Success(c, gin.H{"analysis": analysis, "run": run})
}

// AnalyzeInquiry POST /api/agents/analyze-inquiry?inquiry_id=1
func (h *AgentHandler) AnalyzeInquiry(c *gin.Context) {
	inquiryID, err := strconv.ParseUint(c.Query("inquiry_id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "inquiry_id 必填")
		return
	}
	provider := service.AIProvider(c.DefaultQuery("provider", ""))
	analysis, run, err := h.agentSvc.AnalyzeInquiry(uint(inquiryID), provider, models.TriggerUser)
	if err != nil {
		response.InternalError(c, "询盘分析失败: "+err.Error())
		return
	}
	response.Success(c, gin.H{"analysis": analysis, "run": run})
}

// AdviseQuotation POST /api/agents/advise-quotation?inquiry_id=1&product_id=1
func (h *AgentHandler) AdviseQuotation(c *gin.Context) {
	inquiryID, err := strconv.ParseUint(c.Query("inquiry_id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "inquiry_id 必填")
		return
	}
	productID, err := strconv.ParseUint(c.Query("product_id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "product_id 必填")
		return
	}
	provider := service.AIProvider(c.DefaultQuery("provider", ""))
	advice, run, err := h.agentSvc.AdviseQuotation(uint(inquiryID), uint(productID), provider, models.TriggerUser)
	if err != nil {
		response.InternalError(c, "报价建议失败: "+err.Error())
		return
	}
	response.Success(c, gin.H{"advice": advice, "run": run})
}

// OptimizeListing POST /api/agents/optimize-listing?product_id=1&platform=amazon
func (h *AgentHandler) OptimizeListing(c *gin.Context) {
	productID, err := strconv.ParseUint(c.Query("product_id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "product_id 必填")
		return
	}
	platform := c.DefaultQuery("platform", "amazon")
	if !validListingPlatforms[platform] {
		response.BadRequest(c, "不支持的平台: " + platform + "（支持: amazon, shopify, tiktok, temu）")
		return
	}
	provider := service.AIProvider(c.DefaultQuery("provider", ""))
	opt, run, err := h.agentSvc.OptimizeListing(uint(productID), platform, provider, models.TriggerUser)
	if err != nil {
		response.InternalError(c, "上架优化失败: "+err.Error())
		return
	}
	response.Success(c, gin.H{"optimization": opt, "run": run})
}

// ReviewsAnalysisReq 评论分析请求体。
type ReviewsAnalysisReq struct {
	Reviews string `json:"reviews" binding:"required"`
}

// AnalyzeReviews POST /api/agents/analyze-reviews
func (h *AgentHandler) AnalyzeReviews(c *gin.Context) {
	var req ReviewsAnalysisReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误: "+err.Error())
		return
	}
	if len(req.Reviews) > maxReviewsLen {
		response.BadRequest(c, "评论内容不能超过 50KB")
		return
	}
	provider := service.AIProvider(c.DefaultQuery("provider", ""))
	analysis, run, err := h.agentSvc.AnalyzeReviews(req.Reviews, provider, models.TriggerUser)
	if err != nil {
		response.InternalError(c, "评论分析失败: "+err.Error())
		return
	}
	response.Success(c, gin.H{"analysis": analysis, "run": run})
}
