// Package handler — 统计数据 HTTP 处理器（老板驾驶舱数据源）。
//
// 路由：
//   GET  /api/stats/dashboard          全量驾驶舱数据（一次拿全）
//   GET  /api/stats/behavior/trend     每日行为趋势（?days=14）
//   GET  /api/stats/behavior/keywords  Top 搜索词（?days=14&limit=10）
package handler

import (
	"strconv"

	"github.com/CainGao/trademind/internal/pkg/response"
	"github.com/CainGao/trademind/internal/service"
	"github.com/gin-gonic/gin"
)

type StatsHandler struct {
	svc *service.StatsService
}

func NewStatsHandler(svc *service.StatsService) *StatsHandler {
	return &StatsHandler{svc: svc}
}

func (h *StatsHandler) RegisterRoutes(r *gin.RouterGroup) {
	r.GET("/stats/dashboard", h.Dashboard)
	r.GET("/stats/behavior/trend", h.Trend)
	r.GET("/stats/behavior/keywords", h.Keywords)
}

// Dashboard 驾驶舱全量数据。
// GET /api/stats/dashboard?days=14
func (h *StatsHandler) Dashboard(c *gin.Context) {
	days, _ := strconv.Atoi(c.DefaultQuery("days", "14"))
	data, err := h.svc.Dashboard(days)
	if err != nil {
		response.InternalError(c, "查询失败")
		return
	}
	response.Success(c, data)
}

// Trend 每日趋势。
// GET /api/stats/behavior/trend?days=14
func (h *StatsHandler) Trend(c *gin.Context) {
	days, _ := strconv.Atoi(c.DefaultQuery("days", "14"))
	rows, err := h.svc.DailyTrend(days)
	if err != nil {
		response.InternalError(c, "查询失败")
		return
	}
	response.Success(c, rows)
}

// Keywords Top 搜索词。
// GET /api/stats/behavior/keywords?days=14&limit=10
func (h *StatsHandler) Keywords(c *gin.Context) {
	days, _ := strconv.Atoi(c.DefaultQuery("days", "14"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	rows, err := h.svc.TopKeywords(days, limit)
	if err != nil {
		response.InternalError(c, "查询失败")
		return
	}
	response.Success(c, rows)
}
