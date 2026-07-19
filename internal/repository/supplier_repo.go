// Package repository — 供应商数据访问。
//
// 1688 采集时，商品关联的供应商也会自动入库（去重：source + source_id）。
package repository

import (
	"github.com/CainGao/trademind/internal/models"
	"gorm.io/gorm"
)

// SupplierRepo 供应商数据访问。
type SupplierRepo struct {
	BaseRepo
}

func NewSupplierRepo(db *gorm.DB) *SupplierRepo {
	return &SupplierRepo{BaseRepo{DB: db}}
}

// FindBySource 按来源 + 来源ID查找（去重用）。
func (r *SupplierRepo) FindBySource(source models.DataSource, sourceID string) (*models.Supplier, error) {
	var s models.Supplier
	err := r.DB.Where("source = ? AND source_id = ?", source, sourceID).First(&s).Error
	if err != nil {
		return nil, err
	}
	return &s, nil
}

// Create 创建供应商。
func (r *SupplierRepo) Create(s *models.Supplier) error {
	return r.DB.Create(s).Error
}

// UpsertBySource 按来源 + 来源ID存在则更新，不存在则创建（插件采集用）。
// 返回最终记录和是否新建。
func (r *SupplierRepo) UpsertBySource(s *models.Supplier) (*models.Supplier, bool, error) {
	existing, err := r.FindBySource(s.Source, s.SourceID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			if e := r.DB.Create(s).Error; e != nil {
				return nil, false, e
			}
			return s, true, nil
		}
		return nil, false, err
	}
	// 存在：更新可变字段
	updates := map[string]interface{}{
		"name":          s.Name,
		"location":      s.Location,
		"contact":       s.Contact,
		"ai_score":      s.AIScore,
		"risk_level":    s.RiskLevel,
	}
	if e := r.DB.Model(existing).Updates(updates).Error; e != nil {
		return existing, false, e
	}
	return existing, false, nil
}

// FindByID 按主键查。
func (r *SupplierRepo) FindByID(id uint) (*models.Supplier, error) {
	var s models.Supplier
	if err := r.DB.First(&s, id).Error; err != nil {
		return nil, err
	}
	return &s, nil
}

// ListParams 供应商列表参数（复用通用结构）。
type SupplierListParams struct {
	Page     int
	PageSize int
	Keyword  string
	Source   string
}

// List 分页查询。
func (r *SupplierRepo) List(p SupplierListParams) (*SupplierListResult, error) {
	if p.Page < 1 {
		p.Page = 1
	}
	if p.PageSize < 1 || p.PageSize > 100 {
		p.PageSize = 20
	}
	q := r.DB.Model(&models.Supplier{})
	if p.Keyword != "" {
		q = q.Where("name LIKE ?", "%"+p.Keyword+"%")
	}
	if p.Source != "" {
		q = q.Where("source = ?", p.Source)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, err
	}
	var items []models.Supplier
	err := q.Order("created_at DESC").
		Offset((p.Page - 1) * p.PageSize).
		Limit(p.PageSize).
		Find(&items).Error
	return &SupplierListResult{Total: total, Items: items}, err
}

type SupplierListResult struct {
	Total int64
	Items []models.Supplier
}
