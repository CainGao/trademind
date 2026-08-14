// Package handler — B2B 外贸 HTTP 处理器（客户 + 询盘 + 报价单）。
package handler

import (
	"fmt"
	"strconv"

	"github.com/CainGao/trademind/internal/pkg/pagination"
	"github.com/CainGao/trademind/internal/pkg/response"
	"github.com/CainGao/trademind/internal/service"
	"github.com/gin-gonic/gin"
)

// B2B 输入字段长度上限（对齐 models/b2b.go 的 schema size 与业务合理值）。
// 规则同 gotcha #61/#64/#69：所有接受用户文本输入的端点都必须设长度上限，
// 防止 MB 级粘贴写入 SQLite TEXT 列导致 DB 膨胀。
const (
	maxCustomerCompanyName = 200   // 匹配 DB schema size:200
	maxCustomerCountry     = 100   // size:100
	maxCustomerContact     = 100   // size:100
	maxCustomerEmail       = 200   // size:200
	maxCustomerPhone       = 50    // size:50
	maxCustomerWeChat      = 50    // size:50
	maxCustomerDemand      = 10000 // type:text，10KB 需求描述

	maxInquirySource      = 50    // alibaba|exhibition|email|website 等来源标识
	maxInquiryProductDesc = 10000 // type:text，10KB 询价产品描述
	maxInquiryDest        = 100   // size:100 目的港
	maxInquiryPrice       = 50    // decimal 字符串形式

	maxQuotationCurrency = 10     // size:3，留余量
	maxQuotationAmount   = 50     // decimal 字符串形式
	maxQuotationItems    = 100000 // type:text，100KB 报价明细 JSON
)

// checkLen 通用长度校验 helper：超长返回带字段名的错误。
func checkLen(label, v string, max int) error {
	if len(v) > max {
		return fmt.Errorf("%s过长（上限 %d 字符）", label, max)
	}
	return nil
}

// validateCustomerInput 校验客户创建输入字段长度。
func validateCustomerInput(in service.CreateCustomerInput) error {
	for _, c := range []struct {
		label string
		val   string
		max   int
	}{
		{"公司名称", in.CompanyName, maxCustomerCompanyName},
		{"国家", in.Country, maxCustomerCountry},
		{"联系人", in.ContactPerson, maxCustomerContact},
		{"邮箱", in.Email, maxCustomerEmail},
		{"电话", in.Phone, maxCustomerPhone},
		{"微信", in.WeChat, maxCustomerWeChat},
		{"需求描述", in.Demand, maxCustomerDemand},
	} {
		if err := checkLen(c.label, c.val, c.max); err != nil {
			return err
		}
	}
	return nil
}

// validateCustomerUpdateInput 校验客户更新输入（仅校验非 nil 指针字段）。
func validateCustomerUpdateInput(in service.UpdateCustomerInput) error {
	if in.CompanyName != nil {
		if err := checkLen("公司名称", *in.CompanyName, maxCustomerCompanyName); err != nil {
			return err
		}
	}
	if in.Country != nil {
		if err := checkLen("国家", *in.Country, maxCustomerCountry); err != nil {
			return err
		}
	}
	if in.ContactPerson != nil {
		if err := checkLen("联系人", *in.ContactPerson, maxCustomerContact); err != nil {
			return err
		}
	}
	if in.Email != nil {
		if err := checkLen("邮箱", *in.Email, maxCustomerEmail); err != nil {
			return err
		}
	}
	if in.Phone != nil {
		if err := checkLen("电话", *in.Phone, maxCustomerPhone); err != nil {
			return err
		}
	}
	if in.WeChat != nil {
		if err := checkLen("微信", *in.WeChat, maxCustomerWeChat); err != nil {
			return err
		}
	}
	if in.Demand != nil {
		if err := checkLen("需求描述", *in.Demand, maxCustomerDemand); err != nil {
			return err
		}
	}
	return nil
}

// validateInquiryInput 校验询盘创建输入字段长度。
func validateInquiryInput(in service.CreateInquiryInput) error {
	for _, c := range []struct {
		label string
		val   string
		max   int
	}{
		{"询盘来源", in.Source, maxInquirySource},
		{"产品描述", in.ProductDesc, maxInquiryProductDesc},
		{"目的港", in.Destination, maxInquiryDest},
		{"目标价", in.TargetPrice, maxInquiryPrice},
	} {
		if err := checkLen(c.label, c.val, c.max); err != nil {
			return err
		}
	}
	return nil
}

// validateQuotationInput 校验报价单创建输入字段长度。
func validateQuotationInput(in service.CreateQuotationInput) error {
	for _, c := range []struct {
		label string
		val   string
		max   int
	}{
		{"币种", in.Currency, maxQuotationCurrency},
		{"总金额", in.TotalAmount, maxQuotationAmount},
		{"报价明细", in.Items, maxQuotationItems},
	} {
		if err := checkLen(c.label, c.val, c.max); err != nil {
			return err
		}
	}
	return nil
}

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
	if err := validateCustomerInput(in); err != nil {
		response.BadRequest(c, err.Error())
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
	if err := validateCustomerUpdateInput(in); err != nil {
		response.BadRequest(c, err.Error())
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
	if err := validateInquiryInput(in); err != nil {
		response.BadRequest(c, err.Error())
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
	if err := validateQuotationInput(in); err != nil {
		response.BadRequest(c, err.Error())
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
