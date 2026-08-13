// Package handler — B2C 跨境电商 HTTP 端点。
//
// 端点：
//   - GET    /api/b2c/stores                店铺列表
//   - POST   /api/b2c/stores                创建店铺
//   - PUT    /api/b2c/stores/:id            更新店铺
//   - DELETE /api/b2c/stores/:id            删除店铺
//   - GET    /api/b2c/listings              上架列表
//   - POST   /api/b2c/listings              创建上架
//   - PUT    /api/b2c/listings/:id          更新上架
//   - DELETE /api/b2c/listings/:id          删除上架
//   - GET    /api/b2c/orders                订单列表
//   - POST   /api/b2c/orders                创建订单（手动补录）
//   - PUT    /api/b2c/orders/:id/status     更新订单状态
//   - GET    /api/b2c/overview              订单总览（驾驶舱用）
package handler

import (
	"strconv"
	"time"

	"github.com/CainGao/trademind/internal/models"
	"github.com/CainGao/trademind/internal/pkg/pagination"
	"github.com/CainGao/trademind/internal/pkg/response"
	"github.com/CainGao/trademind/internal/service"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// B2CHandler B2C 跨境电商端点。
type B2CHandler struct {
	svc *service.B2CService
}

// NewB2CHandler 构造。
func NewB2CHandler(svc *service.B2CService) *B2CHandler {
	return &B2CHandler{svc: svc}
}

// ========== 店铺 ==========

// ListStores GET /api/b2c/stores?platform=amazon
func (h *B2CHandler) ListStores(c *gin.Context) {
	page := atoiDefault(c.Query("page"), 1, 0)
	pageSize := atoiDefault(c.Query("page_size"), 20, pagination.MaxPageSize)
	platform := c.Query("platform")

	items, total, err := h.svc.ListStores(page, pageSize, platform)
	if err != nil {
		response.InternalError(c, "查询失败: "+err.Error())
		return
	}
	response.SuccessPage(c, items, total, page, pageSize)
}

// CreateStore POST /api/b2c/stores
func (h *B2CHandler) CreateStore(c *gin.Context) {
	var s models.Store
	if err := c.ShouldBindJSON(&s); err != nil {
		response.BadRequest(c, "参数错误: "+err.Error())
		return
	}
	// 防止 mass assignment：清除客户端可能设置的内部字段
	s.ID = 0
	s.CreatedAt = time.Time{}
	s.UpdatedAt = time.Time{}
	s.DeletedAt = gorm.DeletedAt{}
	s.CreatedBy = c.MustGet("user_id").(uint)
	if err := h.svc.CreateStore(&s); err != nil {
		response.InternalError(c, "创建失败: "+err.Error())
		return
	}
	response.Created(c, s)
}

// UpdateStore PUT /api/b2c/stores/:id
func (h *B2CHandler) UpdateStore(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "无效 ID")
		return
	}
	var s models.Store
	if err := c.ShouldBindJSON(&s); err != nil {
		response.BadRequest(c, "参数错误: "+err.Error())
		return
	}
	s.ID = uint(id)
	// 防止 mass assignment：清除客户端可能篡改的审计字段
	s.CreatedAt = time.Time{}
	s.DeletedAt = gorm.DeletedAt{}
	if err := h.svc.UpdateStore(&s); err != nil {
		response.InternalError(c, "更新失败: "+err.Error())
		return
	}
	response.Success(c, s)
}

// DeleteStore DELETE /api/b2c/stores/:id
func (h *B2CHandler) DeleteStore(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "无效 ID")
		return
	}
	if err := h.svc.DeleteStore(uint(id)); err != nil {
		response.InternalError(c, "删除失败: "+err.Error())
		return
	}
	response.Success(c, gin.H{"deleted": true})
}

// ========== 上架 ==========

// ListListings GET /api/b2c/listings?store_id=1&status=active
func (h *B2CHandler) ListListings(c *gin.Context) {
	page := atoiDefault(c.Query("page"), 1, 0)
	pageSize := atoiDefault(c.Query("page_size"), 20, pagination.MaxPageSize)
	storeID, _ := strconv.ParseUint(c.Query("store_id"), 10, 64)
	status := c.Query("status")

	items, total, err := h.svc.ListListings(page, pageSize, uint(storeID), status)
	if err != nil {
		response.InternalError(c, "查询失败: "+err.Error())
		return
	}
	response.SuccessPage(c, items, total, page, pageSize)
}

