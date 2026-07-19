// Package handler — 老板日报 HTTP 端点。
//
// 端点：
//   - POST /api/daily-reports/generate          手动生成今日日报
//   - GET  /api/daily-reports                   日报列表
//   - GET  /api/daily-reports/:id               单条详情
//   - GET  /api/daily-reports/today             今日日报（如有）
//   - POST /api/daily-reports/:id/deliver-feishu 推送到飞书
//   - GET  /api/daily-reports/feishu-config     飞书 webhook 配置
//   - PUT  /api/daily-reports/feishu-config     更新飞书 webhook 配置
package handler

import (
	"strconv"
	"time"

	"github.com/CainGao/trademind/internal/models"
	"github.com/CainGao/trademind/internal/pkg/response"
	"github.com/CainGao/trademind/internal/service"
	"github.com/gin-gonic/gin"
)

// DailyReportHandler 日报端点。
type DailyReportHandler struct {
	svc *service.DailyReportService
}

// NewDailyReportHandler 构造。
func NewDailyReportHandler(svc *service.DailyReportService) *DailyReportHandler {
	return &DailyReportHandler{svc: svc}
}

// Generate POST /api/daily-reports/generate
func (h *DailyReportHandler) Generate(c *gin.Context) {
	provider := service.AIProvider(c.DefaultQuery("provider", ""))
	report, err := h.svc.Generate(provider, models.TriggerUser)
	if err != nil {
		response.InternalError(c, "日报生成失败: "+err.Error())
		return
	}
	// 生成后自动推送（如已配置 webhook）
	if c.Query("auto_deliver") == "true" {
		if err := h.svc.DeliverToFeishu(report.ID); err != nil {
			// 推送失败不阻塞，返回成功 + warning
			response.Success(c, gin.H{
				"report":           report,
				"deliver_warning":  err.Error(),
			})
			return
		}
	}
	response.Success(c, report)
}

// List GET /api/daily-reports?page=1&page_size=20
func (h *DailyReportHandler) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	items, total, err := h.svc.List(page, pageSize)
	if err != nil {
		response.InternalError(c, "查询失败: "+err.Error())
		return
	}
	response.SuccessPage(c, items, total, page, pageSize)
}

// GetByID GET /api/daily-reports/:id
func (h *DailyReportHandler) GetByID(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "id 非法")
		return
	}
	report, err := h.svc.GetByID(uint(id))
	if err != nil {
		response.NotFound(c, "日报不存在")
		return
	}
	response.Success(c, report)
}

// GetToday GET /api/daily-reports/today
func (h *DailyReportHandler) GetToday(c *gin.Context) {
	report, err := h.svc.GetByDate(time.Now())
	if err != nil {
		response.Success(c, nil) // 今日还没生成，返回 null
		return
	}
	response.Success(c, report)
}

// DeliverToFeishu POST /api/daily-reports/:id/deliver-feishu
func (h *DailyReportHandler) DeliverToFeishu(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "id 非法")
		return
	}
	if err := h.svc.DeliverToFeishu(uint(id)); err != nil {
		response.InternalError(c, "推送失败: "+err.Error())
		return
	}
	response.Success(c, gin.H{"delivered": true})
}

// FeishuConfig 飞书 webhook 配置。
type FeishuConfig struct {
	WebhookURL string `json:"webhook_url"`
	Secret     string `json:"secret,omitempty"` // 只在写入时传，读取时不返回
}

// GetFeishuConfig GET /api/daily-reports/feishu-config
func (h *DailyReportHandler) GetFeishuConfig(c *gin.Context) {
	cfg := FeishuConfig{}
	if s, err := h.svc.GetSetting("feishu_webhook_url"); err == nil && s != nil {
		cfg.WebhookURL = s.Value
	}
	// secret 不返回（敏感）
	response.Success(c, cfg)
}

// UpdateFeishuConfig PUT /api/daily-reports/feishu-config
func (h *DailyReportHandler) UpdateFeishuConfig(c *gin.Context) {
	var req FeishuConfig
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误: "+err.Error())
		return
	}
	if err := h.svc.SetSetting("feishu_webhook_url", req.WebhookURL); err != nil {
		response.InternalError(c, "保存失败: "+err.Error())
		return
	}
	if req.Secret != "" {
		if err := h.svc.SetSetting("feishu_webhook_secret", req.Secret); err != nil {
			response.InternalError(c, "保存 secret 失败: "+err.Error())
			return
		}
	}
	response.Success(c, gin.H{"updated": true})
}
