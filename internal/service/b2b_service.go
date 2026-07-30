// Package service — B2B 外贸业务（客户 + 询盘 + 报价单）。
package service

import (
	"errors"
	"strings"
	"time"

	"github.com/CainGao/trademind/internal/models"
	"github.com/CainGao/trademind/internal/repository"
	"github.com/shopspring/decimal"
)

// ============ 客户 ============

type CustomerService struct {
	repo *repository.CustomerRepo
}

func NewCustomerService(r *repository.CustomerRepo) *CustomerService {
	return &CustomerService{repo: r}
}

type CreateCustomerInput struct {
	CompanyName   string `json:"company_name"`
	Country       string `json:"country"`
	ContactPerson string `json:"contact_person"`
	Email         string `json:"email"`
	Phone         string `json:"phone"`
	WeChat        string `json:"wechat"`
	Demand        string `json:"demand"`
	Stage         string `json:"stage"`
	CreatedBy     uint   `json:"-"`
}

func (s *CustomerService) Create(in CreateCustomerInput) (*models.Customer, error) {
	if strings.TrimSpace(in.CompanyName) == "" {
		return nil, errors.New("公司名称不能为空")
	}
	stage := models.CustomerStageLead
	if in.Stage != "" {
		stage = models.CustomerStage(in.Stage)
	}
	c := &models.Customer{
		CompanyName:   in.CompanyName,
		Country:       in.Country,
		ContactPerson: in.ContactPerson,
		Email:         in.Email,
		Phone:         in.Phone,
		WeChat:        in.WeChat,
		Demand:        in.Demand,
		Stage:         stage,
	}
	c.CreatedBy = in.CreatedBy
	if err := s.repo.Create(c); err != nil {
		return nil, err
	}
	return c, nil
}

type CustomerListQuery struct {
	Page     int
	PageSize int
	Keyword  string
	Country  string
	Stage    string
}

func (s *CustomerService) List(q CustomerListQuery) (*repository.CustomerListResult, error) {
	return s.repo.List(repository.CustomerListParams{
		Page: q.Page, PageSize: q.PageSize,
		Keyword: q.Keyword, Country: q.Country, Stage: q.Stage,
	})
}

func (s *CustomerService) GetByID(id uint) (*models.Customer, error) {
	return s.repo.FindByID(id)
}

type UpdateCustomerInput struct {
	CompanyName   *string `json:"company_name,omitempty"`
	Country       *string `json:"country,omitempty"`
	ContactPerson *string `json:"contact_person,omitempty"`
	Email         *string `json:"email,omitempty"`
	Phone         *string `json:"phone,omitempty"`
	WeChat        *string `json:"wechat,omitempty"`
	Demand        *string `json:"demand,omitempty"`
	Stage         *string `json:"stage,omitempty"`
}

func (s *CustomerService) Update(id uint, in UpdateCustomerInput) (*models.Customer, error) {
	c, err := s.repo.FindByID(id)
	if err != nil {
		return nil, err
	}
	if in.CompanyName != nil { c.CompanyName = *in.CompanyName }
	if in.Country != nil { c.Country = *in.Country }
	if in.ContactPerson != nil { c.ContactPerson = *in.ContactPerson }
	if in.Email != nil { c.Email = *in.Email }
	if in.Phone != nil { c.Phone = *in.Phone }
	if in.WeChat != nil { c.WeChat = *in.WeChat }
	if in.Demand != nil { c.Demand = *in.Demand }
	if in.Stage != nil { c.Stage = models.CustomerStage(*in.Stage) }
	if err := s.repo.Update(c); err != nil {
		return nil, err
	}
	return c, nil
}

func (s *CustomerService) Delete(id uint) error {
	return s.repo.SoftDelete(id)
}

// ============ 询盘 ============

type InquiryService struct {
	repo *repository.InquiryRepo
}

func NewInquiryService(r *repository.InquiryRepo) *InquiryService {
	return &InquiryService{repo: r}
}

type CreateInquiryInput struct {
	CustomerID  *uint   `json:"customer_id,omitempty"`
	Source      string  `json:"source"`
	ProductDesc string  `json:"product_desc"`
	Quantity    *int    `json:"quantity,omitempty"`
	TargetPrice string  `json:"target_price,omitempty"`
	Destination string  `json:"destination"`
	Status      string  `json:"status"`
	CreatedBy   uint    `json:"-"`
}

