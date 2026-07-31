// Package service — B2C 跨境电商业务（店铺/上架/订单）。
//
// B2C 模块的核心三件套：店铺管理（OAuth 授权）+ 上架管理 + 订单聚合。
// 数据源：店铺平台 API 同步（V2 版本接入 Amazon SP-API / Shopify Admin API）。
// 当前版本：本地 CRUD + 手动录入，V2 再接 API 同步。
package service

import (
	"errors"

	"github.com/CainGao/trademind/internal/models"
	"github.com/CainGao/trademind/internal/repository"
)

// B2CService B2C 业务服务。
type B2CService struct {
	storeRepo   *repository.StoreRepo
	listingRepo *repository.ListingRepo
	orderRepo   *repository.OrderRepo
}

// NewB2CService 构造。
func NewB2CService(store *repository.StoreRepo, listing *repository.ListingRepo, order *repository.OrderRepo) *B2CService {
	return &B2CService{storeRepo: store, listingRepo: listing, orderRepo: order}
}

// ========== 店铺 ==========

// ListStores 店铺列表。
func (s *B2CService) ListStores(page, pageSize int, platform string) ([]models.Store, int64, error) {
	return s.storeRepo.List(page, pageSize, platform)
}

// CreateStore 创建店铺。
func (s *B2CService) CreateStore(store *models.Store) error {
	return s.storeRepo.Create(store)
}

// UpdateStore 更新店铺。
func (s *B2CService) UpdateStore(store *models.Store) error {
	return s.storeRepo.Update(store)
}

// DeleteStore 删除店铺（软删除）。
func (s *B2CService) DeleteStore(id uint) error {
	return s.storeRepo.Delete(id)
}

// ========== 上架 ==========

// ListListings 上架列表。
func (s *B2CService) ListListings(page, pageSize int, storeID uint, status string) ([]models.Listing, int64, error) {
	return s.listingRepo.List(page, pageSize, storeID, status)
}

// CreateListing 创建上架。
func (s *B2CService) CreateListing(l *models.Listing) error {
	return s.listingRepo.Create(l)
}

// UpdateListing 更新上架。
func (s *B2CService) UpdateListing(l *models.Listing) error {
	return s.listingRepo.Update(l)
}

// ========== 订单 ==========

// ListOrders 订单列表（带店铺名）。
func (s *B2CService) ListOrders(page, pageSize int, storeID uint, status, country string) ([]repository.OrderListResult, int64, error) {
	return s.orderRepo.List(page, pageSize, storeID, status, country)
}

// OrderOverview 订单总览（驾驶舱用）。
func (s *B2CService) OrderOverview() (*repository.OrderOverview, error) {
	return s.orderRepo.Overview()
}

// CreateOrder 手动创建订单（一般用平台同步，这里支持手动补录）。
func (s *B2CService) CreateOrder(o *models.Order) error {
	return s.orderRepo.Create(o)
}

// validOrderStatuses B2C 订单允许的状态值。
var validOrderStatuses = map[models.OrderStatus]bool{
	models.OrderPending:   true,
	models.OrderPaid:      true,
	models.OrderShipped:   true,
	models.OrderDelivered: true,
	models.OrderCancelled: true,
	models.OrderRefunded:  true,
}

// UpdateOrderStatus 更新订单状态。
func (s *B2CService) UpdateOrderStatus(id uint, status models.OrderStatus) error {
	if !validOrderStatuses[status] {
		return errors.New("无效的订单状态：" + string(status) + "（允许: pending/paid/shipped/delivered/cancelled/refunded）")
	}
	o, err := s.orderRepo.FindByID(id)
	if err != nil {
		return err
	}
	o.Status = status
	return s.orderRepo.Update(o)
}
