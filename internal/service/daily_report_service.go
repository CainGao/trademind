// Package service — 老板日报生成 + 飞书 webhook 推送。
//
// 设计依据：开发方案 V2 §5.4
//
// 流程：
//   1. 每天傍晚 18:00 由 SchedulerService 触发（也可手动触发）
//   2. 聚合当日数据：行为总数/采集商品/新增供应商/Top 搜索词/机会/风险
//   3. AI 生成口语化叙述 + Kill/Scale 决策建议
//   4. 写入 daily_reports 表
//   5. 如配置了飞书 webhook，自动推送（支持签名验证）
package service

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/CainGao/trademind/internal/models"
	"github.com/CainGao/trademind/internal/repository"
	"gorm.io/gorm"
)

// DailyReportService 日报服务。
type DailyReportService struct {
	db           *gorm.DB
	ai           *AIService
	stats        *StatsService
	productRepo  *repository.ProductRepo
	supplierRepo *repository.SupplierRepo
	behaviorRepo *repository.BehaviorRepo
	setting      *repository.SettingRepo
}

// NewDailyReportService 构造。
func NewDailyReportService(
	db *gorm.DB,
	ai *AIService,
	stats *StatsService,
	pr *repository.ProductRepo,
	sr *repository.SupplierRepo,
	br *repository.BehaviorRepo,
	setting *repository.SettingRepo,
) *DailyReportService {
	return &DailyReportService{
		db: db, ai: ai, stats: stats,
		productRepo: pr, supplierRepo: sr, behaviorRepo: br, setting: setting,
	}
}

// DailyReportData 日报的结构化数据（喂给 AI）。
type DailyReportData struct {
	Date            string                 `json:"date"`
	BehaviorTotal   int64                  `json:"behavior_total"`
	ProductsCollected int64                `json:"products_collected"`
	SuppliersAdded  int64                  `json:"suppliers_added"`
	TopKeywords     []map[string]interface{} `json:"top_keywords"`
	TopProducts     []string               `json:"top_products"` // 近期高 AI 评分商品
	SupplierOverview *repository.SupplierOverview `json:"supplier_overview"`
	AgentRuns       int64                  `json:"agent_runs"` // 当日 Agent 执行次数
}

// CollectTodayData 聚合今日数据。
func (s *DailyReportService) CollectTodayData() (*DailyReportData, error) {
	today := time.Now()
	start := time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, today.Location())
	end := start.Add(24 * time.Hour)

	data := &DailyReportData{
		Date: today.Format("2006-01-02"),
	}

	// 今日行为数
	if err := s.db.Table("behavior_events").
		Where("occurred_at >= ? AND occurred_at < ?", start, end).
		Count(&data.BehaviorTotal).Error; err != nil {
		return nil, err
	}

	// 今日新商品
	if err := s.db.Table("products").
		Where("created_at >= ? AND created_at < ?", start, end).
		Count(&data.ProductsCollected).Error; err != nil {
		return nil, err
	}

	// 今日新供应商
	if err := s.db.Table("suppliers").
		Where("created_at >= ? AND created_at < ?", start, end).
		Count(&data.SuppliersAdded).Error; err != nil {
		return nil, err
	}

	// 当日 Agent 执行次数
	if err := s.db.Table("agent_runs").
		Where("started_at >= ? AND started_at < ?", start, end).
		Count(&data.AgentRuns).Error; err != nil {
		return nil, err
	}

	// Top 搜索词（今日）
	kws, _ := s.behaviorRepo.TopKeywords(1, 10)
	data.TopKeywords = kws

	// 高评分商品（AI Score > 0，最多 5 个）
	var topProducts []models.Product
	s.db.Where("ai_score > 0").Order("ai_score DESC").Limit(5).Find(&topProducts)
	for _, p := range topProducts {
		data.TopProducts = append(data.TopProducts, fmt.Sprintf("%s（AI评分:%s）", p.Name, p.AIScore))
	}

	// 供应商概览
	supOv, _ := s.supplierRepo.Overview()
	data.SupplierOverview = supOv

	return data, nil
}

