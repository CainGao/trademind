// Package service — 供应商业务逻辑。
//
// 职责：列表/详情/评分/删除 + 关联商品查询 + 总览统计。
package service

import (
	"errors"
	"strings"

	"github.com/CainGao/trademind/internal/models"
	"github.com/CainGao/trademind/internal/repository"
	"github.com/shopspring/decimal"
)

type SupplierService struct {
	repo *repository.SupplierRepo
}

func NewSupplierService(r *repository.SupplierRepo) *SupplierService {
	return &SupplierService{repo: r}
}

// ListQuery 列表入参。
type SupplierListQuery struct {
	Page      int
	PageSize  int
	Keyword   string
	Source    string
	RiskLevel string
}

// List 分页列表。
func (s *SupplierService) List(q SupplierListQuery) (*repository.SupplierListResult, error) {
	return s.repo.List(repository.SupplierListParams{
		Page:      q.Page,
		PageSize:  q.PageSize,
		Keyword:   q.Keyword,
		Source:    q.Source,
		RiskLevel: q.RiskLevel,
	})
}

// SupplierDetail 详情（含商品计数）。
type SupplierDetail struct {
	models.Supplier
	ProductCount int64 `json:"product_count"`
}

// GetByID 详情。
func (s *SupplierService) GetByID(id uint) (*SupplierDetail, error) {
	sup, err := s.repo.FindByID(id)
	if err != nil {
		return nil, err
	}
	cnt, _ := s.repo.ProductCount(id)
	return &SupplierDetail{Supplier: *sup, ProductCount: cnt}, nil
}

// Products 关联商品列表。
func (s *SupplierService) Products(supplierID uint, page, pageSize int) ([]models.Product, int64, error) {
	return s.repo.ProductsOfSupplier(supplierID, page, pageSize)
}

// UpdateRiskInput 评分入参。
type UpdateRiskInput struct {
	RiskLevel string `json:"risk_level"` // low|medium|high
	AIScore   string `json:"ai_score"`   // 选填，0-10
}

// UpdateRisk 老板/采购员手动评分。
func (s *SupplierService) UpdateRisk(id uint, in UpdateRiskInput) error {
	risk := models.RiskLevel(strings.ToLower(in.RiskLevel))
	switch risk {
	case models.RiskLow, models.RiskMedium, models.RiskHigh:
		// ok
	default:
		return errors.New("risk_level 必须是 low/medium/high")
	}
	// ai_score 校验：0-10
	scoreStr := strings.TrimSpace(in.AIScore)
	if scoreStr != "" {
		d, err := decimal.NewFromString(scoreStr)
		if err != nil {
			return errors.New("ai_score 不是合法数字")
		}
		if d.LessThan(decimal.Zero) || d.GreaterThan(decimal.NewFromInt(10)) {
			return errors.New("ai_score 必须在 0-10 之间")
		}
	}
	return s.repo.UpdateRisk(id, risk, scoreStr)
}

// Delete 软删除。
func (s *SupplierService) Delete(id uint) error {
	return s.repo.SoftDelete(id)
}

// Overview 供应商总览。
func (s *SupplierService) Overview() (*repository.SupplierOverview, error) {
	return s.repo.Overview()
}
