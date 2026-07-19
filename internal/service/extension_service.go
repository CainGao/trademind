// Package service — Chrome 插件对接服务。
//
// 三大职责：
//  1. 接收插件采集的商品数据，去重（source + source_id）后入库商品中心
//  2. 接收插件上报的员工行为事件，写入行为资产库
//  3. 提供连接状态检查（插件登录后轮询）
//
// 这是「行为数据资产化」的入口（架构文档 §1.2）。
package service

import (
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/CainGao/trademind/internal/models"
	"github.com/CainGao/trademind/internal/repository"
	"github.com/shopspring/decimal"
)

// ExtensionService 插件业务服务。
type ExtensionService struct {
	productRepo  *repository.ProductRepo
	supplierRepo *repository.SupplierRepo
	behaviorRepo *repository.BehaviorRepo
}

func NewExtensionService(pr *repository.ProductRepo, sr *repository.SupplierRepo, br *repository.BehaviorRepo) *ExtensionService {
	return &ExtensionService{productRepo: pr, supplierRepo: sr, behaviorRepo: br}
}

// SupplierInfo 采集到的供应商信息（嵌在商品采集数据里）。
type SupplierInfo struct {
	Name     string `json:"name"`
	SourceID string `json:"source_id"`
	Location string `json:"location"`
	Contact  string `json:"contact"` // JSON 字符串
}

// CollectProductInput 插件上报的商品采集数据。
type CollectProductInput struct {
	Source      models.DataSource `json:"source"`       // 1688|alibaba|amazon|tiktok|temu
	SourceID    string            `json:"source_id"`    // 平台商品ID/URL指纹
	SourceURL   string            `json:"source_url"`
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Category    string            `json:"category"`
	// 价格
	Price        string `json:"price"`         // 采集到的单价（字符串）
	PriceCurrency string `json:"price_currency"`
	MOQ          *int   `json:"moq,omitempty"` // 起订量（B2B）
	// 图片（数组）
	ImageURLs []string `json:"image_urls"`
	// 规格
	WeightKG    string `json:"weight_kg"`
	PackageSpec string `json:"package_spec"`
	// 供应商
	Supplier SupplierInfo `json:"supplier"`
	// 场景标记（插件可选填，默认根据来源推断）
	Scenarios []string `json:"scenarios"`
}

// CollectResult 采集结果。
type CollectResult struct {
	ProductID    uint   `json:"product_id"`
	SupplierID   uint   `json:"supplier_id,omitempty"`
	IsNewProduct bool   `json:"is_new_product"`
	Action       string `json:"action"` // created | updated
	Message      string `json:"message"`
}

// CollectProduct 接收插件采集的商品。
// 策略：source + source_id 存在则更新可变字段，不存在则新建。
func (s *ExtensionService) CollectProduct(userID uint, in CollectProductInput) (*CollectResult, error) {
	if strings.TrimSpace(in.Name) == "" {
		return nil, errors.New("商品名称不能为空")
	}
	if in.Source == "" {
		return nil, errors.New("来源平台不能为空")
	}

	// 1. 先处理供应商（去重 upsert）
	var supplierID *uint
	if strings.TrimSpace(in.Supplier.Name) != "" {
		sup := &models.Supplier{
			Name:      in.Supplier.Name,
			Source:    in.Source,
			SourceID:  defaultIfEmpty(in.Supplier.SourceID, string(in.Source)+"_vendor"),
			Location:  in.Supplier.Location,
			Contact:   in.Supplier.Contact,
			RiskLevel: models.RiskMedium,
		}
		result, _, err := s.supplierRepo.UpsertBySource(sup)
		if err == nil && result != nil {
			supplierID = &result.ID
		}
	}

	// 2. 处理图片 JSON
	var imgJSON string
	if len(in.ImageURLs) > 0 {
		b, _ := json.Marshal(in.ImageURLs)
		imgJSON = string(b)
	}

	// 3. 场景标记
	scenarios := in.Scenarios
	if len(scenarios) == 0 {
		scenarios = inferScenarios(in.Source)
	}
	scenJSON, _ := json.Marshal(scenarios)

	// 4. 查是否已存在
	existing, err := s.productRepo.FindBySource(in.Source, in.SourceID)
	if err == nil && existing != nil {
		// 更新
		updates := map[string]interface{}{
			"name":           in.Name,
			"description":   in.Description,
			"category":      in.Category,
			"source_url":    in.SourceURL,
			"image_urls":    imgJSON,
			"package_spec":  in.PackageSpec,
			"scenarios":     string(scenJSON),
		}
		if in.Price != "" {
			p, _ := decimal.NewFromString(in.Price)
			updates["purchase_price"] = p
		}
		if in.PriceCurrency != "" {
			updates["purchase_currency"] = in.PriceCurrency
		}
		if in.MOQ != nil {
			updates["b2b_moq"] = in.MOQ
		}
		if in.WeightKG != "" {
			w, _ := decimal.NewFromString(in.WeightKG)
			updates["weight_kg"] = w
		}
		if supplierID != nil {
			updates["supplier_id"] = supplierID
		}
		// 用 GORM Updates 不走 Repo 封装，直接更新
		if e := s.productRepo.DB.Model(existing).Updates(updates).Error; e != nil {
			return nil, e
		}
		return &CollectResult{
			ProductID:    existing.ID,
			SupplierID:   derefUint(supplierID),
			IsNewProduct: false,
			Action:       "updated",
			Message:      "商品已更新（重复采集）",
		}, nil
	}

	// 5. 新建
	price, _ := decimal.NewFromString(in.Price)
	weight, _ := decimal.NewFromString(in.WeightKG)
	p := &models.Product{
		Name:             in.Name,
		Category:         in.Category,
		Description:      in.Description,
		PurchasePrice:    price,
		PurchaseCurrency: defaultIfEmpty(in.PriceCurrency, "CNY"),
		SupplierID:       supplierID,
		Source:           in.Source,
		SourceID:         in.SourceID,
		SourceURL:        in.SourceURL,
		ImageURLs:        imgJSON,
		WeightKG:         weight,
		PackageSpec:      in.PackageSpec,
		B2BMOQ:           in.MOQ,
		Scenarios:        string(scenJSON),
	}
	p.CreatedBy = userID
	if err := s.productRepo.Create(p); err != nil {
		return nil, err
	}
	return &CollectResult{
		ProductID:    p.ID,
		SupplierID:   derefUint(supplierID),
		IsNewProduct: true,
		Action:       "created",
		Message:      "采集成功，已入库",
	}, nil
}

