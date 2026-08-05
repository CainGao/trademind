// Package handler — 供应商 HTTP 处理器。
//
// 路由：
//   GET    /api/suppliers            列表（分页 + 筛选）
//   GET    /api/suppliers/overview   总览统计（驾驶舱用）
//   GET    /api/suppliers/:id        详情（含商品数）
//   GET    /api/suppliers/:id/products  关联商品
//   PUT    /api/suppliers/:id/risk   更新风险/AI 评分
//   DELETE /api/suppliers/:id        软删除
package handler

import (
	"strconv"

	"github.com/CainGao/trademind/internal/pkg/pagination"
	"github.com/CainGao/trademind/internal/pkg/response"
	"github.com/CainGao/trademind/internal/service"
	"github.com/gin-gonic/gin"
)

type SupplierHandler struct {
	svc *service.SupplierService
}

func NewSupplierHandler(svc *service.SupplierService) *SupplierHandler {
	return &SupplierHandler{svc: svc}
}

func (h *SupplierHandler) RegisterRoutes(r *gin.RouterGroup) {
	r.GET("/suppliers", h.List)
	r.GET("/suppliers/overview", h.Overview)
	r.GET("/suppliers/:id", h.Get)
	r.GET("/suppliers/:id/products", h.Products)
	r.PUT("/suppliers/:id/risk", h.UpdateRisk)
	r.DELETE("/suppliers/:id", h.Delete)
}

func (h *SupplierHandler) List(c *gin.Context) {
	q := service.SupplierListQuery{
		Page:      atoiDefault(c.Query("page"), 1, 0),
		PageSize:  atoiDefault(c.Query("page_size"), 20, pagination.MaxPageSize),
		Keyword:   c.Query("keyword"),
		Source:    c.Query("source"),
		RiskLevel: c.Query("risk_level"),
	}
	res, err := h.svc.List(q)
	if err != nil {
		response.InternalError(c, "查询失败")
		return
	}
	response.SuccessPage(c, res.Items, res.Total, q.Page, q.PageSize)
}

func (h *SupplierHandler) Overview(c *gin.Context) {
	ov, err := h.svc.Overview()
	if err != nil {
		response.InternalError(c, "查询失败")
		return
	}
	response.Success(c, ov)
}

func (h *SupplierHandler) Get(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "无效的 ID")
		return
	}
	d, err := h.svc.GetByID(uint(id))
	if err != nil {
		response.NotFound(c, "供应商不存在")
		return
	}
	response.Success(c, d)
}

func (h *SupplierHandler) Products(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "无效的 ID")
		return
	}
	page := atoiDefault(c.Query("page"), 1, 0)
	pageSize := atoiDefault(c.Query("page_size"), 20, pagination.MaxPageSize)
	items, total, err := h.svc.Products(uint(id), page, pageSize)
	if err != nil {
		response.InternalError(c, "查询失败")
		return
	}
	response.SuccessPage(c, items, total, page, pageSize)
}

func (h *SupplierHandler) UpdateRisk(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "无效的 ID")
		return
	}
	var in service.UpdateRiskInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.BadRequest(c, "参数错误: "+err.Error())
		return
	}
	if err := h.svc.UpdateRisk(uint(id), in); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	response.Success(c, gin.H{"updated": true})
}

func (h *SupplierHandler) Delete(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "无效的 ID")
		return
	}
	if err := h.svc.Delete(uint(id)); err != nil {
		response.InternalError(c, "删除失败")
		return
	}
	response.Success(c, gin.H{"deleted": true})
}
