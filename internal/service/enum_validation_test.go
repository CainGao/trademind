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
