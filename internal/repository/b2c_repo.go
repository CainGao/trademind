// Package repository — B2C 跨境 Pack 数据访问（店铺/上架/订单/库存/广告/评论）。
//
// 设计依据：架构设计文档 §1.2 B2C 模块。
// 所有表已在 AutoMigrate 时建好，这里只做 CRUD。
package repository

import (
	"github.com/CainGao/trademind/internal/models"
	"gorm.io/gorm"
)

// StoreRepo 店铺授权。
type StoreRepo struct {
	BaseRepo
}

func NewStoreRepo(db *gorm.DB) *StoreRepo {
	return &StoreRepo{BaseRepo{DB: db}}
}

func (r *StoreRepo) Create(s *models.Store) error { return r.DB.Create(s).Error }
func (r *StoreRepo) Update(s *models.Store) error { return r.DB.Save(s).Error }
func (r *StoreRepo) Delete(id uint) error         { return r.DB.Delete(&models.Store{}, id).Error }
func (r *StoreRepo) FindByID(id uint) (*models.Store, error) {
	var s models.Store
	err := r.DB.First(&s, id).Error
	return &s, err
}

// List 店铺列表（支持 platform 过滤）。
func (r *StoreRepo) List(page, pageSize int, platform string) (items []models.Store, total int64, err error) {
	q := r.DB.Model(&models.Store{})
	if platform != "" {
		q = q.Where("platform = ?", platform)
	}
	if err = q.Count(&total).Error; err != nil {
		return
	}
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	err = q.Order("created_at DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&items).Error
	return
}

// ListingRepo 上架商品。
type ListingRepo struct {
	BaseRepo
}

func NewListingRepo(db *gorm.DB) *ListingRepo {
	return &ListingRepo{BaseRepo{DB: db}}
}

func (r *ListingRepo) Create(l *models.Listing) error { return r.DB.Create(l).Error }
func (r *ListingRepo) Update(l *models.Listing) error { return r.DB.Save(l).Error }
func (r *ListingRepo) Delete(id uint) error            { return r.DB.Delete(&models.Listing{}, id).Error }
func (r *ListingRepo) FindByID(id uint) (*models.Listing, error) {
	var l models.Listing
	err := r.DB.First(&l, id).Error
	return &l, err
}

// List 上架列表（支持 store_id / status 过滤）。
func (r *ListingRepo) List(page, pageSize int, storeID uint, status string) (items []models.Listing, total int64, err error) {
	q := r.DB.Model(&models.Listing{})
	if storeID > 0 {
		q = q.Where("store_id = ?", storeID)
	}
	if status != "" {
		q = q.Where("status = ?", status)
	}
	if err = q.Count(&total).Error; err != nil {
		return
	}
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	err = q.Order("created_at DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&items).Error
	return
}

// OrderRepo 订单（跨平台聚合）。
type OrderRepo struct {
	BaseRepo
}

func NewOrderRepo(db *gorm.DB) *OrderRepo {
	return &OrderRepo{BaseRepo{DB: db}}
}

func (r *OrderRepo) Create(o *models.Order) error { return r.DB.Create(o).Error }
func (r *OrderRepo) Update(o *models.Order) error { return r.DB.Save(o).Error }
func (r *OrderRepo) FindByID(id uint) (*models.Order, error) {
	var o models.Order
	err := r.DB.First(&o, id).Error
	return &o, err
}

// OrderListResult 订单列表项（带店铺名，避免前端 N+1 查询）。
type OrderListResult struct {
	models.Order
	StoreName string `json:"store_name"`
	Platform  string `json:"platform"`
}

// List 订单列表（支持 store_id / status / 国家过滤）。JOIN 店铺拿名字。
func (r *OrderRepo) List(page, pageSize int, storeID uint, status, country string) (items []OrderListResult, total int64, err error) {
	q := r.DB.Table("orders o").
		Select("o.*, s.name AS store_name, s.platform AS platform").
		Joins("LEFT JOIN stores s ON s.id = o.store_id").
		Where("o.deleted_at IS NULL")
	if storeID > 0 {
		q = q.Where("o.store_id = ?", storeID)
	}
	if status != "" {
		q = q.Where("o.status = ?", status)
	}
	if country != "" {
		q = q.Where("o.buyer_country = ?", country)
	}
	if err = q.Count(&total).Error; err != nil {
		return
	}
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	err = q.Order("o.ordered_at DESC").Offset((page - 1) * pageSize).Limit(pageSize).Scan(&items).Error
	return
}

// Overview B2C 订单总览（驾驶舱用）。
type OrderOverview struct {
	TotalOrders   int64   `json:"total_orders"`
	TotalRevenue  float64 `json:"total_revenue"`
	PendingCount  int64   `json:"pending_count"`
	ShippedCount  int64   `json:"shipped_count"`
	DeliveredCount int64  `json:"delivered_count"`
}

func (r *OrderRepo) Overview() (*OrderOverview, error) {
	var ov OrderOverview
	err := r.DB.Table("orders").
		Select(`COUNT(*) AS total_orders,
			COALESCE(SUM(amount), 0) AS total_revenue,
			SUM(CASE WHEN status='pending' THEN 1 ELSE 0 END) AS pending_count,
			SUM(CASE WHEN status='shipped' THEN 1 ELSE 0 END) AS shipped_count,
			SUM(CASE WHEN status='delivered' THEN 1 ELSE 0 END) AS delivered_count`).
		Where("deleted_at IS NULL").
		Scan(&ov).Error
	return &ov, err
}

// InventoryRepo 库存（FBA/海外仓/自有仓）。
type InventoryRepo struct {
	BaseRepo
}

func NewInventoryRepo(db *gorm.DB) *InventoryRepo {
	return &InventoryRepo{BaseRepo{DB: db}}
}

func (r *InventoryRepo) Upsert(inv *models.Inventory) error {
	// uniqueIndex(product_id, warehouse) — 存在则更新
	return r.DB.Where("product_id = ? AND warehouse = ?", inv.ProductID, inv.Warehouse).
		Assign(inv).FirstOrCreate(inv).Error
}

// LowStock 低库存预警（quantity < threshold）。
// 加 LIMIT 防止超大数据集内存耗尽（gotcha #58 同类问题）。
func (r *InventoryRepo) LowStock(threshold int) (items []models.Inventory, err error) {
	err = r.DB.Where("quantity < ?", threshold).Order("quantity ASC").Limit(500).Find(&items).Error
	return
}
