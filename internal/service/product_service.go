// Package service — 商品中心业务逻辑（规范 V1.0 §2.3）。
//
// 职责：参数校验、业务规则、调用 Repo。不直接碰 DB，不写 HTTP。
package service

import (
	"errors"
	"strings"

	"github.com/CainGao/trademind/internal/models"
	"github.com/CainGao/trademind/internal/repository"
	"github.com/shopspring/decimal"
)

// ProductService 商品中心业务。
type ProductService struct {
	productRepo *repository.ProductRepo
}

func NewProductService(pr *repository.ProductRepo) *ProductService {
	return &ProductService{productRepo: pr}
}

// CreateInput 创建商品入参。
type CreateProductInput struct {
	Name             string  `json:"name"`
	Category         string  `json:"category"`
	Description      string  `json:"description"`
	PurchasePrice    string  `json:"purchase_price"` // 字符串前端传入，service 转 decimal
	PurchaseCurrency string  `json:"purchase_currency"`
	SourceURL        string  `json:"source_url"`
	ImageURLs        string  `json:"image_urls"` // JSON 数组字符串
	WeightKG         string  `json:"weight_kg"`
	PackageSpec      string  `json:"package_spec"`
	Scenarios        string  `json:"scenarios"` // JSON 数组
	CreatedBy        uint    `json:"-"`
}

// Create 手动创建商品（前端录入）。
func (s *ProductService) Create(in CreateProductInput) (*models.Product, error) {
	if strings.TrimSpace(in.Name) == "" {
		return nil, errors.New("商品名称不能为空")
	}
	p := &models.Product{
		Name:             in.Name,
		Category:         in.Category,
		Description:      in.Description,
		PurchaseCurrency: defaultIfEmpty(in.PurchaseCurrency, "CNY"),
		SourceURL:        in.SourceURL,
		ImageURLs:        in.ImageURLs,
		PackageSpec:      in.PackageSpec,
		Source:           models.SourceManual,
		Scenarios:        defaultIfEmpty(in.Scenarios, `["b2b","b2c"]`),
	}
	if in.PurchasePrice != "" {
		v, err := decimal.NewFromString(in.PurchasePrice)
		if err != nil {
			return nil, errors.New("采购价格式无效：" + in.PurchasePrice)
		}
		p.PurchasePrice = v
	}
	if in.WeightKG != "" {
		v, err := decimal.NewFromString(in.WeightKG)
		if err != nil {
			return nil, errors.New("重量格式无效：" + in.WeightKG)
		}
		p.WeightKG = v
	}
	p.CreatedBy = in.CreatedBy

	if err := s.productRepo.Create(p); err != nil {
		return nil, err
	}
	return p, nil
}

// GetByID 查看详情。
func (s *ProductService) GetByID(id uint) (*models.Product, error) {
	return s.productRepo.FindByID(id)
}

// UpdateInput 更新入参。
type UpdateProductInput struct {
	Name             *string `json:"name,omitempty"`
	Category         *string `json:"category,omitempty"`
	Description      *string `json:"description,omitempty"`
	PurchasePrice    *string `json:"purchase_price,omitempty"`
	PurchaseCurrency *string `json:"purchase_currency,omitempty"`
	SourceURL        *string `json:"source_url,omitempty"`
	ImageURLs        *string `json:"image_urls,omitempty"`
	WeightKG         *string `json:"weight_kg,omitempty"`
	PackageSpec      *string `json:"package_spec,omitempty"`
	Scenarios        *string `json:"scenarios,omitempty"`
}

// Update 局部更新商品。
func (s *ProductService) Update(id uint, in UpdateProductInput) (*models.Product, error) {
	p, err := s.productRepo.FindByID(id)
	if err != nil {
		return nil, err
	}
	if in.Name != nil {
		p.Name = *in.Name
	}
	if in.Category != nil {
		p.Category = *in.Category
	}
	if in.Description != nil {
		p.Description = *in.Description
	}
	if in.PurchasePrice != nil {
		v, err := decimal.NewFromString(*in.PurchasePrice)
		if err != nil {
			return nil, errors.New("采购价格式无效：" + *in.PurchasePrice)
		}
		p.PurchasePrice = v
	}
	if in.PurchaseCurrency != nil {
		p.PurchaseCurrency = *in.PurchaseCurrency
	}
	if in.SourceURL != nil {
		p.SourceURL = *in.SourceURL
	}
	if in.ImageURLs != nil {
		p.ImageURLs = *in.ImageURLs
	}
	if in.WeightKG != nil {
		v, err := decimal.NewFromString(*in.WeightKG)
		if err != nil {
			return nil, errors.New("重量格式无效：" + *in.WeightKG)
		}
		p.WeightKG = v
	}
	if in.PackageSpec != nil {
		p.PackageSpec = *in.PackageSpec
	}
	if in.Scenarios != nil {
		p.Scenarios = *in.Scenarios
	}
	if err := s.productRepo.Update(p); err != nil {
		return nil, err
	}
	return p, nil
}

// Delete 软删除。
func (s *ProductService) Delete(id uint) error {
	return s.productRepo.SoftDelete(id)
}

// ListQuery 列表查询入参（转 Repo 参数）。
type ListQuery struct {
	Page     int
	PageSize int
	Keyword  string
	Category string
	Source   string
	SortBy   string
	Order    string
}

// List 分页列表。
func (s *ProductService) List(q ListQuery) (*repository.ListResult, error) {
	return s.productRepo.List(repository.ListParams{
		Page:     q.Page,
		PageSize: q.PageSize,
		Keyword:  q.Keyword,
		Category: q.Category,
		Source:   q.Source,
		SortBy:   q.SortBy,
		Order:    q.Order,
	})
}

// Categories 全部分类（前端筛选下拉）。
func (s *ProductService) Categories() ([]string, error) {
	return s.productRepo.Categories()
}

func defaultIfEmpty(s, def string) string {
	if strings.TrimSpace(s) == "" {
		return def
	}
	return s
}
