package service

import (
	"testing"

	"github.com/shopspring/decimal"
)

// ===== safeParseDecimal（gotcha #53 同类修复：插件采集容错解析）=====

func TestSafeParseDecimal_ValidNumber(t *testing.T) {
	d, ok := safeParseDecimal("123.45")
	if !ok {
		t.Fatal("应解析成功")
	}
	if !d.Equal(decimal.NewFromFloat(123.45)) {
		t.Errorf("解析结果 %s 不等于 123.45", d.String())
	}
}

func TestSafeParseDecimal_EmptyString(t *testing.T) {
	d, ok := safeParseDecimal("")
	if ok {
		t.Error("空字符串应返回 false")
	}
	if !d.IsZero() {
		t.Error("空字符串应返回零值")
	}
}

func TestSafeParseDecimal_WhitespaceOnly(t *testing.T) {
	d, ok := safeParseDecimal("   ")
	if ok {
		t.Error("纯空白字符串应返回 false")
	}
	if !d.IsZero() {
		t.Error("纯空白字符串应返回零值")
	}
}

func TestSafeParseDecimal_InvalidString(t *testing.T) {
	invalidInputs := []string{"abc", "面议", "12.34.56", "$$$", "价格"}
	for _, s := range invalidInputs {
		d, ok := safeParseDecimal(s)
		if ok {
			t.Errorf("无效输入 %q 应返回 false", s)
		}
		if !d.IsZero() {
			t.Errorf("无效输入 %q 应返回零值", s)
		}
	}
}

func TestSafeParseDecimal_Zero(t *testing.T) {
	d, ok := safeParseDecimal("0")
	if !ok {
		t.Fatal("'0' 应解析成功")
	}
	if !d.IsZero() {
		t.Errorf("'0' 解析结果 %s 应为零值", d.String())
	}
}

func TestSafeParseDecimal_Negative(t *testing.T) {
	d, ok := safeParseDecimal("-99.99")
	if !ok {
		t.Fatal("负数应解析成功")
	}
	if !d.Equal(decimal.NewFromFloat(-99.99)) {
		t.Errorf("负数解析结果 %s 不等于 -99.99", d.String())
	}
}

func TestSafeParseDecimal_Integer(t *testing.T) {
	d, ok := safeParseDecimal("1000")
	if !ok {
		t.Fatal("整数应解析成功")
	}
	if !d.Equal(decimal.NewFromInt(1000)) {
		t.Errorf("整数解析结果 %s 不等于 1000", d.String())
	}
}

// ===== 行为事件类型白名单校验（gotcha #55 枚举校验）=====

func TestValidBehaviorEventTypes_AllDefined(t *testing.T) {
	expected := []string{"browse", "search", "collect", "favorite", "export", "compare"}
	for _, s := range expected {
		if !validBehaviorEventTypes[s] {
			t.Errorf("validBehaviorEventTypes 缺少 %q", s)
		}
	}
	if len(validBehaviorEventTypes) != len(expected) {
		t.Errorf("validBehaviorEventTypes 有 %d 项，期望 %d", len(validBehaviorEventTypes), len(expected))
	}
}

func TestValidBehaviorEventTypes_RejectInvalid(t *testing.T) {
	invalid := []string{"", "hacked", "BROWSE", "click", "view", "login", "logout", "delete", "admin"}
	for _, s := range invalid {
		if validBehaviorEventTypes[s] {
			t.Errorf("validBehaviorEventTypes 不应接受 %q", s)
		}
	}
}
