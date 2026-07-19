// Package service — Agent 服务：在 AI 网关之上封装业务 Agent。
//
// Agent 与原始 Chat 的区别：
//   - Chat 是通用对话接口
//   - Agent 是「带业务上下文 + 固定目标」的 AI 调用，例如：
//     * AnalyzeProduct：把商品数据喂给 AI → 生成选品建议/风险提示/定价参考
//     * AnalyzeInquiry：解析询盘文本 → 提取国家/需求/阶段/成交概率
//     * GenerateReply：邮件自动回复建议
//
// 架构文档 §1.2: Agent 体系是「行为数据资产化」的产出端。
package service

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/CainGao/trademind/internal/models"
	"github.com/CainGao/trademind/internal/repository"
	"github.com/shopspring/decimal"
)

// AgentService 业务 Agent。
type AgentService struct {
	ai          *AIService
	productRepo *repository.ProductRepo
}

func NewAgentService(ai *AIService, pr *repository.ProductRepo) *AgentService {
	return &AgentService{ai: ai, productRepo: pr}
}

// ProductAnalysis 商品 AI 分析结果（结构化）。
type ProductAnalysis struct {
	Score              decimal.Decimal       `json:"score"`                // 综合评分 0-10
	SourcingAdvice     string                `json:"sourcing_advice"`      // 选品建议
	RiskWarnings       []string              `json:"risk_warnings"`        // 风险提示
	PriceAnalysis      string                `json:"price_analysis"`       // 定价参考
	MarketOutlook      string                `json:"market_outlook"`       // 市场前景
	RecommendedActions []string              `json:"recommended_actions"`  // 建议动作
	TargetMarkets      []string              `json:"target_markets"`       // 推荐目标市场
	B2BSuitability     string                `json:"b2b_suitability"`      // B2B 适配度
	B2CSuitability     string                `json:"b2c_suitability"`      // B2C 适配度
	Provider           AIProvider            `json:"provider"`
}

// AnalyzeProduct 分析商品（自动取商品数据 + 喂给 AI + 回写 ai_score/ai_analysis）。
//
// POST /api/agent/analyze-product?product_id=1
func (s *AgentService) AnalyzeProduct(productID uint, provider AIProvider) (*ProductAnalysis, error) {
	// 1. 取商品
	p, err := s.productRepo.FindByID(productID)
	if err != nil {
		return nil, fmt.Errorf("商品不存在: %w", err)
	}

	// 2. 构造 prompt
	system := `你是资深外贸选品顾问和供应链专家，服务中国外贸/跨境电商公司。
分析给定商品数据，输出严格 JSON 格式（不要 markdown 代码块），字段：
{
  "score": 0-10 的数字（综合选品价值）,
  "sourcing_advice": "选品建议（100字内）",
  "risk_warnings": ["风险1", "风险2"],
  "price_analysis": "定价分析（100字内）",
  "market_outlook": "市场前景（100字内）",
  "recommended_actions": ["动作1", "动作2"],
  "target_markets": ["国家/地区1", "国家/地区2"],
  "b2b_suitability": "B2B 适配度评价（80字内）",
  "b2c_suitability": "B2C 适配度评价（80字内）"
}
只输出 JSON，不要任何额外文字。`

	user := buildProductPrompt(p)

	// 3. 调 AI
	resp, err := s.ai.Chat(ChatRequest{
		Provider: provider,
		Messages: []ChatMessage{
			{Role: "system", Content: system},
			{Role: "user", Content: user},
		},
		Temperature: 0.3, // 分析类任务低温保证稳定
	})
	if err != nil {
		return nil, err
	}

	// 4. 解析 JSON（容错：去掉可能的 markdown 代码块）
	content := cleanJSON(resp.Content)
	var analysis ProductAnalysis
	if err := json.Unmarshal([]byte(content), &analysis); err != nil {
		// 解析失败：把原始内容塞进 SourcingAdvice，不让用户看到 500
		return &ProductAnalysis{
			SourcingAdvice: resp.Content,
			Provider:       resp.Provider,
		}, nil
	}
	analysis.Provider = resp.Provider

	// 5. 回写到商品表（ai_score + ai_analysis）
	scoreStr := "0"
	if analysis.Score.GreaterThan(decimal.Zero) {
		scoreStr = analysis.Score.String()
	}
	_ = s.productRepo.DB.Model(p).Updates(map[string]interface{}{
		"ai_score":    scoreStr,
		"ai_analysis": content,
	})

	return &analysis, nil
}

// buildProductPrompt 把商品数据格式化成 prompt。
func buildProductPrompt(p *models.Product) string {
	var b strings.Builder
	b.WriteString("请分析以下商品：\n\n")
	b.WriteString(fmt.Sprintf("商品名称: %s\n", p.Name))
	if p.Category != "" {
		b.WriteString(fmt.Sprintf("分类: %s\n", p.Category))
	}
	if p.Description != "" {
		b.WriteString(fmt.Sprintf("描述: %s\n", p.Description))
	}
	if p.PurchasePrice.GreaterThan(decimal.Zero) {
		b.WriteString(fmt.Sprintf("采购价: %s %s\n", p.PurchasePrice, p.PurchaseCurrency))
	}
	if p.Source != "" {
		b.WriteString(fmt.Sprintf("来源: %s (ID: %s)\n", p.Source, p.SourceID))
	}
	if p.B2BMOQ != nil {
		b.WriteString(fmt.Sprintf("起订量: %d\n", *p.B2BMOQ))
	}
	if p.WeightKG.GreaterThan(decimal.Zero) {
		b.WriteString(fmt.Sprintf("重量: %s kg\n", p.WeightKG))
	}
	if p.PackageSpec != "" {
		b.WriteString(fmt.Sprintf("包装: %s\n", p.PackageSpec))
	}
	if p.Scenarios != "" {
		b.WriteString(fmt.Sprintf("目标场景: %s\n", p.Scenarios))
	}
	return b.String()
}

// cleanJSON 去掉 AI 返回的 markdown 代码块包裹。
func cleanJSON(s string) string {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "```json")
	s = strings.TrimPrefix(s, "```")
	s = strings.TrimSuffix(s, "```")
	return strings.TrimSpace(s)
}