// Generate 生成今日日报（聚合数据 → AI 叙述 → 入库）。
//
// POST /api/daily-reports/generate
// cron: 每天 18:00
func (s *DailyReportService) Generate(provider AIProvider, triggeredBy models.TriggerType) (*models.DailyReport, error) {
	// 1. 聚合数据
	data, err := s.CollectTodayData()
	if err != nil {
		return nil, fmt.Errorf("聚合数据失败: %w", err)
	}

	// 2. 构造 AI prompt
	system := `你是外贸公司老板的 AI 助手。基于当日业务数据，生成一份简洁有力的日报。
严格 JSON（不要 markdown）：
{
  "narrative": "口语化叙述（200字内，像秘书在跟老板汇报）",
  "opportunities": ["今日发现的机会1", "机会2"],
  "kill_scale": [
    {"action": "Scale", "target": "建议放量的商品/方向", "reason": "原因"},
    {"action": "Kill", "target": "建议砍掉的方向", "reason": "原因"}
  ]
}
要求：
1. narrative 必须是中文，像真人说话，不要列表，要讲重点
2. 如果数据很少（今日 0 行为 0 商品），也要生成"今天没什么动作，建议关注..."这样的叙述
3. kill_scale 至少 1 条建议，可以是 Scale 或 Kill
4. 只输出 JSON。`

	userBytes, _ := json.Marshal(data)
	user := "今日数据：\n" + string(userBytes)

	// 3. 记录 Agent 运行
	now := time.Now()
	reportDate := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	// 4. 调 AI
	resp, err := s.ai.Chat(ChatRequest{
		Provider: provider,
		Messages: []ChatMessage{
			{Role: "system", Content: system},
			{Role: "user", Content: user},
		},
		Temperature: 0.5, // 叙述类要一些创造性
	})
	if err != nil {
		log.Printf("[daily-report] AI 调用失败: %v", err)
		// AI 失败也生成一份纯数据日报（不阻塞日报产出）
		summaryJSON, _ := json.Marshal(data)
		report := &models.DailyReport{
			ReportDate:  reportDate,
			Summary:     string(summaryJSON),
			AINarrative: "（AI 暂不可用，今日数据见 summary 字段）",
			Opportunities: "[]",
			KillScale:   "[]",
		}
		s.upsertReport(report)
		return report, nil
	}

	// 5. 解析 AI 输出
	content := cleanJSON(resp.Content)
	var aiResult struct {
		Narrative    string             `json:"narrative"`
		Opportunities []string          `json:"opportunities"`
		KillScale    []map[string]string `json:"kill_scale"`
	}
	if err := json.Unmarshal([]byte(content), &aiResult); err != nil {
		// 解析失败：原始文本当叙述
		aiResult.Narrative = resp.Content
	}

	summaryJSON, _ := json.Marshal(data)
	oppJSON, _ := json.Marshal(aiResult.Opportunities)
	ksJSON, _ := json.Marshal(aiResult.KillScale)

	report := &models.DailyReport{
		ReportDate:  reportDate,
		Summary:     string(summaryJSON),
		AINarrative: aiResult.Narrative,
		Opportunities: string(oppJSON),
		KillScale:   string(ksJSON),
	}
	s.upsertReport(report)

	// 6. 记录 Agent 运行（status=done）
	run := &models.AgentRun{
		AgentType:   models.AgentReport,
		TriggeredBy: triggeredBy,
		Input:       user,
		Output:      resp.Content,
		Status:      models.AgentRunDone,
		StartedAt:   now,
		FinishedAt:  &now,
	}
	s.db.Create(run)

	return report, nil
}

// upsertReport 按 report_date 去重（同一天重复生成会覆盖）。
func (s *DailyReportService) upsertReport(r *models.DailyReport) {
	var existing models.DailyReport
	err := s.db.Where("report_date = ?", r.ReportDate).First(&existing).Error
	if err == gorm.ErrRecordNotFound {
		s.db.Create(r)
	} else if err == nil {
		r.ID = existing.ID
		s.db.Save(r)
	}
}

// List 日报列表。
func (s *DailyReportService) List(page, pageSize int) ([]models.DailyReport, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	var items []models.DailyReport
	var total int64
	s.db.Model(&models.DailyReport{}).Count(&total)
	err := s.db.Order("report_date DESC").
		Offset((page - 1) * pageSize).Limit(pageSize).
		Find(&items).Error
	return items, total, err
}

// GetByID 按 ID 查询。
func (s *DailyReportService) GetByID(id uint) (*models.DailyReport, error) {
	var r models.DailyReport
	err := s.db.First(&r, id).Error
	return &r, err
}

// GetByDate 按日期查询。
func (s *DailyReportService) GetByDate(date time.Time) (*models.DailyReport, error) {
	var r models.DailyReport
	day := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location())
	err := s.db.Where("report_date = ?", day).First(&r).Error
	return &r, err
}

// GetSetting 代理 setting 读取（Handler 用）。
func (s *DailyReportService) GetSetting(key string) (*models.Setting, error) {
	return s.setting.Get(key)
}

// SetSetting 代理 setting 写入。
func (s *DailyReportService) SetSetting(key, value string) error {
	return s.setting.Set(key, value, false)
}

