package service

import (
	"strings"
	"testing"

	"github.com/CainGao/trademind/internal/models"
	"github.com/shopspring/decimal"
)

// TestFmtCount 验证聚合计数值格式化（nil 兜底 0，绝不打印 <nil>）。
func TestFmtCount(t *testing.T) {
	cases := []struct {
		name string
		in   interface{}
		want string
	}{
		{"nil 值兜底 0", nil, "0"},
		{"int64 正常", int64(42), "42"},
		{"零值", int64(0), "0"},
		{"字符串数字", "7", "7"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := fmtCount(c.in); got != c.want {
				t.Fatalf("fmtCount(%v) = %q, want %q", c.in, got, c.want)
			}
		})
	}
}

// TestFmtSupplierID 验证 *uint 指针格式化（nil → 无，非 nil → 数字，绝不打印内存地址）。
func TestFmtSupplierID(t *testing.T) {
	if got := fmtSupplierID(nil); got != "无" {
		t.Fatalf("fmtSupplierID(nil) = %q, want %q", got, "无")
	}
	id := uint(7)
	if got := fmtSupplierID(&id); got != "7" {
		t.Fatalf("fmtSupplierID(&7) = %q, want %q", got, "7")
	}
}

// TestBuildSelectionPrompt_NoNilLeak 验证选品 prompt 不含 <nil>（聚合别名 cnt + nil 兜底）。
func TestBuildSelectionPrompt_NoNilLeak(t *testing.T) {
	// 模拟 behavior_repo 返回（别名 cnt；故意带一个缺 cnt 的脏行验证兜底）
	topKeywords := []map[string]interface{}{
		{"keyword": "钢化膜", "cnt": int64(12)},
		{"keyword": "手机壳", "cnt": int64(5)},
		{"keyword": "脏数据行"}, // 缺 cnt → fmtCount 兜底 0
	}
	byType := []map[string]interface{}{
		{"event_type": "search", "cnt": int64(30)},
		{"event_type": "browse"}, // 缺 cnt
	}
	products := []models.Product{
		{Name: "硅胶手机壳", Category: "手机配件", PurchasePrice: decimal.NewFromFloat(2.5)},
	}

	prompt := buildSelectionPrompt(14, topKeywords, byType, products)

	if strings.Contains(prompt, "<nil>") {
		t.Fatalf("选品 prompt 不应包含 <nil>:\n%s", prompt)
	}
	if !strings.Contains(prompt, "钢化膜（12 次）") {
		t.Fatalf("Top 关键词计数应使用 cnt 别名（钢化膜（12 次））:\n%s", prompt)
	}
	if !strings.Contains(prompt, "search: 30 次") {
		t.Fatalf("行为分布应使用 cnt 别名（search: 30 次）:\n%s", prompt)
	}
	if !strings.Contains(prompt, "脏数据行（0 次）") {
		t.Fatalf("缺 cnt 的行应兜底 0（脏数据行（0 次））:\n%s", prompt)
	}
}

// TestBuildSourcingPrompt_NoPointerLeak 验证采购 prompt 不含指针地址（0x...）。
func TestBuildSourcingPrompt_NoPointerLeak(t *testing.T) {
	supplierID := uint(3)
	products := []models.Product{
		{Name: "蓝牙耳机", PurchasePrice: decimal.NewFromFloat(15.5), SupplierID: &supplierID},
		{Name: "无供应商商品", PurchasePrice: decimal.NewFromFloat(9.9), SupplierID: nil},
	}

	prompt := buildSourcingPrompt(products, nil)

	if strings.Contains(prompt, "0x") {
		t.Fatalf("采购 prompt 不应包含指针内存地址:\n%s", prompt)
	}
	if !strings.Contains(prompt, "供应商ID: 3") {
		t.Fatalf("非 nil 供应商应打印数字 ID:\n%s", prompt)
	}
	if !strings.Contains(prompt, "供应商ID: 无") {
		t.Fatalf("nil 供应商应打印 无:\n%s", prompt)
	}
}