func (s *InquiryService) Create(in CreateInquiryInput) (*models.Inquiry, error) {
	if strings.TrimSpace(in.ProductDesc) == "" {
		return nil, errors.New("询价产品描述不能为空")
	}
	status := "new"
	if in.Status != "" {
		status = in.Status
	}
	var price decimal.Decimal
	if in.TargetPrice != "" {
		v, err := decimal.NewFromString(in.TargetPrice)
		if err != nil {
			return nil, errors.New("目标价格式无效：" + in.TargetPrice)
		}
		price = v
	}
	i := &models.Inquiry{
		CustomerID:  in.CustomerID,
		Source:      in.Source,
		ProductDesc: in.ProductDesc,
		Quantity:    in.Quantity,
		TargetPrice: price,
		Destination: in.Destination,
		Status:      status,
	}
	i.CreatedBy = in.CreatedBy
	if err := s.repo.Create(i); err != nil {
		return nil, err
	}
	return i, nil
}

type InquiryListQuery struct {
	Page     int
	PageSize int
	Source   string
	Status   string
}

func (s *InquiryService) List(q InquiryListQuery) (*repository.InquiryListResult, error) {
	return s.repo.List(repository.InquiryListParams{
		Page: q.Page, PageSize: q.PageSize,
		Source: q.Source, Status: q.Status,
	})
}

func (s *InquiryService) GetByID(id uint) (*models.Inquiry, error) {
	return s.repo.FindByID(id)
}

func (s *InquiryService) Delete(id uint) error {
	return s.repo.SoftDelete(id)
}

// ============ 报价单 ============

type QuotationService struct {
	repo *repository.QuotationRepo
}

func NewQuotationService(r *repository.QuotationRepo) *QuotationService {
	return &QuotationService{repo: r}
}

type CreateQuotationInput struct {
	InquiryID   *uint  `json:"inquiry_id,omitempty"`
	CustomerID  *uint  `json:"customer_id,omitempty"`
	Currency    string `json:"currency"`
	TotalAmount string `json:"total_amount"`
	Items       string `json:"items"`       // JSON 字符串
	ValidDays   int    `json:"valid_days"`  // 有效期天数（生成 valid_until）
	CreatedBy   uint   `json:"-"`
}

func (s *QuotationService) Create(in CreateQuotationInput) (*models.Quotation, error) {
	if in.Currency == "" {
		in.Currency = "USD"
	}
	amount, err := decimal.NewFromString(in.TotalAmount)
	if err != nil {
		return nil, errors.New("报价总金额格式无效：" + in.TotalAmount)
	}

	q := &models.Quotation{
		QuotationNo: "", // 由 repo 生成
		InquiryID:   in.InquiryID,
		CustomerID:  in.CustomerID,
		Currency:    in.Currency,
		TotalAmount: amount,
		Status:      "draft",
		Items:       in.Items,
	}
	q.CreatedBy = in.CreatedBy

	if in.ValidDays > 0 {
		t := time.Now().AddDate(0, 0, in.ValidDays)
		q.ValidUntil = &t
	}

	// 生成报价单号
	no, err := s.repo.NextQuotationNo()
	if err != nil {
		return nil, err
	}
	q.QuotationNo = no

	if err := s.repo.Create(q); err != nil {
		return nil, err
	}
	return q, nil
}

type QuotationListQuery struct {
	Page     int
	PageSize int
	Status   string
}

func (s *QuotationService) List(q QuotationListQuery) (*repository.QuotationListResult, error) {
	return s.repo.List(repository.QuotationListParams{
		Page: q.Page, PageSize: q.PageSize, Status: q.Status,
	})
}

func (s *QuotationService) GetByID(id uint) (*models.Quotation, error) {
	return s.repo.FindByID(id)
}

type UpdateQuotationStatusInput struct {
	Status string `json:"status"` // draft|sent|accepted|rejected|expired
}

// validQuotationStatuses 报价单允许的状态值。
var validQuotationStatuses = map[string]bool{
	"draft": true, "sent": true, "accepted": true, "rejected": true, "expired": true,
}

func (s *QuotationService) UpdateStatus(id uint, in UpdateQuotationStatusInput) error {
	if !validQuotationStatuses[in.Status] {
		return errors.New("无效的报价单状态：" + in.Status + "（允许: draft/sent/accepted/rejected/expired）")
	}
	q, err := s.repo.FindByID(id)
	if err != nil {
		return err
	}
	q.Status = in.Status
	return s.repo.Update(q)
}

func (s *QuotationService) Delete(id uint) error {
	return s.repo.SoftDelete(id)
}
