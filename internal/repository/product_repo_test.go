package repository

import "testing"

// ===== isValidSortColumn（SQL 注入防护白名单，gotcha #56）=====

func TestIsValidSortColumn_AllowedColumns(t *testing.T) {
	valid := []string{"created_at", "ai_score", "purchase_price", "name", "updated_at"}
	for _, col := range valid {
		if !isValidSortColumn(col) {
			t.Errorf("排序字段 %q 应在白名单内", col)
		}
	}
}

func TestIsValidSortColumn_RejectsSQLInjection(t *testing.T) {
	// 常见 SQL 注入 payload
	injections := []string{
		"created_at; DROP TABLE products--",
		"created_at; DELETE FROM users--",
		"(SELECT CASE WHEN 1=1 THEN created_at ELSE name END)",
		"1;--",
		"*",
		"'; --",
		"created_at ASC; DROP TABLE products",
	}
	for _, payload := range injections {
		if isValidSortColumn(payload) {
			t.Errorf("SQL 注入 payload %q 不应通过白名单校验", payload)
		}
	}
}

func TestIsValidSortColumn_RejectsEmpty(t *testing.T) {
	if isValidSortColumn("") {
		t.Error("空字符串不应通过白名单校验")
	}
}

func TestIsValidSortColumn_RejectsUnknownColumn(t *testing.T) {
	if isValidSortColumn("password_hash") {
		t.Error("非白名单字段 password_hash 不应通过校验")
	}
	if isValidSortColumn("id") {
		t.Error("非白名单字段 id 不应通过校验")
	}
}
