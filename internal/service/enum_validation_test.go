package service

import (
	"testing"

	"github.com/CainGao/trademind/internal/models"
)

// ===== 客户阶段枚举校验 =====

func TestValidCustomerStages_AllDefined(t *testing.T) {
	expected := []string{"lead", "quoting", "negotiating", "won", "lost"}
	for _, s := range expected {
		if !validCustomerStages[s] {
			t.Errorf("validCustomerStages 缺少 %q", s)
		}
	}
	if len(validCustomerStages) != len(expected) {
		t.Errorf("validCustomerStages 有 %d 项，期望 %d", len(validCustomerStages), len(expected))
	}
}

func TestValidCustomerStages_RejectInvalid(t *testing.T) {
	invalid := []string{"", "hacked", "LEAD", "won!", "prospecting", "closed"}
	for _, s := range invalid {
		if validCustomerStages[s] {
			t.Errorf("validCustomerStages 不应接受 %q", s)
		}
	}
}

func TestValidCustomerStages_MatchModelConstants(t *testing.T) {
	if !validCustomerStages[string(models.CustomerStageLead)] {
		t.Error("缺少 CustomerStageLead")
	}
	if !validCustomerStages[string(models.CustomerStageWon)] {
		t.Error("缺少 CustomerStageWon")
	}
	if !validCustomerStages[string(models.CustomerStageLost)] {
		t.Error("缺少 CustomerStageLost")
	}
}

// ===== 询盘状态枚举校验 =====

func TestValidInquiryStatuses_AllDefined(t *testing.T) {
	expected := []string{"new", "quoting", "quoted", "won", "lost"}
	for _, s := range expected {
		if !validInquiryStatuses[s] {
			t.Errorf("validInquiryStatuses 缺少 %q", s)
		}
	}
	if len(validInquiryStatuses) != len(expected) {
		t.Errorf("validInquiryStatuses 有 %d 项，期望 %d", len(validInquiryStatuses), len(expected))
	}
}

func TestValidInquiryStatuses_RejectInvalid(t *testing.T) {
	invalid := []string{"", "hacked", "NEW", "pending", "done", "cancelled"}
	for _, s := range invalid {
		if validInquiryStatuses[s] {
			t.Errorf("validInquiryStatuses 不应接受 %q", s)
		}
	}
}

// ===== 报价单状态枚举校验 =====

func TestValidQuotationStatuses_AllDefined(t *testing.T) {
	expected := []string{"draft", "sent", "accepted", "rejected", "expired"}
	for _, s := range expected {
		if !validQuotationStatuses[s] {
			t.Errorf("validQuotationStatuses 缺少 %q", s)
		}
	}
	if len(validQuotationStatuses) != len(expected) {
		t.Errorf("validQuotationStatuses 有 %d 项，期望 %d", len(validQuotationStatuses), len(expected))
	}
}

func TestValidQuotationStatuses_RejectInvalid(t *testing.T) {
	invalid := []string{"", "hacked", "DRAFT", "approved", "pending", "closed"}
	for _, s := range invalid {
		if validQuotationStatuses[s] {
			t.Errorf("validQuotationStatuses 不应接受 %q", s)
		}
	}
}

// ===== B2C 订单状态枚举校验 =====

func TestValidOrderStatuses_AllDefined(t *testing.T) {
	expected := []models.OrderStatus{
		models.OrderPending, models.OrderPaid, models.OrderShipped,
		models.OrderDelivered, models.OrderCancelled, models.OrderRefunded,
	}
	for _, s := range expected {
		if !validOrderStatuses[s] {
			t.Errorf("validOrderStatuses 缺少 %q", s)
		}
	}
	if len(validOrderStatuses) != len(expected) {
		t.Errorf("validOrderStatuses 有 %d 项，期望 %d", len(validOrderStatuses), len(expected))
	}
}

