// Package repository — B2B 客户/询盘/报价数据访问。
//
// 仅当启用 b2b 模块时使用。
package repository

import (
	"fmt"
	"time"

	"github.com/CainGao/trademind/internal/models"
	"gorm.io/gorm"
)

// CustomerRepo 客户公司档案。
type CustomerRepo struct {
	BaseRepo
}

func NewCustomerRepo(db *gorm.DB) *CustomerRepo {
	return &CustomerRepo{BaseRepo{DB: db}}
}

func (r *CustomerRepo) Create(c *models.Customer) error {
	return r.DB.Create(c).Error
}

func (r *CustomerRepo) FindByID(id uint) (*models.Customer, error) {
	var c models.Customer
	if err := r.DB.First(&c, id).Error; err != nil {
		return nil, err
	}
	return &c, nil
}

// CustomerListParams 客户列表参数。
type CustomerListParams struct {
	Page     int
	PageSize int
	Keyword  string // 公司名/联系人/邮箱
	Country  string
	Stage    string // lead|quoting|negotiating|won|lost
}

type CustomerListResult struct {
	Total int64
	Items []models.Customer
}

func (r *CustomerRepo) List(p CustomerListParams) (*CustomerListResult, error) {
	if p.Page < 1 {
		p.Page = 1
	}
	if p.PageSize < 1 || p.PageSize > 100 {
		p.PageSize = 20
	}
	q := r.DB.Model(&models.Customer{})
	if p.Keyword != "" {
		kw := "%" + p.Keyword + "%"
		q = q.Where("company_name LIKE ? OR contact_person LIKE ? OR email LIKE ?", kw, kw, kw)
	}
	if p.Country != "" {
		q = q.Where("country = ?", p.Country)
	}
	if p.Stage != "" {
		q = q.Where("stage = ?", p.Stage)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, err
	}
	var items []models.Customer
	err := q.Order("created_at DESC").
		Offset((p.Page - 1) * p.PageSize).
		Limit(p.PageSize).
		Find(&items).Error
	return &CustomerListResult{Total: total, Items: items}, err
}

func (r *CustomerRepo) Update(c *models.Customer) error {
	return r.DB.Save(c).Error
}

func (r *CustomerRepo) SoftDelete(id uint) error {
	return r.DB.Delete(&models.Customer{}, id).Error
}

// InquiryRepo 询盘。
type InquiryRepo struct {
	BaseRepo
}

func NewInquiryRepo(db *gorm.DB) *InquiryRepo {
	return &InquiryRepo{BaseRepo{DB: db}}
}

func (r *InquiryRepo) Create(i *models.Inquiry) error {
	return r.DB.Create(i).Error
}

func (r *InquiryRepo) FindByID(id uint) (*models.Inquiry, error) {
	var i models.Inquiry
	if err := r.DB.First(&i, id).Error; err != nil {
		return nil, err
	}
	return &i, nil
}

// InquiryListParams 询盘列表参数。
type InquiryListParams struct {
	Page     int
	PageSize int
	Source   string
	Status   string // new|quoting|quoted|won|lost
}

type InquiryListResult struct {
	Total int64
	Items []models.Inquiry
}

func (r *InquiryRepo) List(p InquiryListParams) (*InquiryListResult, error) {
	if p.Page < 1 {
		p.Page = 1
	}
	if p.PageSize < 1 || p.PageSize > 100 {
		p.PageSize = 20
	}
	q := r.DB.Model(&models.Inquiry{})
	if p.Source != "" {
		q = q.Where("source = ?", p.Source)
	}
	if p.Status != "" {
		q = q.Where("status = ?", p.Status)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, err
	}
	var items []models.Inquiry
	err := q.Order("created_at DESC").
		Offset((p.Page - 1) * p.PageSize).
		Limit(p.PageSize).
		Find(&items).Error
	return &InquiryListResult{Total: total, Items: items}, err
}

func (r *InquiryRepo) Update(i *models.Inquiry) error {
	return r.DB.Save(i).Error
}

func (r *InquiryRepo) SoftDelete(id uint) error {
	return r.DB.Delete(&models.Inquiry{}, id).Error
}

// QuotationRepo 报价单。
type QuotationRepo struct {
	BaseRepo
}

func NewQuotationRepo(db *gorm.DB) *QuotationRepo {
	return &QuotationRepo{BaseRepo{DB: db}}
}

func (r *QuotationRepo) Create(q *models.Quotation) error {
	return r.DB.Create(q).Error
}

func (r *QuotationRepo) FindByID(id uint) (*models.Quotation, error) {
	var q models.Quotation
	if err := r.DB.First(&q, id).Error; err != nil {
		return nil, err
	}
	return &q, nil
}

type QuotationListParams struct {
	Page     int
	PageSize int
	Status   string
}

type QuotationListResult struct {
	Total int64
	Items []models.Quotation
}

func (r *QuotationRepo) List(p QuotationListParams) (*QuotationListResult, error) {
	if p.Page < 1 {
		p.Page = 1
	}
	if p.PageSize < 1 || p.PageSize > 100 {
		p.PageSize = 20
	}
	q := r.DB.Model(&models.Quotation{})
	if p.Status != "" {
		q = q.Where("status = ?", p.Status)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, err
	}
	var items []models.Quotation
	err := q.Order("created_at DESC").
		Offset((p.Page - 1) * p.PageSize).
		Limit(p.PageSize).
		Find(&items).Error
	return &QuotationListResult{Total: total, Items: items}, err
}

func (r *QuotationRepo) Update(q *models.Quotation) error {
	return r.DB.Save(q).Error
}

func (r *QuotationRepo) SoftDelete(id uint) error {
	return r.DB.Delete(&models.Quotation{}, id).Error
}

// NextQuotationNo 生成下一个报价单号 QUO-2026-0001。
func (r *QuotationRepo) NextQuotationNo() (string, error) {
	var count int64
	if err := r.DB.Model(&models.Quotation{}).Count(&count).Error; err != nil {
		return "", err
	}
	return fmt.Sprintf("QUO-%d-%04d", time.Now().Year(), count+1), nil
}
