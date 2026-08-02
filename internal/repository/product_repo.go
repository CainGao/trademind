// Package repository — 商品数据访问（规范 V1.0 §2.2）。
//
// 职责：纯 CRUD + 查询，不写业务逻辑。
// 软删除由 BaseModel 的 DeletedAt 字段支持（gorm.DeletedAt）。
package repository

import (
	"github.com/CainGao/trademind/internal/models"
	"gorm.io/gorm"
)

// ProductRepo 商品表数据访问。
type ProductRepo struct {
	BaseRepo
}

func NewProductRepo(db *gorm.DB) *ProductRepo {
	return &ProductRepo{BaseRepo{DB: db}}
}

// Create 创建商品。
func (r *ProductRepo) Create(p *models.Product) error {
	return r.DB.Create(p).Error
}

// FindByID 按主键查（软删除自动过滤）。
func (r *ProductRepo) FindByID(id uint) (*models.Product, error) {
	var p models.Product
	if err := r.DB.First(&p, id).Error; err != nil {
		return nil, err
	}
	return &p, nil
}

// FindBySource 按来源平台 + 来源 ID 查（用于插件采集去重）。
func (r *ProductRepo) FindBySource(source models.DataSource, sourceID string) (*models.Product, error) {
	var p models.Product
	err := r.DB.Where("source = ? AND source_id = ?", source, sourceID).First(&p).Error
	if err != nil {
		return nil, err
	}
	return &p, nil
}

// ListParams 商品列表查询参数。
type ListParams struct {
	Page     int    // 1-based
	PageSize int    // 每页条数
	Keyword  string // 名称模糊匹配
	Category string // 分类精确匹配
	Source   string // 来源平台筛选
	SortBy   string // 排序字段：created_at（默认）|ai_score|purchase_price
	Order    string // asc|desc（默认 desc）
}

// ListResult 分页结果。
type ListResult struct {
	Total int64
	Items []models.Product
}

// List 分页查询商品列表。
func (r *ProductRepo) List(p ListParams) (*ListResult, error) {
	if p.Page < 1 {
		p.Page = 1
	}
	if p.PageSize < 1 || p.PageSize > 100 {
		p.PageSize = 20
	}
	if p.SortBy == "" {
		p.SortBy = "created_at"
	}
	// 白名单校验排序字段，防止 SQL 注入（gotcha #56）
	if !isValidSortColumn(p.SortBy) {
		p.SortBy = "created_at"
	}
	if p.Order != "asc" {
		p.Order = "desc"
	}

	q := r.DB.Model(&models.Product{})

	if p.Keyword != "" {
		q = q.Where("name LIKE ?", "%"+p.Keyword+"%")
	}
	if p.Category != "" {
		q = q.Where("category = ?", p.Category)
	}
	if p.Source != "" {
		q = q.Where("source = ?", p.Source)
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, err
	}

	var items []models.Product
	err := q.Order(p.SortBy + " " + p.Order).
		Offset((p.Page - 1) * p.PageSize).
		Limit(p.PageSize).
		Find(&items).Error
	if err != nil {
		return nil, err
	}
	return &ListResult{Total: total, Items: items}, nil
}

// Update 更新商品（全字段更新，仅更新非零字段由前端控制）。
// 注意：decimal/text 类型零值判断由 service 层处理，这里用 Save 全量覆盖。
func (r *ProductRepo) Update(p *models.Product) error {
	return r.DB.Save(p).Error
}

// SoftDelete 软删除（gorm.DeletedAt 自动处理）。
func (r *ProductRepo) SoftDelete(id uint) error {
	return r.DB.Delete(&models.Product{}, id).Error
}

// Categories 返回所有非空分类（去重，用于前端筛选下拉框）。
func (r *ProductRepo) Categories() ([]string, error) {
	var cats []string
	err := r.DB.Model(&models.Product{}).
		Where("category != '' AND deleted_at IS NULL").
		Distinct("category").
		Order("category").
		Pluck("category", &cats).Error
	return cats, err
}

// productSortWhitelist 允许排序的字段白名单（防止 SQL 注入，gotcha #56）。
var productSortWhitelist = map[string]bool{
	"created_at":     true,
	"ai_score":       true,
	"purchase_price": true,
	"name":           true,
	"updated_at":     true,
}

// isValidSortColumn 检查排序字段是否在白名单内。
func isValidSortColumn(col string) bool {
	return productSortWhitelist[col]
}
