// Package models — B2C 跨境电商 Pack 专属表。
//
// 仅当启用 b2c 模块时才有数据。包含：店铺/上架/订单/库存/广告/评论。
package models

import (
	"time"

	"github.com/shopspring/decimal"
)

// Store 店铺授权（多平台 OAuth 令牌）。
type Store struct {
	BaseModel
	CreatedByMixin

	Name         string    `gorm:"not null;size:200;index" json:"name"`
	Platform     string    `gorm:"not null;type:text;size:30;index" json:"platform"` // amazon|shopify|tiktok|temu
	Region       string    `gorm:"size:10" json:"region"`                            // us|uk|de|jp
	StoreID      string    `gorm:"size:100" json:"store_id"`                         // 平台店铺 ID
	AccessToken  string    `gorm:"type:text" json:"-"`                                // AES 加密
	RefreshToken string    `gorm:"type:text" json:"-"`                                // AES 加密
	TokenExpiry  *time.Time `json:"token_expiry,omitempty"`
	Status       string    `gorm:"type:text;default:'active';index" json:"status"` // active|expired|revoked
	SyncedAt     *time.Time `gorm:"index" json:"synced_at,omitempty"`
}

// Listing 上架商品（跨平台）。
type Listing struct {
	BaseModel
	CreatedByMixin

	StoreID        uint   `gorm:"not null;index" json:"store_id"`
	ProductID      *uint  `gorm:"index" json:"product_id,omitempty"` // 关联本地商品池
	PlatformSKU    string `gorm:"not null;size:100;index" json:"platform_sku"`
	PlatformASIN   string `gorm:"size:20" json:"platform_asin"`       // Amazon ASIN
	Title          string `gorm:"not null;size:500" json:"title"`
	Status         string `gorm:"not null;type:text;default:'draft';index" json:"status"` // draft|active|paused|closed
	ListingURL     string `gorm:"type:text" json:"listing_url"`
	SellingPrice   decimal.Decimal `gorm:"type:decimal(10,2)" json:"selling_price"`
	Currency       string `gorm:"size:3;default:'USD'" json:"currency"`
	Stock          *int   `json:"stock,omitempty"`
	PublishedAt    *time.Time `gorm:"index" json:"published_at,omitempty"`
}

// Order 订单（跨平台聚合）。
type Order struct {
	BaseModel

	StoreID        uint            `gorm:"not null;index" json:"store_id"`
	ListingID      *uint           `gorm:"index" json:"listing_id,omitempty"`
	PlatformOrderNo string         `gorm:"not null;uniqueIndex:uk_orders_platform_no;size:100" json:"platform_order_no"`
	Status         OrderStatus     `gorm:"not null;type:text;index" json:"status"`
	Amount         decimal.Decimal `gorm:"type:decimal(10,2)" json:"amount"`
	Currency       string          `gorm:"size:3;default:'USD'" json:"currency"`
	BuyerName      string          `gorm:"size:200" json:"buyer_name"`
	BuyerCountry   string          `gorm:"size:100;index" json:"buyer_country"`
	Items          string          `gorm:"type:text" json:"items"` // JSON: 订单明细
	TrackingNo     string          `gorm:"size:100" json:"tracking_no"`
	OrderedAt      time.Time       `gorm:"index" json:"ordered_at"`
	ShippedAt      *time.Time      `json:"shipped_at,omitempty"`
	DeliveredAt    *time.Time      `json:"delivered_at,omitempty"`
}

// Inventory 库存（FBA/海外仓/自有仓）。
type Inventory struct {
	BaseModel

	ProductID  uint   `gorm:"not null;uniqueIndex:uk_inventory_product_warehouse,priority:1;index" json:"product_id"`
	Warehouse  string `gorm:"not null;size:50;uniqueIndex:uk_inventory_product_warehouse,priority:2" json:"warehouse"` // fba|overseas|own
	Quantity   int    `gorm:"default:0" json:"quantity"`
	Reserved   *int   `gorm:"default:0" json:"reserved"`
	Inbound    *int   `gorm:"default:0" json:"inbound"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// AdCampaign 广告投放（Amazon PPC 等）。
type AdCampaign struct {
	BaseModel
	CreatedByMixin

	StoreID      uint   `gorm:"not null;index" json:"store_id"`
	CampaignID   string `gorm:"not null;size:100;index" json:"campaign_id"` // 平台广告 ID
	Name         string `gorm:"not null;size:200" json:"name"`
	Platform     string `gorm:"type:text" json:"platform"`                  // amazon_sponsored|google|meta|tiktok
	Status       string `gorm:"type:text;default:'active';index" json:"status"` // active|paused|ended
	DailyBudget  decimal.Decimal `gorm:"type:decimal(10,2)" json:"daily_budget"`
	Spend        decimal.Decimal `gorm:"type:decimal(10,2)" json:"spend"`
	Sales        decimal.Decimal `gorm:"type:decimal(10,2)" json:"sales"`
	ACOS         decimal.Decimal `gorm:"type:decimal(5,2)" json:"acos"` // 广告成本销售比 %
	StartDate    *time.Time `json:"start_date,omitempty"`
	EndDate      *time.Time `json:"end_date,omitempty"`
}

// Review 商品评论（AI 情感分析）。
type Review struct {
	BaseModel

	StoreID      *uint  `gorm:"index" json:"store_id,omitempty"`
	ListingID    *uint  `gorm:"index" json:"listing_id,omitempty"`
	PlatformReviewID string `gorm:"size:100" json:"platform_review_id"`
	Rating       *int   `gorm:"index" json:"rating"` // 1-5 星
	Title        string `gorm:"size:300" json:"title"`
	Content      string `gorm:"type:text" json:"content"`
	Sentiment    string `gorm:"type:text" json:"sentiment"` // positive|neutral|negative
	AIAnalysis   string `gorm:"type:text" json:"ai_analysis"` // JSON: AI 提取的关键问题
	IsHandled    *bool  `gorm:"default:false;index" json:"is_handled"`
	ReviewedAt   time.Time `gorm:"index" json:"reviewed_at"`
}
