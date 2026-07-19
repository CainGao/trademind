// Package handler — AI 网关 + Agent HTTP 处理器。
//
// 路由：
//   GET  /api/ai/providers                查询已配置的 provider 列表
//   POST /api/ai/chat                     通用对话（测试用）
//   POST /api/agent/analyze-product       商品分析 Agent（?product_id=1&provider=deepseek）
package handler

import (
	"strconv"

	"github.com/CainGao/trademind/internal/pkg/response"
	"github.com/CainGao/trademind/internal/service"
	"github.com/gin-gonic/gin"
)

type AIHandler struct {
	aiSvc    *service.AIService
	agentSvc *service.AgentService
}

func NewAIHandler(ai *service.AIService, agent *service.AgentService) *AIHandler {
	return &AIHandler{aiSvc: ai, agentSvc: agent}
}

func (h *AIHandler) RegisterRoutes(r *gin.RouterGroup) {
	r.GET("/ai/providers", h.Providers)
	r.POST("/ai/chat", h.Chat)

	// Agent 端点
	r.POST("/agent/analyze-product", h.AnalyzeProduct)
}

// Providers 已配置的 provider 列表（不含 Key）。
func (h *AIHandler) Providers(c *gin.Context) {
	response.Success(c, gin.H{"providers": h.aiSvc.ConfiguredProviders()})
}

// ChatInput chat 入参。
type ChatInput struct {
	Provider    string             `json:"provider,omitempty"`
	Model       string             `json:"model,omitempty"`
	Messages    []service.ChatMessage `json:"messages"`
	Temperature float64            `json:"temperature,omitempty"`
	MaxTokens   int                `json:"max_tokens,omitempty"`
}

// Chat 通用对话（测试用）。
func (h *AIHandler) Chat(c *gin.Context) {
	var in ChatInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.BadRequest(c, "参数错误: "+err.Error())
		return
	}
	if len(in.Messages) == 0 {
		response.BadRequest(c, "messages 不能为空")
		return
	}
	resp, err := h.aiSvc.Chat(service.ChatRequest{
		Provider:    service.AIProvider(in.Provider),
		Model:       in.Model,
		Messages:    in.Messages,
		Temperature: in.Temperature,
		MaxTokens:   in.MaxTokens,
	})
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	response.Success(c, resp)
}

// AnalyzeProduct 商品分析 Agent。
// POST /api/agent/analyze-product?product_id=1&provider=deepseek
func (h *AIHandler) AnalyzeProduct(c *gin.Context) {
	productID, err := strconv.ParseUint(c.Query("product_id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "product_id 参数错误")
		return
	}
	provider := service.AIProvider(c.DefaultQuery("provider", ""))
	result, err := h.agentSvc.AnalyzeProduct(uint(productID), provider)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	response.Success(c, result)
}