// ============================================================================
// 飞书 webhook 推送
// ============================================================================

// feishuWebhookPayload 飞书自定义机器人消息体（简化版，text 类型）。
type feishuWebhookPayload struct {
	MsgType string `json:"msg_type"`
	Content struct {
		Text string `json:"text"`
	} `json:"content"`
}

// FeishuSign 飞书 webhook 签名（启用安全设置时用）。
// 算法：sign = base64(hmac_sha256(timestamp + "\n" + secret))
func FeishuSign(timestamp int64, secret string) string {
	stringToSign := fmt.Sprintf("%d\n%s", timestamp, secret)
	h := hmac.New(sha256.New, []byte(stringToSign))
	h.Write([]byte{})
	return base64.StdEncoding.EncodeToString(h.Sum(nil))
}

// DeliverToFeishu 推送日报到飞书。
// webhook URL + secret 存在 settings 表（用户在设置页配置）。
func (s *DailyReportService) DeliverToFeishu(reportID uint) error {
	// 1. 取 webhook 配置
	webhookURL, err := s.setting.Get("feishu_webhook_url")
	if err != nil || webhookURL == nil || webhookURL.Value == "" {
		return fmt.Errorf("未配置飞书 webhook URL（在系统设置里配置）")
	}
	secretSetting, _ := s.setting.Get("feishu_webhook_secret")
	secret := ""
	if secretSetting != nil {
		secret = secretSetting.Value
	}

	// 2. 取日报
	report, err := s.GetByID(reportID)
	if err != nil {
		return fmt.Errorf("日报不存在: %w", err)
	}

	// 3. 构造消息文本
	text := formatReportForFeishu(report)

	// 4. 构造 payload
	payload := struct {
		MsgType string                 `json:"msg_type"`
		Content map[string]string      `json:"content"`
		Timestamp string               `json:"timestamp,omitempty"`
		Sign    string                 `json:"sign,omitempty"`
	}{
		MsgType: "text",
		Content: map[string]string{"text": text},
	}
	if secret != "" {
		ts := time.Now().Unix()
		payload.Timestamp = fmt.Sprintf("%d", ts)
		payload.Sign = FeishuSign(ts, secret)
	}
	body, _ := json.Marshal(payload)

	// 5. POST 到飞书
	req, err := http.NewRequest("POST", webhookURL.Value, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("飞书 webhook URL 无效: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("推送飞书失败: %w", err)
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != 200 {
		return fmt.Errorf("飞书返回 %d: %s", resp.StatusCode, string(respBody))
	}

	// 6. 检查飞书业务错误
	var feishuResp struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
	}
	if err := json.Unmarshal(respBody, &feishuResp); err == nil {
		if feishuResp.Code != 0 {
			return fmt.Errorf("飞书业务错误: code=%d msg=%s", feishuResp.Code, feishuResp.Msg)
		}
	}

	// 7. 标记已推送
	delivered := true
	s.db.Model(&models.DailyReport{}).Where("id = ?", reportID).
		Update("delivered_to_feishu", delivered)

	log.Printf("[daily-report] 日报 #%d 已推送到飞书", reportID)
	return nil
}

// formatReportForFeishu 把日报格式化成飞书消息文本。
func formatReportForFeishu(r *models.DailyReport) string {
	var b bytes.Buffer
	b.WriteString(fmt.Sprintf("📊 老板日报 %s\n", r.ReportDate.Format("2006-01-02")))
	b.WriteString(strings.Repeat("=", 30) + "\n\n")

	// AI 叙述
	b.WriteString("📝 今日汇报：\n")
	b.WriteString(r.AINarrative + "\n\n")

	// 机会
	var opportunities []string
	if r.Opportunities != "" {
		json.Unmarshal([]byte(r.Opportunities), &opportunities)
	}
	if len(opportunities) > 0 {
		b.WriteString("💡 发现机会：\n")
		for i, opp := range opportunities {
			b.WriteString(fmt.Sprintf("%d. %s\n", i+1, opp))
		}
		b.WriteString("\n")
	}

	// Kill/Scale 建议
	var killScale []map[string]string
	if r.KillScale != "" {
		json.Unmarshal([]byte(r.KillScale), &killScale)
	}
	if len(killScale) > 0 {
		b.WriteString("🎯 Kill/Scale 建议：\n")
		for _, ks := range killScale {
			emoji := "📈"
			if ks["action"] == "Kill" {
				emoji = "📉"
			}
			b.WriteString(fmt.Sprintf("%s %s: %s\n   原因: %s\n",
				emoji, ks["action"], ks["target"], ks["reason"]))
		}
	}

	b.WriteString("\n— TradeMind AI 自动生成")
	return b.String()
}
