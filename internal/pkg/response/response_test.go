package response

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// TestSuccessPage_NilSlice Ensures nil slice is serialized as [] not null (gotcha #45).
func TestSuccessPage_NilSlice(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)

	var nilSlice []string // nil slice
	SuccessPage(c, nilSlice, 0, 1, 20)

	var resp struct {
		Code int `json:"code"`
		Data struct {
			List  interface{} `json:"list"`
			Total int64       `json:"total"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if resp.Data.List == nil {
		t.Errorf("nil slice serialized as null, want [] — body: %s", w.Body.String())
	}
}

// TestSuccessPage_EmptySlice ensures empty (non-nil) slice stays as [].
func TestSuccessPage_EmptySlice(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)

	emptySlice := []int{}
	SuccessPage(c, emptySlice, 0, 1, 20)

	var resp struct {
		Data struct {
			List []int `json:"list"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if len(resp.Data.List) != 0 {
		t.Errorf("empty slice len = %d, want 0", len(resp.Data.List))
	}
}

// TestSuccessPage_NonNilSlice ensures actual data passes through correctly.
func TestSuccessPage_NonNilSlice(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)

	data := []string{"a", "b"}
	SuccessPage(c, data, 2, 1, 20)

	var resp struct {
		Data struct {
			List  []string `json:"list"`
			Total int64    `json:"total"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if len(resp.Data.List) != 2 {
		t.Errorf("list len = %d, want 2", len(resp.Data.List))
	}
	if resp.Data.Total != 2 {
		t.Errorf("total = %d, want 2", resp.Data.Total)
	}
}

// TestSuccess_NilSlice ensures Success() also converts nil slice to [] (gotcha #45 extension to Success).
func TestSuccess_NilSlice(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)

	var nilSlice []map[string]interface{} // nil slice — same type as stats handler returns
	Success(c, nilSlice)

	var resp struct {
		Code int             `json:"code"`
		Data json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if string(resp.Data) == "null" {
		t.Errorf("nil slice serialized as null, want [] — body: %s", w.Body.String())
	}
	if string(resp.Data) != "[]" {
		t.Errorf("nil slice serialized as %s, want []", string(resp.Data))
	}
}

// TestSuccess_NilData ensures Success(nil) still omits data (single-object endpoints).
func TestSuccess_NilData(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)

	Success(c, nil)

	// data has omitempty tag, so nil interface should omit the field
	var resp map[string]json.RawMessage
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if _, ok := resp["data"]; ok {
		t.Errorf("nil interface should omit data field, but got: %s", w.Body.String())
	}
}

// TestSuccess_EmptySlice ensures Success() keeps empty (non-nil) slice as [].
func TestSuccess_EmptySlice(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)

	emptySlice := []int{}
	Success(c, emptySlice)

	var resp struct {
		Data json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if string(resp.Data) != "[]" {
		t.Errorf("empty slice serialized as %s, want []", string(resp.Data))
	}
}
