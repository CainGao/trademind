package service

import (
	"strings"
	"testing"

	"github.com/CainGao/trademind/internal/models"
	"github.com/CainGao/trademind/internal/repository"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

// setupQuotationTestDB 创建内存 SQLite + 迁移报价单表。
func setupQuotationTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("打开内存数据库失败: %v", err)
	}
	if err := db.AutoMigrate(&models.Quotation{}); err != nil {
		t.Fatalf("AutoMigrate 失败: %v", err)
	}
	return db
}

// TestQuotationCreate_ValidDaysBounds 验证报价有效期天数边界（gotcha #70）：
// 负数拒绝 / 超大值拒绝（防 AddDate 溢出产生 2739933 年脏数据）/ 合法值正常创建。
func TestQuotationCreate_ValidDaysBounds(t *testing.T) {
	db := setupQuotationTestDB(t)
	svc := NewQuotationService(repository.NewQuotationRepo(db))

	// 负数 → 拒绝
	if _, err := svc.Create(CreateQuotationInput{
		TotalAmount: "100.00", ValidDays: -1,
	}); err == nil || !strings.Contains(err.Error(), "负数") {
		t.Errorf("ValidDays=-1 应报「不能为负数」，got err=%v", err)
	}

	// 超大值 → 拒绝（不产生 2739933 年的 valid_until）
	if _, err := svc.Create(CreateQuotationInput{
		TotalAmount: "100.00", ValidDays: 999999999,
	}); err == nil || !strings.Contains(err.Error(), "有效期天数过长") {
		t.Errorf("ValidDays=999999999 应报「有效期天数过长」，got err=%v", err)
	}

	// 合法值 → 正常创建且 valid_until 已设置
	q, err := svc.Create(CreateQuotationInput{
		TotalAmount: "250.50", ValidDays: 30, Items: `[{"sku":"A1","qty":100,"price":"2.50"}]`,
	})
	if err != nil {
		t.Fatalf("ValidDays=30 不应报错: %v", err)
	}
	if q.ValidUntil == nil {
		t.Error("ValidDays=30 应生成 valid_until")
	}
	if q.ValidUntil.Year() > 2100 {
		t.Errorf("valid_until 年份异常: %d", q.ValidUntil.Year())
	}
}

// TestQuotationCreate_ValidDaysZeroOmitted ValidDays=0 表示不设置有效期。
func TestQuotationCreate_ValidDaysZeroOmitted(t *testing.T) {
	db := setupQuotationTestDB(t)
	svc := NewQuotationService(repository.NewQuotationRepo(db))

	q, err := svc.Create(CreateQuotationInput{TotalAmount: "99.00"})
	if err != nil {
		t.Fatalf("ValidDays 缺省不应报错: %v", err)
	}
	if q.ValidUntil != nil {
		t.Error("ValidDays=0 时 valid_until 应为空")
	}
}
