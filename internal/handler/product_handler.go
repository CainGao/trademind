// Package handler — 商品中心 HTTP 处理器。
//
// 规范 V1.0 §3.1: RESTful 路由
//   GET    /api/products          列表（分页 + 筛选）
//   POST   /api/products          创建（手动录入）
//   GET    /api/products/:id      详情
//   PUT    /api/products/:id      更新
//   DELETE /api/products/:id      删除（软删除）
//   GET    /api/products/categories 分类列表
package handler

import (
	"strconv"

	"github.com/CainGao/trademind/internal/pkg/pagination"
	"github.com/CainGao/trademind/internal/pkg/response"
	"github.com/CainGao/trademind/internal/service"
	"github.com/gin-gonic/gin"
)

type ProductHandler struct {
	svc *service.ProductService
}

func NewProductHandler(svc *service.ProductService) *ProductHandler {
	return &ProductHandler{svc: svc}
}

func (h *ProductHandler) RegisterRoutes(r *gin.RouterGroup) {
	r.GET("/products", h.List)
	r.POST("/products", h.Create)
	r.GET("/products/categories", h.Categories)
	r.GET("/products/:id", h.Get)
	r.PUT("/products/:id", h.Update)
	r.DELETE("/products/:id", h.Delete)
}

// Create 手动创建商品。
// POST /api/products
func (h *ProductHandler) Create(c *gin.Context) {
	var in service.CreateProductInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.BadRequest(c, "参数错误: "+err.Error())
		return
	}
	in.CreatedBy = c.MustGet("user_id").(uint)
	p, err := h.svc.Create(in)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	response.Created(c, p)
}

// Get 商品详情。
// GET /api/products/:id
func (h *ProductHandler) Get(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "无效的 ID")
		return
	}
	p, err := h.svc.GetByID(uint(id))
	if err != nil {
		response.NotFound(c, "商品不存在")
		return
	}
	response.Success(c, p)
}

// Update 更新商品。
// PUT /api/products/:id
func (h *ProductHandler) Update(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "无效的 ID")
		return
	}
	var in service.UpdateProductInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.BadRequest(c, "参数错误: "+err.Error())
		return
	}
	p, err := h.svc.Update(uint(id), in)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	response.Success(c, p)
}

// Delete 软删除商品。
// DELETE /api/products/:id
func (h *ProductHandler) Delete(c *gin.Context) {
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

// List 商品列表（分页 + 筛选）。
// GET /api/products?page=1&page_size=20&keyword=手机壳&category=配件&source=1688&sort_by=created_at&order=desc
func (h *ProductHandler) List(c *gin.Context) {
	q := service.ListQuery{
		Page:     atoiDefault(c.Query("page"), 1, 0),
		PageSize: atoiDefault(c.Query("page_size"), 20, pagination.MaxPageSize),
		Keyword:  c.Query("keyword"),
		Category: c.Query("category"),
		Source:   c.Query("source"),
		SortBy:   c.Query("sort_by"),
		Order:    c.Query("order"),
	}
	result, err := h.svc.List(q)
	if err != nil {
		response.InternalError(c, "查询失败")
		return
	}
	response.SuccessPage(c, result.Items, result.Total, q.Page, q.PageSize)
}

// Categories 全部分类。
// GET /api/products/categories
func (h *ProductHandler) Categories(c *gin.Context) {
	cats, err := h.svc.Categories()
	if err != nil {
		response.InternalError(c, "查询失败")
		return
	}
	response.Success(c, cats)
}

// atoiDefault parses an int from a query string, returning def on error/empty.
// max > 0 caps the value (use for page_size to prevent unbounded LIMIT).
// max == 0 means no cap (use for page number).
func atoiDefault(s string, def, max int) int {
	if s == "" {
		return def
	}
	n, err := strconv.Atoi(s)
	if err != nil || n < 1 {
		return def
	}
	if max > 0 && n > max {
		return max
	}
	return n
}
