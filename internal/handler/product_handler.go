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
	"fmt"
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

// 商品输入字段长度上限（defense-in-depth，防止超长文本写入 DB）。
const (
	maxProductName     = 200    // 匹配 DB schema size:200
	maxProductDesc     = 10000  // 10KB，商品描述
	maxProductURL      = 2048   // 标准 URL 最大长度
	maxProductImageURL = 10000  // 10KB，图片 URL JSON 数组
	maxProductPkgSpec  = 200    // 匹配 DB schema size:200
)

// validateProductInput 校验商品输入字段长度。
func validateProductInput(name, desc, sourceURL, imageUrls, pkgSpec string) error {
	if len(name) > maxProductName {
		return fmt.Errorf("商品名称过长（上限 %d 字符）", maxProductName)
	}
	if len(desc) > maxProductDesc {
		return fmt.Errorf("商品描述过长（上限 %d 字符）", maxProductDesc)
	}
	if len(sourceURL) > maxProductURL {
		return fmt.Errorf("来源 URL 过长（上限 %d 字符）", maxProductURL)
	}
	if len(imageUrls) > maxProductImageURL {
		return fmt.Errorf("图片 URL 列表过长（上限 %d 字符）", maxProductImageURL)
	}
	if len(pkgSpec) > maxProductPkgSpec {
		return fmt.Errorf("包装规格过长（上限 %d 字符）", maxProductPkgSpec)
	}
	return nil
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
	if err := validateProductInput(in.Name, in.Description, in.SourceURL, in.ImageURLs, in.PackageSpec); err != nil {
		response.BadRequest(c, err.Error())
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
	// 校验提供的字段长度（指针字段非 nil 时校验）
	if in.Name != nil {
		if err := validateProductInput(*in.Name, "", "", "", ""); err != nil {
			response.BadRequest(c, err.Error())
			return
		}
	}
	if in.Description != nil && len(*in.Description) > maxProductDesc {
		response.BadRequest(c, fmt.Sprintf("商品描述过长（上限 %d 字符）", maxProductDesc))
		return
	}
	if in.SourceURL != nil && len(*in.SourceURL) > maxProductURL {
		response.BadRequest(c, fmt.Sprintf("来源 URL 过长（上限 %d 字符）", maxProductURL))
		return
	}
	if in.ImageURLs != nil && len(*in.ImageURLs) > maxProductImageURL {
		response.BadRequest(c, fmt.Sprintf("图片 URL 列表过长（上限 %d 字符）", maxProductImageURL))
		return
	}
	if in.PackageSpec != nil && len(*in.PackageSpec) > maxProductPkgSpec {
		response.BadRequest(c, fmt.Sprintf("包装规格过长（上限 %d 字符）", maxProductPkgSpec))
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
