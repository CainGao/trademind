package handler

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

// gotcha #70：B2B 三件套（客户/询盘/报价单）输入字段长度校验。
// 模式与 product_input_test.go 一致：nil service，校验在 service 之前触发。

// newB2BTestRouter 构造带 Recovery 的测试路由（正常输入时 nil service panic 被兜住）。
func newB2BTestRouter(method, path string, hf gin.HandlerFunc) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(gin.Recovery())
	r.Handle(method, path, hf)
	return r
}

func doJSON(r *gin.Engine, method, path, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

// TestB2B_CreateCustomer_RejectsOversizedCompanyName 公司名称超长被拒绝。
func TestB2B_CreateCustomer_RejectsOversizedCompanyName(t *testing.T) {
	h := &B2BHandler{}
	r := newB2BTestRouter("POST", "/api/customers", h.CreateCustomer)

	name := strings.Repeat("公", maxCustomerCompanyName/3+1) // UTF-8 中文每字 3 字节，确保超限
	w := doJSON(r, "POST", "/api/customers", `{"company_name":"`+name+`"}`)
	if w.Code != http.StatusBadRequest {
		t.Errorf("oversized company_name: got status %d, want %d; body=%s", w.Code, http.StatusBadRequest, w.Body.String())
	}
}

// TestB2B_CreateCustomer_RejectsOversizedDemand 需求描述超长被拒绝。
func TestB2B_CreateCustomer_RejectsOversizedDemand(t *testing.T) {
	h := &B2BHandler{}
	r := newB2BTestRouter("POST", "/api/customers", h.CreateCustomer)

	demand := strings.Repeat("a", maxCustomerDemand+1)
	w := doJSON(r, "POST", "/api/customers", `{"company_name":"OK","demand":"`+demand+`"}`)
	if w.Code != http.StatusBadRequest {
		t.Errorf("oversized demand: got status %d, want %d", w.Code, http.StatusBadRequest)
	}
}

// TestB2B_UpdateCustomer_RejectsOversizedEmail 更新时邮箱超长被拒绝（指针字段路径）。
func TestB2B_UpdateCustomer_RejectsOversizedEmail(t *testing.T) {
	h := &B2BHandler{}
	r := newB2BTestRouter("PUT", "/api/customers/:id", h.UpdateCustomer)

	email := strings.Repeat("a", maxCustomerEmail+1)
	w := doJSON(r, "PUT", "/api/customers/1", `{"email":"`+email+`"}`)
	if w.Code != http.StatusBadRequest {
		t.Errorf("oversized email on update: got status %d, want %d", w.Code, http.StatusBadRequest)
	}
}

// TestB2B_UpdateCustomer_AcceptsNormalUpdate 正常更新输入不被长度校验拒绝。
func TestB2B_UpdateCustomer_AcceptsNormalUpdate(t *testing.T) {
	h := &B2BHandler{}
	r := newB2BTestRouter("PUT", "/api/customers/:id", h.UpdateCustomer)

	w := doJSON(r, "PUT", "/api/customers/1", `{"country":"Germany","stage":"won"}`)
	if w.Code == http.StatusBadRequest {
		t.Errorf("正常更新输入不应被校验拒绝: %s", w.Body.String())
	}
}

// TestB2B_CreateInquiry_RejectsOversizedProductDesc 询盘产品描述超长被拒绝。
func TestB2B_CreateInquiry_RejectsOversizedProductDesc(t *testing.T) {
	h := &B2BHandler{}
	r := newB2BTestRouter("POST", "/api/inquiries", h.CreateInquiry)

	desc := strings.Repeat("a", maxInquiryProductDesc+1)
	w := doJSON(r, "POST", "/api/inquiries", `{"product_desc":"`+desc+`"}`)
	if w.Code != http.StatusBadRequest {
		t.Errorf("oversized product_desc: got status %d, want %d", w.Code, http.StatusBadRequest)
	}
}

// TestB2B_CreateInquiry_RejectsOversizedSource 询盘来源超长被拒绝。
func TestB2B_CreateInquiry_RejectsOversizedSource(t *testing.T) {
	h := &B2BHandler{}
	r := newB2BTestRouter("POST", "/api/inquiries", h.CreateInquiry)

	src := strings.Repeat("s", maxInquirySource+1)
	w := doJSON(r, "POST", "/api/inquiries", `{"product_desc":"steel pipe","source":"`+src+`"}`)
	if w.Code != http.StatusBadRequest {
		t.Errorf("oversized source: got status %d, want %d", w.Code, http.StatusBadRequest)
	}
}

// TestB2B_CreateQuotation_RejectsOversizedItems 报价明细 JSON 超长被拒绝。
func TestB2B_CreateQuotation_RejectsOversizedItems(t *testing.T) {
	h := &B2BHandler{}
	r := newB2BTestRouter("POST", "/api/quotations", h.CreateQuotation)

	items := strings.Repeat("a", maxQuotationItems+1)
	w := doJSON(r, "POST", "/api/quotations", `{"total_amount":"100.00","items":"`+items+`"}`)
	if w.Code != http.StatusBadRequest {
		t.Errorf("oversized items: got status %d, want %d", w.Code, http.StatusBadRequest)
	}
}

// TestB2B_CreateQuotation_RejectsOversizedCurrency 币种超长被拒绝。
func TestB2B_CreateQuotation_RejectsOversizedCurrency(t *testing.T) {
	h := &B2BHandler{}
	r := newB2BTestRouter("POST", "/api/quotations", h.CreateQuotation)

	cur := strings.Repeat("U", maxQuotationCurrency+1)
	w := doJSON(r, "POST", "/api/quotations", `{"total_amount":"100.00","currency":"`+cur+`"}`)
	if w.Code != http.StatusBadRequest {
		t.Errorf("oversized currency: got status %d, want %d", w.Code, http.StatusBadRequest)
	}
}

// TestB2B_CreateCustomer_AcceptsNormalInput 正常客户输入通过长度校验。
func TestB2B_CreateCustomer_AcceptsNormalInput(t *testing.T) {
	h := &B2BHandler{}
	r := newB2BTestRouter("POST", "/api/customers", h.CreateCustomer)

	w := doJSON(r, "POST", "/api/customers",
		`{"company_name":"Acme Trading","country":"USA","email":"buy@acme.com","demand":"monthly 5000 pcs"}`)
	if w.Code == http.StatusBadRequest {
		t.Errorf("正常输入不应被校验拒绝: %s", w.Body.String())
	}
}
