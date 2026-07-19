// Package models — B2B 外贸 Pack 专属表。
//
// 仅当启用 b2b 模块时才有数据。包含：客户/询盘/报价单/样品/合同/邮件。
package models

import (
	"time"

	"github.com/shopspring/decimal"
)

// Customer 客户公司档案（B2B）。
type Customer struct {
	BaseModel
	CreatedByMixin

	CompanyName     string         `gorm:"not null;size:200;index" json:"company_name"`
	Country         string         `gorm:"size:100;index" json:"country"`
	ContactPerson   string         `gorm:"size:100" json:"contact_person"`
	Email           string         `gorm:"size:200;index" json:"email"`
	Phone           string         `gorm:"size:50" json:"phone"`
	WeChat          string         `gorm:"size:50" json:"wechat"`
	Demand          string         `gorm:"type:text" json:"demand"` // 需求描述
	Stage           CustomerStage  `gorm:"type:text;default:'lead';index" json:"stage"`
	DealProbability decimal.Decimal `gorm:"type:decimal(3,2)" json:"deal_probability"` // AI 预测成交概率
	LastContactAt   *time.Time     `gorm:"index" json:"last_contact_at,omitempty"`
}

// CustomerContact 客户联系人（一个客户多个联系人）。
type CustomerContact struct {
	BaseModel

	CustomerID   uint   `gorm:"not null;index" json:"customer_id"`
	Name         string `gorm:"not null;size:100" json:"name"`
	Title        string `gorm:"size:100" json:"title"`
	Email        string `gorm:"size:200" json:"email"`
	Phone        string `gorm:"size:50" json:"phone"`
	WeChat       string `gorm:"size:50" json:"wechat"`
	IsPrimary    *bool  `gorm:"default:false" json:"is_primary"`
}

// CustomerCommunication 客户沟通记录。
type CustomerCommunication struct {
	BaseModel

	CustomerID uint      `gorm:"not null;index" json:"customer_id"`
	Channel    string    `gorm:"type:text" json:"channel"` // email|wechat|phone|meeting
	Direction  string    `gorm:"type:text" json:"direction"` // inbound|outbound
	Content    string    `gorm:"type:text" json:"content"`
	Summary    string    `gorm:"type:text" json:"summary"`     // AI 生成摘要
	OccurredAt time.Time `gorm:"index" json:"occurred_at"`
}

// Inquiry 询盘。来自 Alibaba.com/展会/邮件的采购询价。
type Inquiry struct {
	BaseModel
	CreatedByMixin

	CustomerID    *uint   `gorm:"index" json:"customer_id,omitempty"`
	Source        string  `gorm:"type:text;index" json:"source"` // alibaba|exhibition|email|website
	ProductDesc   string  `gorm:"not null;type:text" json:"product_desc"` // 询价的产品描述
	Quantity      *int    `json:"quantity,omitempty"`
	TargetPrice   decimal.Decimal `gorm:"type:decimal(10,2)" json:"target_price,omitempty"`
	Destination   string  `gorm:"size:100" json:"destination"` // 目的港
	Status        string  `gorm:"type:text;default:'new';index" json:"status"` // new|quoting|quoted|won|lost
	AIAnalysis    string  `gorm:"type:text" json:"ai_analysis"` // JSON: AI 解析（国家/需求/阶段/概率）
	HandledBy     *uint   `gorm:"index" json:"handled_by,omitempty"`
	HandledAt     *time.Time `gorm:"index" json:"handled_at,omitempty"`
}

// Quotation 报价单。
type Quotation struct {
	BaseModel
	CreatedByMixin

	QuotationNo  string    `gorm:"not null;uniqueIndex:uk_quotations_no;size:50" json:"quotation_no"` // QUO-2026-0001
	InquiryID    *uint     `gorm:"index" json:"inquiry_id,omitempty"`
	CustomerID   *uint     `gorm:"index" json:"customer_id,omitempty"`
	Currency     string    `gorm:"size:3;default:'USD'" json:"currency"`
	TotalAmount  decimal.Decimal `gorm:"type:decimal(12,2)" json:"total_amount"`
	Status       string    `gorm:"type:text;default:'draft';index" json:"status"` // draft|sent|accepted|rejected|expired
	ValidUntil   *time.Time `json:"valid_until,omitempty"`
	Items        string    `gorm:"type:text" json:"items"` // JSON: 报价明细（商品/数量/单价/小计）
	PDFPath      string    `gorm:"type:text" json:"pdf_path"` // 生成的 PDF 路径
}

// Sample 样品追踪。
type Sample struct {
	BaseModel
	CreatedByMixin

	CustomerID    *uint   `gorm:"index" json:"customer_id,omitempty"`
	ProductID     *uint   `gorm:"index" json:"product_id,omitempty"`
	SampleType    string  `gorm:"type:text" json:"sample_type"` // free|charged|returnable
	Status        string  `gorm:"type:text;default:'pending';index" json:"status"` // pending|shipped|received|feedback
	TrackingNo    string  `gorm:"size:100" json:"tracking_no"`
	LogisticsCost decimal.Decimal `gorm:"type:decimal(10,2)" json:"logistics_cost"`
	ShippedAt     *time.Time `json:"shipped_at,omitempty"`
	Feedback      string  `gorm:"type:text" json:"feedback"`
}

// Contract 合同管理。
type Contract struct {
	BaseModel
	CreatedByMixin

	ContractNo   string    `gorm:"not null;uniqueIndex:uk_contracts_no;size:50" json:"contract_no"`
	CustomerID   *uint     `gorm:"index" json:"customer_id,omitempty"`
	QuotationID  *uint     `gorm:"index" json:"quotation_id,omitempty"`
	Amount       decimal.Decimal `gorm:"type:decimal(12,2)" json:"amount"`
	Currency     string    `gorm:"size:3;default:'USD'" json:"currency"`
	Status       string    `gorm:"type:text;default:'draft';index" json:"status"` // draft|signed|executing|completed|terminated
	SignedAt     *time.Time `json:"signed_at,omitempty"`
	FilePath     string    `gorm:"type:text" json:"file_path"`
	Milestones   string    `gorm:"type:text" json:"milestones"` // JSON: 履约节点
}

// EmailThread 邮件线程。用于粘贴邮件 → AI 分析。
type EmailThread struct {
	BaseModel
	CreatedByMixin

	CustomerID    *uint   `gorm:"index" json:"customer_id,omitempty"`
	Subject       string  `gorm:"not null;size:300" json:"subject"`
	FromEmail     string  `gorm:"size:200;index" json:"from_email"`
	ToEmail       string  `gorm:"size:200" json:"to_email"`
	Content       string  `gorm:"not null;type:text" json:"content"`
	AIAnalysis    string  `gorm:"type:text" json:"ai_analysis"` // JSON: {intent, products, urgency, suggested_reply}
	IsHandled     *bool   `gorm:"default:false;index" json:"is_handled"`
	ReceivedAt    time.Time `gorm:"index" json:"received_at"`
}