// BehaviorInput 单条行为事件。
type BehaviorInput struct {
	EventType    string                 `json:"event_type"` // browse|search|collect|favorite|export|compare
	Source       models.DataSource      `json:"source"`
	TargetID     string                 `json:"target_id"`
	TargetMeta   map[string]interface{} `json:"target_meta"`
	DurationSec  *int                   `json:"duration_sec,omitempty"`
	OccurredAt   time.Time              `json:"occurred_at"`
}

// ReportBehavior 接收插件上报的单条行为。
func (s *ExtensionService) ReportBehavior(userID uint, in BehaviorInput) error {
	if in.EventType == "" {
		return errors.New("event_type 不能为空")
	}
	var metaJSON string
	if len(in.TargetMeta) > 0 {
		b, _ := json.Marshal(in.TargetMeta)
		metaJSON = string(b)
	}
	ts := in.OccurredAt
	if ts.IsZero() {
		ts = time.Now()
	}
	e := &models.BehaviorEvent{
		UserID:     userID,
		EventType:  in.EventType,
		Source:     in.Source,
		TargetID:   in.TargetID,
		TargetMeta: metaJSON,
		DurationSec: in.DurationSec,
		OccurredAt: ts,
	}
	return s.behaviorRepo.Create(e)
}

// ReportBehaviorBatch 批量上报（插件离线攒一批再发）。
func (s *ExtensionService) ReportBehaviorBatch(userID uint, inputs []BehaviorInput) (int, error) {
	if len(inputs) == 0 {
		return 0, nil
	}
	events := make([]models.BehaviorEvent, 0, len(inputs))
	for _, in := range inputs {
		if in.EventType == "" {
			continue
		}
		var metaJSON string
		if len(in.TargetMeta) > 0 {
			b, _ := json.Marshal(in.TargetMeta)
			metaJSON = string(b)
		}
		ts := in.OccurredAt
		if ts.IsZero() {
			ts = time.Now()
		}
		events = append(events, models.BehaviorEvent{
			UserID:      userID,
			EventType:   in.EventType,
			Source:      in.Source,
			TargetID:    in.TargetID,
			TargetMeta:  metaJSON,
			DurationSec: in.DurationSec,
			OccurredAt:  ts,
		})
	}
	if err := s.behaviorRepo.CreateBatch(events); err != nil {
		return 0, err
	}
	return len(events), nil
}

// ConnectionStatus 插件连接状态检查。
type ConnectionStatus struct {
	Connected bool   `json:"connected"`
	Server    string `json:"server"`
	Version   string `json:"version"`
	User      string `json:"user"`
	Role      string `json:"role"`
}

// Status 返回连接状态（插件登录后轮询）。
func (s *ExtensionService) Status(userName, role string) ConnectionStatus {
	return ConnectionStatus{
		Connected: true,
		Server:    "TradeMind",
		Version:   "0.1.0",
		User:      userName,
		Role:      role,
	}
}

// inferScenarios 根据来源推断场景标记。
func inferScenarios(src models.DataSource) []string {
	switch src {
	case models.Source1688, models.SourceAlibaba, models.SourceFactory:
		return []string{"b2b"}
	case models.SourceAmazon, models.SourceTikTok, models.SourceTemu:
		return []string{"b2c"}
	default:
		return []string{"b2b", "b2c"}
	}
}

func derefUint(p *uint) uint {
	if p == nil {
		return 0
	}
	return *p
}
