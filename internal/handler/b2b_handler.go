// Package handler — B2B 外贸 HTTP 处理器（客户 + 询盘 + 报价单）。
package handler

import (
	"strconv"

	"github.com/CainGao/trademind/internal/pkg/pagination"
	"github.com/CainGao/trademind/internal/pkg/response"
	"github.com/CainGao/trademind/internal/service"
	"github.com/gin-gonic/gin"
)

type B2BHandler struct {
	customerSvc  *service.CustomerService
	inquirySvc   *service.InquiryService
	quotationSvc *service.QuotationService
}

func NewB2BHandler(c *service.CustomerService, i *service.InquiryService, q *service.QuotationService) *B2BHandler {
	return &B2BHandler{customerSvc: c, inquirySvc: i, quotationSvc: q}
}

func (h *B2BHandler) RegisterRoutes(r *gin.RouterGroup) {
	// 客户
	r.GET("/customers", h.ListCustomers)
	r.POST("/customers", h.CreateCustomer)
	r.GET("/customers/:id", h.GetCustomer)
	r.PUT("/customers/:id", h.UpdateCustomer)
	r.DELETE("/customers/:id", h.DeleteCustomer)

	// 询盘
	r.GET("/inquiries", h.ListInquiries)
	r.POST("/inquiries", h.CreateInquiry)
	r.GET("/inquiries/:id", h.GetInquiry)
	r.DELETE("/inquiries/:id", h.DeleteInquiry)

	// 报价单
	r.GET("/quotations", h.ListQuotations)
	r.POST("/quotations", h.CreateQuotation)
	r.GET("/quotations/:id", h.GetQuotation)
	r.PUT("/quotations/:id/status", h.UpdateQuotationStatus)
	r.DELETE("/quotations/:id", h.DeleteQuotation)
}

// ===== 客户 =====

func (h *B2BHandler) ListCustomers(c *gin.Context) {
	q := service.CustomerListQuery{
		Page: atoiDefault(c.Query("page"), 1, 0),
		PageSize: atoiDefault(c.Query("page_size"), 20, pagination.MaxPageSize),
		Keyword: c.Query("keyword"),
		Country: c.Query("country"),
		Stage: c.Query("stage"),
	}
	res, err := h.customerSvc.List(q)
	if err != nil {
		response.InternalError(c, "查询失败")
		return
	}
	response.SuccessPage(c, res.Items, res.Total, q.Page, q.PageSize)
}

func (h *B2BHandler) CreateCustomer(c *gin.Context) {
	var in service.CreateCustomerInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.BadRequest(c, "参数错误: "+err.Error())
		return
	}
	in.CreatedBy = c.MustGet("user_id").(uint)
	cust, err := h.customerSvc.Create(in)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	response.Created(c, cust)
}

func (h *B2BHandler) GetCustomer(c *gin.Context) {
	id := parseID(c)
	if id == 0 { return }
	cust, err := h.customerSvc.GetByID(id)
	if err != nil {
		response.NotFound(c, "客户不存在")
		return
	}
	response.Success(c, cust)
}

func (h *B2BHandler) UpdateCustomer(c *gin.Context) {
	id := parseID(c)
	if id == 0 { return }
	var in service.UpdateCustomerInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.BadRequest(c, "参数错误: "+err.Error())
		return
	}
	cust, err := h.customerSvc.Update(id, in)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	response.Success(c, cust)
}

func (h *B2BHandler) DeleteCustomer(c *gin.Context) {
	id := parseID(c)
	if id == 0 { return }
	if err := h.customerSvc.Delete(id); err != nil {
		response.InternalError(c, "删除失败")
		return
	}
	response.Success(c, gin.H{"deleted": true})
}

// ===== 询盘 =====

func (h *B2BHandler) ListInquiries(c *gin.Context) {
	q := service.InquiryListQuery{
		Page: atoiDefault(c.Query("page"), 1, 0),
		PageSize: atoiDefault(c.Query("page_size"), 20, pagination.MaxPageSize),
		Source: c.Query("source"),
		Status: c.Query("status"),
	}
	res, err := h.inquirySvc.List(q)
	if err != nil {
		response.InternalError(c, "查询失败")
		return
	}
	response.SuccessPage(c, res.Items, res.Total, q.Page, q.PageSize)
}

func (h *B2BHandler) CreateInquiry(c *gin.Context) {
	var in service.CreateInquiryInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.BadRequest(c, "参数错误: "+err.Error())
		return
	}
	in.CreatedBy = c.MustGet("user_id").(uint)
	iq, err := h.inquirySvc.Create(in)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	response.Created(c, iq)
}

func (h *B2BHandler) GetInquiry(c *gin.Context) {
	id := parseID(c)
	if id == 0 { return }
	iq, err := h.inquirySvc.GetByID(id)
	if err != nil {
		response.NotFound(c, "询盘不存在")
		return
	}
	response.Success(c, iq)
}

func (h *B2BHandler) DeleteInquiry(c *gin.Context) {
	id := parseID(c)
	if id == 0 { return }
	if err := h.inquirySvc.Delete(id); err != nil {
		response.InternalError(c, "删除失败")
		return
	}
	response.Success(c, gin.H{"deleted": true})
}

// ===== 报价单 =====

func (h *B2BHandler) ListQuotations(c *gin.Context) {
	q := service.QuotationListQuery{
		Page: atoiDefault(c.Query("page"), 1, 0),
		PageSize: atoiDefault(c.Query("page_size"), 20, pagination.MaxPageSize),
		Status: c.Query("status"),
	}
	res, err := h.quotationSvc.List(q)
	if err != nil {
		response.InternalError(c, "查询失败")
		return
	}
	response.SuccessPage(c, res.Items, res.Total, q.Page, q.PageSize)
}

func (h *B2BHandler) CreateQuotation(c *gin.Context) {
	var in service.CreateQuotationInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.BadRequest(c, "参数错误: "+err.Error())
		return
	}
	in.CreatedBy = c.MustGet("user_id").(uint)
	q, err := h.quotationSvc.Create(in)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	response.Created(c, q)
}

func (h *B2BHandler) GetQuotation(c *gin.Context) {
	id := parseID(c)
	if id == 0 { return }
	q, err := h.quotationSvc.GetByID(id)
	if err != nil {
		response.NotFound(c, "报价单不存在")
		return
	}
	response.Success(c, q)
}

func (h *B2BHandler) UpdateQuotationStatus(c *gin.Context) {
	id := parseID(c)
	if id == 0 { return }
	var in service.UpdateQuotationStatusInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.BadRequest(c, "参数错误: "+err.Error())
		return
	}
	if err := h.quotationSvc.UpdateStatus(id, in); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	response.Success(c, gin.H{"updated": true})
}

func (h *B2BHandler) DeleteQuotation(c *gin.Context) {
	id := parseID(c)
	if id == 0 { return }
	if err := h.quotationSvc.Delete(id); err != nil {
		response.InternalError(c, "删除失败")
		return
	}
	response.Success(c, gin.H{"deleted": true})
}

// parseID 解析 URL 中的 id 参数，失败时已写入响应，返回 0 表示无效。
func parseID(c *gin.Context) uint {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "无效的 ID")
		return 0
	}
	return uint(id)
}