func TestValidOrderStatuses_RejectInvalid(t *testing.T) {
	invalid := []models.OrderStatus{
		"", "hacked", "PENDING", "processing", "completed", "returned",
	}
	for _, s := range invalid {
		if validOrderStatuses[s] {
			t.Errorf("validOrderStatuses 不应接受 %q", s)
		}
	}
}

// ===== B2C 店铺平台枚举校验 =====

func TestValidStorePlatforms_AllDefined(t *testing.T) {
	expected := []string{"amazon", "shopify", "tiktok", "temu"}
	for _, s := range expected {
		if !validStorePlatforms[s] {
			t.Errorf("validStorePlatforms 缺少 %q", s)
		}
	}
	if len(validStorePlatforms) != len(expected) {
		t.Errorf("validStorePlatforms 有 %d 项，期望 %d", len(validStorePlatforms), len(expected))
	}
}

func TestValidStorePlatforms_RejectInvalid(t *testing.T) {
	invalid := []string{"", "hacked", "AMAZON", "ebay", "walmart", "aliexpress"}
	for _, s := range invalid {
		if validStorePlatforms[s] {
			t.Errorf("validStorePlatforms 不应接受 %q", s)
		}
	}
}

// ===== B2C 店铺状态枚举校验 =====

func TestValidStoreStatuses_AllDefined(t *testing.T) {
	expected := []string{"active", "expired", "revoked"}
	for _, s := range expected {
		if !validStoreStatuses[s] {
			t.Errorf("validStoreStatuses 缺少 %q", s)
		}
	}
	if len(validStoreStatuses) != len(expected) {
		t.Errorf("validStoreStatuses 有 %d 项，期望 %d", len(validStoreStatuses), len(expected))
	}
}

func TestValidStoreStatuses_RejectInvalid(t *testing.T) {
	invalid := []string{"", "hacked", "ACTIVE", "disabled", "pending", "suspended"}
	for _, s := range invalid {
		if validStoreStatuses[s] {
			t.Errorf("validStoreStatuses 不应接受 %q", s)
		}
	}
}

// ===== B2C 上架状态枚举校验 =====

func TestValidListingStatuses_AllDefined(t *testing.T) {
	expected := []string{"draft", "active", "paused", "closed"}
	for _, s := range expected {
		if !validListingStatuses[s] {
			t.Errorf("validListingStatuses 缺少 %q", s)
		}
	}
	if len(validListingStatuses) != len(expected) {
		t.Errorf("validListingStatuses 有 %d 项，期望 %d", len(validListingStatuses), len(expected))
	}
}

func TestValidListingStatuses_RejectInvalid(t *testing.T) {
	invalid := []string{"", "hacked", "DRAFT", "published", "archived", "deleted"}
	for _, s := range invalid {
		if validListingStatuses[s] {
			t.Errorf("validListingStatuses 不应接受 %q", s)
		}
	}
}

// ===== 调度器 Agent 类型白名单校验 (gotcha #60) =====

func TestValidSchedulableAgents_AllDefined(t *testing.T) {
	expected := []models.AgentType{models.AgentSelection, models.AgentSourcing}
	for _, a := range expected {
		if !validSchedulableAgents[a] {
			t.Errorf("validSchedulableAgents 缺少 %q", a)
		}
	}
	if len(validSchedulableAgents) != len(expected) {
		t.Errorf("validSchedulableAgents 有 %d 项，期望 %d", len(validSchedulableAgents), len(expected))
	}
}

func TestValidSchedulableAgents_RejectInvalid(t *testing.T) {
	// 这些 Agent 类型存在但不应有定时调度
	invalid := []models.AgentType{
		"hacked", "", "report", "email", "inquiry",
		"quotation", "listing", "ad", "review", "analysis",
	}
	for _, a := range invalid {
		if validSchedulableAgents[a] {
			t.Errorf("validSchedulableAgents 不应接受 %q", a)
		}
	}
}
