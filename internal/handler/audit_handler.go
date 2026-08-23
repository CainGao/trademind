// Package handler — 审计日志 HTTP 处理器（管理员）。
//
// 路由：
//   GET /api/audit/logs  审计日志列表（分页 + user_id/action/日期筛选）
//
// 审计日志含登录 IP 等敏感信息，仅管理员可查（规范 §6.7）。
package handler

import (
	"strconv"
	"time"

	"github.com/CainGao/trademind/internal/pkg/pagination"
	"github.com/CainGao/trademind/internal/pkg/response"
	"github.com/CainGao/trademind/internal/repository"
	"github.com/CainGao/trademind/internal/service"
	"github.com/gin-gonic/gin"
)

// 输入长度上限（gotcha #61/#64 模式：外部输入必设上限）
const (
	maxAuditActionLen  = 50 // action 枚举值最长的也就 login_failed/export
	auditDateFormat    = "2006-01-02"
)

type AuditHandler struct {
	svc *service.AuditService
}

func NewAuditHandler(svc *service.AuditService) *AuditHandler {
	return &AuditHandler{svc: svc}
}

func (h *AuditHandler) RegisterRoutes(r *gin.RouterGroup) {
	audit := r.Group("/audit")
	audit.GET("/logs", h.List)
}

// List 审计日志列表。
// 查询参数：page/page_size（clamp 到安全范围）、user_id、action、
// start_date/end_date（YYYY-MM-DD，闭区间）。
func (h *AuditHandler) List(c *gin.Context) {
	page := atoiDefault(c.Query("page"), 1, 0)
	size := atoiDefault(c.Query("page_size"), 20, pagination.MaxPageSize)

	f := repository.AuditLogFilter{}

	if uidStr := c.Query("user_id"); uidStr != "" {
		uid, err := strconv.ParseUint(uidStr, 10, 64)
		if err != nil || uid == 0 {
			response.BadRequest(c, "user_id 必须为正整数")
			return
		}
		u := uint(uid)
		f.UserID = &u
	}

	if action := c.Query("action"); action != "" {
		if len(action) > maxAuditActionLen {
			response.BadRequest(c, "action 参数过长")
			return
		}
		f.Action = action
	}

	if ds := c.Query("start_date"); ds != "" {
		t, err := time.ParseInLocation(auditDateFormat, ds, time.Local)
		if err != nil {
			response.BadRequest(c, "start_date 格式应为 YYYY-MM-DD")
			return
		}
		f.StartAt = &t
	}
	if de := c.Query("end_date"); de != "" {
		t, err := time.ParseInLocation(auditDateFormat, de, time.Local)
		if err != nil {
			response.BadRequest(c, "end_date 格式应为 YYYY-MM-DD")
			return
		}
		// end_date 含当天：+24h 转为开区间上界
		end := t.Add(24 * time.Hour)
		f.EndAt = &end
	}

	res, err := h.svc.List(f, page, size)
	if err != nil {
		response.InternalError(c, "查询失败")
		return
	}
	response.SuccessPage(c, res.Items, res.Total, res.Page, res.Size)
}
