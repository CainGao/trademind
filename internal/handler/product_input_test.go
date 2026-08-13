package handler

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

// TestProduct_Create_RejectsOversizedDescription 验证商品描述超长被拒绝。
func TestProduct_Create_RejectsOversizedDescription(t *testing.T) {
	h := &ProductHandler{} // nil service — 校验在 service 之前触发
	r := gin.New()
	r.POST("/api/products", h.Create)

	desc := strings.Repeat("a", maxProductDesc+1)
	body := `{"name":"test","description":"` + desc + `"}`
	req := httptest.NewRequest("POST", "/api/products", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("oversized description: got status %d, want %d", w.Code, http.StatusBadRequest)
	}
}

// TestProduct_Create_RejectsOversizedName 验证商品名称超长被拒绝。
func TestProduct_Create_RejectsOversizedName(t *testing.T) {
	h := &ProductHandler{}
	r := gin.New()
	r.POST("/api/products", h.Create)

	name := strings.Repeat("n", maxProductName+1)
	body := `{"name":"` + name + `"}`
	req := httptest.NewRequest("POST", "/api/products", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("oversized name: got status %d, want %d", w.Code, http.StatusBadRequest)
	}
}

// TestProduct_Create_RejectsOversizedURL 验证来源 URL 超长被拒绝。
func TestProduct_Create_RejectsOversizedURL(t *testing.T) {
	h := &ProductHandler{}
	r := gin.New()
	r.POST("/api/products", h.Create)

	url := strings.Repeat("a", maxProductURL+1)
	body := `{"name":"test","source_url":"` + url + `"}`
	req := httptest.NewRequest("POST", "/api/products", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("oversized URL: got status %d, want %d", w.Code, http.StatusBadRequest)
	}
}

// TestProduct_Create_AcceptsNormalInput 验证正常长度输入不被拒绝（不因 service nil panic）。
// ShouldBindJSON 成功 → validateProductInput 通过 → 尝试调用 svc.Create（nil panic 被 gin Recovery 兜住）。
// 我们只验证 status != 400（即校验通过）。
func TestProduct_Create_AcceptsNormalInput(t *testing.T) {
	h := &ProductHandler{}
	r := gin.New()
	r.Use(gin.Recovery()) // 防 nil service panic
	r.POST("/api/products", h.Create)

	body := `{"name":"正常商品","description":"这是一个正常的商品描述"}`
	req := httptest.NewRequest("POST", "/api/products", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// 不应因输入校验返回 400（可能因 nil service 返回 500，那也说明校验通过了）
	if w.Code == http.StatusBadRequest {
		t.Errorf("正常输入不应被校验拒绝: %s", w.Body.String())
	}
}
