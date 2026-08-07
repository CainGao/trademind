package handler

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

// init 测试初始化。
func init() {
	gin.SetMode(gin.TestMode)
}

// TestPaste_RejectsOversizedContent 验证粘贴文本超长被拒绝。
func TestPaste_RejectsOversizedContent(t *testing.T) {
	h := &KnowledgeHandler{} // nil service is OK — validation should fire first
	router := gin.New()
	router.POST("/api/knowledge/paste", h.Paste)

	body := `{"title":"big","content":"` + strings.Repeat("a", maxPasteTextLen+1) + `"}`
	req := httptest.NewRequest("POST", "/api/knowledge/paste", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("oversized paste: got status %d, want %d", w.Code, http.StatusBadRequest)
	}
	if !strings.Contains(w.Body.String(), "500KB") {
		t.Errorf("oversized paste: response should mention 500KB, got: %s", w.Body.String())
	}
}

// TestSearch_RejectsOversizedQuery 验证检索查询超长被拒绝。
func TestSearch_RejectsOversizedQuery(t *testing.T) {
	h := &KnowledgeHandler{}
	router := gin.New()
	router.POST("/api/knowledge/search", h.Search)

	body := `{"query":"` + strings.Repeat("a", maxSearchQueryLen+1) + `"}`
	req := httptest.NewRequest("POST", "/api/knowledge/search", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("oversized search query: got status %d, want %d", w.Code, http.StatusBadRequest)
	}
	if !strings.Contains(w.Body.String(), "10KB") {
		t.Errorf("oversized search query: response should mention 10KB, got: %s", w.Body.String())
	}
}

// TestChat_RejectsOversizedQuery 验证 RAG 对话查询超长被拒绝。
func TestChat_RejectsOversizedQuery(t *testing.T) {
	h := &KnowledgeHandler{}
	router := gin.New()
	router.POST("/api/knowledge/chat", h.Chat)

	body := `{"query":"` + strings.Repeat("a", maxChatQueryLen+1) + `"}`
	req := httptest.NewRequest("POST", "/api/knowledge/chat", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("oversized chat query: got status %d, want %d", w.Code, http.StatusBadRequest)
	}
	if !strings.Contains(w.Body.String(), "10KB") {
		t.Errorf("oversized chat query: response should mention 10KB, got: %s", w.Body.String())
	}
}

// TestChat_RejectsOversizedHistory 验证 RAG 对话历史超长被拒绝。
func TestChat_RejectsOversizedHistory(t *testing.T) {
	h := &KnowledgeHandler{}
	router := gin.New()
	router.POST("/api/knowledge/chat", h.Chat)

	body := `{"query":"hello","history":"` + strings.Repeat("a", maxChatHistoryLen+1) + `"}`
	req := httptest.NewRequest("POST", "/api/knowledge/chat", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("oversized chat history: got status %d, want %d", w.Code, http.StatusBadRequest)
	}
	if !strings.Contains(w.Body.String(), "50KB") {
		t.Errorf("oversized chat history: response should mention 50KB, got: %s", w.Body.String())
	}
}

// TestInputLengthConstants 验证长度常量值合理。
func TestInputLengthConstants(t *testing.T) {
	if maxPasteTextLen != 500*1024 {
		t.Errorf("maxPasteTextLen = %d, want %d", maxPasteTextLen, 500*1024)
	}
	if maxSearchQueryLen != 10*1024 {
		t.Errorf("maxSearchQueryLen = %d, want %d", maxSearchQueryLen, 10*1024)
	}
	if maxChatQueryLen != 10*1024 {
		t.Errorf("maxChatQueryLen = %d, want %d", maxChatQueryLen, 10*1024)
	}
	if maxChatHistoryLen != 50*1024 {
		t.Errorf("maxChatHistoryLen = %d, want %d", maxChatHistoryLen, 50*1024)
	}
}
