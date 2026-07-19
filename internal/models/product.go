// Package models — 商品主表 + 商品-供应商多对多关联。
//
// 设计依据：架构设计文档 §2「统一商品数据模型」。
// 一张表覆盖 B2B/B2C 两个场景，扩展字段都可空（SQLite 动态类型空字段零成本）。
package models

import (
	"time"

	"github.com/shopspring/decimal"
)

// Product 商品主表。统一模型，B2B/B2C 字段可空。
//
// 字段分组：
//   - 基础信息（共享）
//   - 采购信息（共享，来源 1688/工厂）
//   - 物理属性（共享）
//   - B2B 扩展（外贸专属，可空）
//   - B2C 扩展（跨境专属，可空）
//   - AI 分析（共享）
type Product struct {
	BaseModel
	CreatedByMixin

	// === 基础信息（共享） ===
	Name        string `gorm:"not null;size:200;index" json:"name"`
	Category    string `gorm:"size:100;index" json:"category"`
	Description string `gorm:"type:text" json:"description"`

	// === 采购信息（共享） ===
	PurchasePrice    decimal.Decimal `gorm:"type:decimal(10,2)" json:"purchase_price"`
	PurchaseCurrency string          `gorm:"type:text;default:'CNY';size:3" json:"purchase_currency"`
	SupplierID       *uint           `gorm:"index" json:"supplier_id,omitempty"`        // 主供应商（多对多见 SupplierProduct）
	Source           DataSource      `gorm:"type:text;index:idx_products_source,priority:1" json:"source"`
	SourceID         string          `gorm:"size:100;index:idx_products_source,priority:2" json:"source_id"`
	SourceURL        string          `gorm:"type:text" json:"source_url"`
	ImageURLs        string          `gorm:"type:text" json:"image_urls"` // JSON 数组

	// === 物理属性（共享） ===
	WeightKG     decimal.Decimal `gorm:"type:decimal(8,3)" json:"weight_kg"`
	VolumeCBM    decimal.Decimal `gorm:"type:decimal(8,4)" json:"volume_cbm"`
	PackageSpec  string          `gorm:"size:200" json:"package_spec"`

	// === B2B 扩展（外贸专属，可空） ===
	B2BFOBPrice      decimal.Decimal `gorm:"type:decimal(10,2)" json:"b2b_fob_price,omitempty"`
	B2BMOQ           *int            `gorm:"index:idx_products_b2b_moq" json:"b2b_moq,omitempty"`
	B2BPriceTiers    string          `gorm:"type:text" json:"b2b_price_tiers,omitempty"`         // JSON: [{qty:100, price:5.8}]
	B2BExportRebate  decimal.Decimal `gorm:"type:decimal(5,2)" json:"b2b_export_rebate,omitempty"` // 出口退税率 %
	B2BHsCode        string          `gorm:"size:20" json:"b2b_hs_code,omitempty"`
	B2BSampleAvail   *bool           `json:"b2b_sample_available,omitempty"`
	B2BLeadTimeDays  *int            `json:"b2b_lead_time_days,omitempty"`

	// === B2C 扩展（跨境专属，可空） ===
	B2CListingURL    string          `gorm:"type:text" json:"b2c_listing_url,omitempty"`
	B2CPlatform      string          `gorm:"size:30;index:idx_products_b2c_platform" json:"b2c_platform,omitempty"` // amazon|shopify|tiktok|temu
	B2CPlatformSKU   string          `gorm:"size:50" json:"b2c_platform_sku,omitempty"`
	B2CSellingPrice  decimal.Decimal `gorm:"type:decimal(10,2)" json:"b2c_selling_price,omitempty"`
	B2CSellingCcy    string          `gorm:"size:3" json:"b2c_selling_currency,omitempty"`
	B2CFBAStock      *int            `json:"b2c_fba_stock,omitempty"`
	B2CWarehouse     string          `gorm:"size:50" json:"b2c_warehouse,omitempty"`
	B2CReviewScore   decimal.Decimal `gorm:"type:decimal(3,1)" json:"b2c_review_score,omitempty"`
	B2CReviewCount   *int            `json:"b2c_review_count,omitempty"`

	// === AI 分析（共享） ===
	AIScore   decimal.Decimal `gorm:"type:decimal(3,1);index" json:"ai_score"`
	AIAnalysis string         `gorm:"type:text" json:"ai_analysis"` // JSON: 完整分析结果

	// === 场景标记（冗余，便于筛选） ===
	Scenarios string `gorm:"type:text;default:'[\"b2b\",\"b2c\"]'" json:"scenarios"` // JSON: ["b2b"]|["b2c"]|["b2b","b2c"]
}

// SupplierProduct 商品-供应商多对多关联。
// 一个商品可以从多个供应商采购（比价）；一个供应商有多个商品。
type SupplierProduct struct {
	BaseModel

	ProductID     uint            `gorm:"not null;uniqueIndex:uk_sp_product_supplier,priority:1;index:idx_sp_product" json:"product_id"`
	SupplierID    uint            `gorm:"not null;uniqueIndex:uk_sp_product_supplier,priority:2;index:idx_sp_supplier" json:"supplier_id"`
	SupplierPrice decimal.Decimal `gorm:"type:decimal(10,2)" json:"supplier_price"`
	SupplierMOQ   *int            `json:"supplier_moq,omitempty"`
	LeadTimeDays  *int            `json:"lead_time_days,omitempty"`
	IsPrimary     *bool           `gorm:"default:false" json:"is_primary"`
	Notes         string          `gorm:"type:text" json:"notes"`
}

// Supplier 供应商（独立模块，架构文档 §1.2）。
type Supplier struct {
	BaseModel
	CreatedByMixin

	Name        string    `gorm:"not null;size:200;index" json:"name"`
	Source      DataSource `gorm:"type:text;index" json:"source"`
	SourceID    string    `gorm:"size:100;uniqueIndex:uk_suppliers_source_id" json:"source_id"`
	Location    string    `gorm:"size:200" json:"location"`
	Contact     string    `gorm:"type:text" json:"contact"` // JSON: {phone,wechat,email}
	ProductCount *int     `gorm:"default:0" json:"product_count"`
	AIScore     decimal.Decimal `gorm:"type:decimal(3,1);index" json:"ai_score"`
	RiskLevel   RiskLevel `gorm:"type:text;default:'medium'" json:"risk_level"`
	LastActiveAt *time.Time `gorm:"index" json:"last_active_at,omitempty"`
}
