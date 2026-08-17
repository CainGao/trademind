package service

import (
	"strings"
	"testing"

	"github.com/CainGao/trademind/internal/repository"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"

	"github.com/CainGao/trademind/internal/models"
)

// setupInquiryTestDB 创建内存 SQLite + 迁移询盘表。
func setupInquiryTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:inquiry_update_test?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("打开内存数据库失败: %v", err)
	}
	if err := db.AutoMigrate(&models.Inquiry{}); err != nil {
		t.Fatalf("AutoMigrate 失败: %v", err)
	}
	return db
}

// TestInquiryUpdate_StatusTransition 验证询盘状态流转更新（合法值通过 + 白名单校验）。
func TestInquiryUpdate_StatusTransition(t *testing.T) {
	db := setupInquiryTestDB(t)
	svc := NewInquiryService(repository.NewInquiryRepo(db))

	created, err := svc.Create(CreateInquiryInput{
		Source:      "alibaba",
		ProductDesc: "stainless steel water bottle 500ml",
	})
	if err != nil {
		t.Fatalf("创建询盘失败: %v", err)
	}
	if created.Status != "new" {
		t.Fatalf("新询盘状态应为 new，got %q", created.Status)
	}

	// 合法流转 new → quoting
	quoting := "quoting"
	updated, err := svc.Update(created.ID, UpdateInquiryInput{Status: &quoting})
	if err != nil {
		t.Fatalf("更新状态失败: %v", err)
	}
	if updated.Status != "quoting" {
		t.Errorf("状态应更新为 quoting，got %q", updated.Status)
	}

	// 非法状态 → 拒绝（gotcha #55 白名单）
	hacked := "hacked"
	if _, err := svc.Update(created.ID, UpdateInquiryInput{Status: &hacked}); err == nil ||
		!strings.Contains(err.Error(), "无效的询盘状态") {
		t.Errorf("status=hacked 应报「无效的询盘状态」，got err=%v", err)
	}

	// 不存在的询盘 → 报错
	if _, err := svc.Update(99999, UpdateInquiryInput{Status: &quoting}); err == nil {
		t.Error("更新不存在的询盘应报错")
	}
}

// TestInquiryUpdate_Fields 验证询盘字段更新（decimal 解析错误拒绝，gotcha #53）。
func TestInquiryUpdate_Fields(t *testing.T) {
	db := setupInquiryTestDB(t)
	svc := NewInquiryService(repository.NewInquiryRepo(db))

	created, err := svc.Create(CreateInquiryInput{Source: "email", ProductDesc: "LED strip light"})
	if err != nil {
		t.Fatalf("创建询盘失败: %v", err)
	}

	// 无效目标价 → 拒绝（不静默写零值）
	badPrice := "abc"
	if _, err := svc.Update(created.ID, UpdateInquiryInput{TargetPrice: &badPrice}); err == nil ||
		!strings.Contains(err.Error(), "目标价格式无效") {
		t.Errorf("target_price=abc 应报「目标价格式无效」，got err=%v", err)
	}

	// 空产品描述 → 拒绝
	empty := "   "
	if _, err := svc.Update(created.ID, UpdateInquiryInput{ProductDesc: &empty}); err == nil ||
		!strings.Contains(err.Error(), "不能为空") {
		t.Errorf("空描述应报「不能为空」，got err=%v", err)
	}

	// 合法字段更新
	desc := "LED strip light 5m IP65"
	price := "2.35"
	qty := 5000
	dest := "Hamburg"
	updated, err := svc.Update(created.ID, UpdateInquiryInput{
		ProductDesc: &desc,
		TargetPrice: &price,
		Quantity:    &qty,
		Destination: &dest,
	})
	if err != nil {
		t.Fatalf("合法更新失败: %v", err)
	}
	if updated.ProductDesc != desc || updated.TargetPrice.String() != "2.35" ||
		updated.Quantity == nil || *updated.Quantity != 5000 || updated.Destination != "Hamburg" {
		t.Errorf("字段更新结果不符: %+v", updated)
	}

	// 部分更新：只改状态，其他字段保持
	won := "won"
	updated2, err := svc.Update(created.ID, UpdateInquiryInput{Status: &won})
	if err != nil {
		t.Fatalf("部分更新失败: %v", err)
	}
	if updated2.ProductDesc != desc || updated2.Status != "won" {
		t.Errorf("部分更新不应覆盖其他字段: %+v", updated2)
	}
}
