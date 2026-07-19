// Package service — B2C 专用 Agent（上架优化 / 评论分析）。
//
// 这两个 Agent 服务于跨境电商场景：
//   - 上架 Agent：商品数据 → AI 生成 SEO 标题 / 五点描述 / 关键词 / 类目建议
//   - 评论 Agent：批量评论 → AI 情感分析 + 提取产品改进点 + 回复模板
//
// 数据来源：用户粘贴评论文本 / 从 products 表读取商品
package service

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/CainGao/trademind/internal/models"
)

// ============================================================================
// 上架优化 Agent
// ============================================================================

// ListingOptimization 上架优化结果。
type ListingOptimization struct {
	Title            string   `json:"title"`            // SEO 优化标题
	BulletPoints     []string `json:"bullet_points"`    // 五点描述
	Description      string   `json:"description"`      // 长描述
	SearchTerms      []string `json:"search_terms"`     // 后台搜索词
	BackendKeywords  []string `json:"backend_keywords"` // 类目建议
	CategorySuggest  string   `json:"category_suggest"` // 推荐类目
	EstimatedSales   *int     `json:"estimated_sales"`  // 预估月销量
	CompetitorPrice  *float64 `json:"competitor_price"` // 竞品价格
	OptimizationTips []string `json:"optimization_tips"`
	Summary          string   `json:"summary"`
}

// OptimizeListing 上架优化 Agent。
// 输入：product_id → 取商品数据 → 输出上架文案。
//
// POST /api/agents/optimize-listing?product_id=1&platform=amazon
func (s *AgentService) OptimizeListing(productID uint, platform string, provider AIProvider, triggeredBy models.TriggerType) (*ListingOptimization, *models.AgentRun, error) {
	product, err := s.productRepo.FindByID(productID)
	if err != nil {
		return nil, nil, fmt.Errorf("商品不存在: %w", err)
	}
	if platform == "" {
		platform = "amazon"
	}

	system := `你是跨境电商上架专家，精通 Amazon/Shopify/TikTok Shop/Temu 的 SEO 规则。
基于商品数据，生成上架文案。严格 JSON：
{
  "title": "SEO 优化标题（<200 字符，含核心关键词）",
  "bullet_points": ["卖点1", "卖点2", "卖点3", "卖点4", "卖点5"],
  "description": "长描述（HTML 或纯文本，<2000 字符）",
  "search_terms": ["后台搜索词1", "词2"],
  "backend_keywords": ["关键词1", "关键词2"],
  "category_suggest": "推荐类目",
  "estimated_sales": 300,
  "competitor_price": 19.99,
  "optimization_tips": ["优化建议1"],
  "summary": "一句话总结"
}
要求：
1. bullet_points 必须 5 条，每条 < 200 字符
2. 标题前置核心关键词
3. 所有文案必须英文（如果商品名是中文，翻译成英文）
4. 只输出 JSON。`

	user := fmt.Sprintf(`商品信息：
名称: %s
类目: %s
描述: %s
采购价: %s %s
售价: %s %s
重量: %s kg
目标平台: %s`,
		product.Name, product.Category, product.Description,
		product.PurchasePrice, product.PurchaseCurrency,
		product.B2CSellingPrice, product.B2CSellingCcy,
		product.WeightKG, platform)

	run := &models.AgentRun{
		AgentType:   models.AgentListing,
		TriggeredBy: triggeredBy,
		Input:       user,
		Status:      models.AgentRunRunning,
		StartedAt:   time.Now(),
	}
	_ = s.agentRepo.Create(run)

	resp, err := s.ai.Chat(ChatRequest{
		Provider: provider,
		Messages: []ChatMessage{
			{Role: "system", Content: system},
			{Role: "user", Content: user},
		},
		Temperature: 0.5, // 文案生成要一些创造性
	})
	now := time.Now()
	run.FinishedAt = &now
	if err != nil {
		run.Output = err.Error()
		run.Status = models.AgentRunFailed
		_ = s.agentRepo.Update(run)
		return nil, run, err
	}
	run.Output = resp.Content
	if resp.Usage != nil {
		tokens := resp.Usage.TotalTokens
		run.TokensUsed = &tokens
	}

	content2 := cleanJSON(resp.Content)
	var opt ListingOptimization
	if err := json.Unmarshal([]byte(content2), &opt); err != nil {
		run.Status = models.AgentRunDone
		_ = s.agentRepo.Update(run)
		return &ListingOptimization{Summary: resp.Content}, run, nil
	}
	run.Status = models.AgentRunDone
	_ = s.agentRepo.Update(run)
	return &opt, run, nil
}