// CreateListing POST /api/b2c/listings
func (h *B2CHandler) CreateListing(c *gin.Context) {
	var l models.Listing
	if err := c.ShouldBindJSON(&l); err != nil {
		response.BadRequest(c, "参数错误: "+err.Error())
		return
	}
	// 防止 mass assignment：清除客户端可能设置的内部字段
	l.ID = 0
	l.CreatedAt = time.Time{}
	l.UpdatedAt = time.Time{}
	l.DeletedAt = gorm.DeletedAt{}
	l.CreatedBy = c.MustGet("user_id").(uint)
	if err := h.svc.CreateListing(&l); err != nil {
		response.InternalError(c, "创建失败: "+err.Error())
		return
	}
	response.Created(c, l)
}

// UpdateListing PUT /api/b2c/listings/:id
func (h *B2CHandler) UpdateListing(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "无效 ID")
		return
	}
	var l models.Listing
	if err := c.ShouldBindJSON(&l); err != nil {
		response.BadRequest(c, "参数错误: "+err.Error())
		return
	}
	l.ID = uint(id)
	// 防止 mass assignment：清除客户端可能篡改的审计字段
	l.CreatedAt = time.Time{}
	l.DeletedAt = gorm.DeletedAt{}
	if err := h.svc.UpdateListing(&l); err != nil {
		response.InternalError(c, "更新失败: "+err.Error())
		return
	}
	response.Success(c, l)
}

// DeleteListing DELETE /api/b2c/listings/:id
func (h *B2CHandler) DeleteListing(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "无效 ID")
		return
	}
	if err := h.svc.DeleteListing(uint(id)); err != nil {
		response.InternalError(c, "删除失败: "+err.Error())
		return
	}
	response.Success(c, gin.H{"deleted": true})
}

// ========== 订单 ==========

// ListOrders GET /api/b2c/orders?store_id=1&status=paid&country=US
func (h *B2CHandler) ListOrders(c *gin.Context) {
	page := atoiDefault(c.Query("page"), 1, 0)
	pageSize := atoiDefault(c.Query("page_size"), 20, pagination.MaxPageSize)
	storeID, _ := strconv.ParseUint(c.Query("store_id"), 10, 64)
	status := c.Query("status")
	country := c.Query("country")

	items, total, err := h.svc.ListOrders(page, pageSize, uint(storeID), status, country)
	if err != nil {
		response.InternalError(c, "查询失败: "+err.Error())
		return
	}
	response.SuccessPage(c, items, total, page, pageSize)
}

// CreateOrder POST /api/b2c/orders（手动补录）
func (h *B2CHandler) CreateOrder(c *gin.Context) {
	var o models.Order
	if err := c.ShouldBindJSON(&o); err != nil {
		response.BadRequest(c, "参数错误: "+err.Error())
		return
	}
	// 防止 mass assignment：清除客户端可能设置的内部字段
	o.ID = 0
	o.CreatedAt = time.Time{}
	o.UpdatedAt = time.Time{}
	o.DeletedAt = gorm.DeletedAt{}
	if err := h.svc.CreateOrder(&o); err != nil {
		response.InternalError(c, "创建失败: "+err.Error())
		return
	}
	response.Created(c, o)
}

// UpdateOrderStatus PUT /api/b2c/orders/:id/status
func (h *B2CHandler) UpdateOrderStatus(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "无效 ID")
		return
	}
	var body struct {
		Status models.OrderStatus `json:"status" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		response.BadRequest(c, "参数错误: "+err.Error())
		return
	}
	if err := h.svc.UpdateOrderStatus(uint(id), body.Status); err != nil {
		response.InternalError(c, "更新失败: "+err.Error())
		return
	}
	response.Success(c, gin.H{"updated": true})
}

// Overview GET /api/b2c/overview（订单总览，驾驶舱用）
func (h *B2CHandler) Overview(c *gin.Context) {
	ov, err := h.svc.OrderOverview()
	if err != nil {
		response.InternalError(c, "查询失败: "+err.Error())
		return
	}
	response.Success(c, ov)
}
