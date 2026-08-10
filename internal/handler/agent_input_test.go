package handler

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

// TestAnalyzeEmail_RejectsOversizedSubject 验证邮件主题超长被拒绝。
func TestAnalyzeEmail_RejectsOversizedSubject(t *testing.T) {
	h := &AgentHandler{} // nil service — validation fires before service call
	router := gin.New()
	router.POST("/api/agents/analyze-email", h.AnalyzeEmail)

	body := `{"subject":"` + strings.Repeat("a", maxEmailSubjectLen+1) + `","content":"hello"}`
	req := httptest.NewRequest("POST", "/api/agents/analyze-email", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("oversized email subject: got status %d, want %d", w.Code, http.StatusBadRequest)
	}
	if !strings.Contains(w.Body.String(), "1KB") {
		t.Errorf("oversized email subject: response should mention 1KB, got: %s", w.Body.String())
	}
}

// TestAnalyzeEmail_RejectsOversizedContent 验证邮件正文超长被拒绝。
func TestAnalyzeEmail_RejectsOversizedContent(t *testing.T) {
	h := &AgentHandler{}
	router := gin.New()
	router.POST("/api/agents/analyze-email", h.AnalyzeEmail)

	body := `{"subject":"test","content":"` + strings.Repeat("a", maxEmailContentLen+1) + `"}`
	req := httptest.NewRequest("POST", "/api/agents/analyze-email", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("oversized email content: got status %d, want %d", w.Code, http.StatusBadRequest)
	}
	if !strings.Contains(w.Body.String(), "50KB") {
		t.Errorf("oversized email content: response should mention 50KB, got: %s", w.Body.String())
	}
}

// TestAnalyzeReviews_RejectsOversizedReviews 验证评论内容超长被拒绝。
func TestAnalyzeReviews_RejectsOversizedReviews(t *testing.T) {
	h := &AgentHandler{}
	router := gin.New()
	router.POST("/api/agents/analyze-reviews", h.AnalyzeReviews)

	body := `{"reviews":"` + strings.Repeat("a", maxReviewsLen+1) + `"}`
	req := httptest.NewRequest("POST", "/api/agents/analyze-reviews", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("oversized reviews: got status %d, want %d", w.Code, http.StatusBadRequest)
	}
	if !strings.Contains(w.Body.String(), "50KB") {
		t.Errorf("oversized reviews: response should mention 50KB, got: %s", w.Body.String())
	}
}

// TestOptimizeListing_RejectsInvalidPlatform 验证不支持的平台被拒绝。
func TestOptimizeListing_RejectsInvalidPlatform(t *testing.T) {
	h := &AgentHandler{}
	router := gin.New()
	router.POST("/api/agents/optimize-listing", h.OptimizeListing)

	req := httptest.NewRequest("POST", "/api/agents/optimize-listing?product_id=1&platform=hacked", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("invalid platform: got status %d, want %d", w.Code, http.StatusBadRequest)
	}
	if !strings.Contains(w.Body.String(), "不支持的平台") {
		t.Errorf("invalid platform: response should mention 不支持的平台, got: %s", w.Body.String())
	}
}

// TestOptimizeListing_AcceptsValidPlatform 验证合法平台通过校验（到达 service 层后 nil panic 预期）。
func TestOptimizeListing_AcceptsValidPlatform(t *testing.T) {
	h := &AgentHandler{}
	router := gin.New()
	router.POST("/api/agents/optimize-listing", h.OptimizeListing)

	for _, p := range []string{"amazon", "shopify", "tiktok", "temu"} {
		func() {
			defer func() {
				// 合法平台通过校验后会到达 nil service 导致 panic — 这是预期的。
				// 只要不返回"不支持的平台"错误就算通过。
				_ = recover()
			}()
			req := httptest.NewRequest("POST", "/api/agents/optimize-listing?product_id=1&platform="+p, nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
			if strings.Contains(w.Body.String(), "不支持的平台") {
				t.Errorf("valid platform %s was rejected: %s", p, w.Body.String())
			}
		}()
	}
}