// ============================================================================
// 评论分析 Agent
// ============================================================================

// ReviewAnalysis 评论分析结果。
type ReviewAnalysis struct {
	SentimentDistribution map[string]int `json:"sentiment_distribution"` // positive/neutral/negative 数量
	TopIssues          []string `json:"top_issues"`          // 最常见差评问题
	TopPraises         []string `json:"top_praises"`         // 最常见好评点
	ProductImprovements []string `json:"product_improvements"` // 产品改进建议
	ReplyTemplate4Negative string `json:"reply_template_4_negative"` // 差评回复模板
	ReplyTemplate4Positive string `json:"reply_template_4_positive"` // 好评回复模板
	OverallScore       *float64 `json:"overall_score"` // 综合评分 0-10
	RiskAlerts         []string `json:"risk_alerts"` // 风险预警（如大面积质量投诉）
	Summary            string   `json:"summary"`
}

// AnalyzeReviews 评论分析 Agent。
// 输入：评论文本（多条评论用 --- 分隔）→ 输出情感分析 + 改进建议。
//
// POST /api/agents/analyze-reviews
// Body: { "reviews": "评论1\n---\n评论2\n---\n评论3" }
func (s *AgentService) AnalyzeReviews(reviews string, provider AIProvider, triggeredBy models.TriggerType) (*ReviewAnalysis, *models.AgentRun, error) {
	system := `你是电商评论分析专家。分析用户提供的多条商品评论，输出严格 JSON：
{
  "sentiment_distribution": {"positive": 7, "neutral": 2, "negative": 3},
  "top_issues": ["最差评问题1", "问题2"],
  "top_praises": ["最好评点1", "好评点2"],
  "product_improvements": ["改进建议1", "建议2"],
  "reply_template_4_negative": "差评回复英文模板（150字内，官方语气）",
  "reply_template_4_positive": "好评回复英文模板（80字内）",
  "overall_score": 7.5,
  "risk_alerts": ["风险预警（如有）"],
  "summary": "一句话总结"
}
要求：
1. 如果评论里有 3 条以上同类型质量问题，加入 risk_alerts
2. reply_template 用英文
3. 只输出 JSON。`

	run := &models.AgentRun{
		AgentType:   models.AgentReview,
		TriggeredBy: triggeredBy,
		Input:       reviews,
		Status:      models.AgentRunRunning,
		StartedAt:   time.Now(),
	}
	_ = s.agentRepo.Create(run)

	// 截断超长评论
	reviewText := reviews
	if len(reviewText) > 8000 {
		reviewText = reviewText[:8000] + "\n...(truncated)"
	}

	resp, err := s.ai.Chat(ChatRequest{
		Provider: provider,
		Messages: []ChatMessage{
			{Role: "system", Content: system},
			{Role: "user", Content: "以下是商品评论（用 --- 分隔）：\n\n" + reviewText},
		},
		Temperature: 0.3,
	})
	now := time.Now()
	run.FinishedAt = &now
	if err != nil {
		run.Output = err.Error()
		run.Status = models.AgentRunFailed
		_ = s.agentRepo.Update(run)
		return nil, run, err
	}
	run.Output = resp.Content
	if resp.Usage != nil {
		tokens := resp.Usage.TotalTokens
		run.TokensUsed = &tokens
	}

	content2 := cleanJSON(resp.Content)
	var analysis ReviewAnalysis
	if err := json.Unmarshal([]byte(content2), &analysis); err != nil {
		run.Status = models.AgentRunDone
		_ = s.agentRepo.Update(run)
		return &ReviewAnalysis{Summary: resp.Content}, run, nil
	}
	run.Status = models.AgentRunDone
	_ = s.agentRepo.Update(run)
	return &analysis, run, nil
}
