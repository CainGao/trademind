package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/CainGao/trademind/internal/models"
	"github.com/CainGao/trademind/internal/repository"
	"github.com/CainGao/trademind/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

// setupB2CTestDB 创建内存 SQLite + AutoMigrate B2C 表。
func setupB2CTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("打开内存数据库失败: %v", err)
	}
	if err := db.AutoMigrate(&models.Store{}, &models.Listing{}, &models.Order{}); err != nil {
		t.Fatalf("AutoMigrate 失败: %v", err)
	}
	return db
}

// setupB2CHandler 构造 B2C handler + 注入 user_id=42 的中间件。
func setupB2CHandler(db *gorm.DB) (*B2CHandler, *gin.Engine) {
	storeRepo := repository.NewStoreRepo(db)
	listingRepo := repository.NewListingRepo(db)
	orderRepo := repository.NewOrderRepo(db)
	svc := service.NewB2CService(storeRepo, listingRepo, orderRepo)
	h := NewB2CHandler(svc)

	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("user_id", uint(42))
	})
	return h, r
}

// TestB2C_CreateStore_ClearsMassAssignment 验证创建店铺时客户端注入的 id/deleted_at/created_by 被清除。
func TestB2C_CreateStore_ClearsMassAssignment(t *testing.T) {
	db := setupB2CTestDB(t)
	h, r := setupB2CHandler(db)
	r.POST("/api/b2c/stores", h.CreateStore)

	// 客户端尝试注入 id=999, deleted_at, created_by=1
	body := `{"id":999,"name":"测试店铺","platform":"amazon","created_by":1,"deleted_at":"2025-01-01T00:00:00Z"}`
	req := httptest.NewRequest("POST", "/api/b2c/stores", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("创建店铺: got status %d, want %d, body: %s", w.Code, http.StatusCreated, w.Body.String())
	}

	// 验证返回数据中 id 不是 999
	var resp struct {
		Data models.Store `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("解析响应失败: %v", err)
	}
	if resp.Data.ID == 999 {
		t.Error("mass assignment 漏洞：客户端注入的 id=999 被接受")
	}
	if resp.Data.ID == 0 {
		t.Error("创建后 ID 不应为 0")
	}
	// created_by 应为 JWT 中的 42，而非客户端注入的 1
	if resp.Data.CreatedBy != 42 {
		t.Errorf("created_by: got %d, want 42（来自 JWT）", resp.Data.CreatedBy)
	}
	// deleted_at 应为空
	if resp.Data.DeletedAt.Valid {
		t.Error("deleted_at 应为空，客户端注入的值未被清除")
	}
}

// TestB2C_UpdateStore_ClearsDeletedAt 验证更新店铺时客户端注入的 deleted_at 被清除。
func TestB2C_UpdateStore_ClearsDeletedAt(t *testing.T) {
	db := setupB2CTestDB(t)
	h, r := setupB2CHandler(db)

	// 先创建一条店铺
	store := &models.Store{Name: "原店铺", Platform: "amazon", Status: "active"}
	db.Create(store)

	r.PUT("/api/b2c/stores/:id", h.UpdateStore)

	// 客户端尝试注入 deleted_at 软删除
	body := `{"name":"被篡改的店铺","deleted_at":"2025-01-01T00:00:00Z"}`
	req := httptest.NewRequest("PUT", "/api/b2c/stores/1", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("更新店铺: got status %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	// 验证记录未被软删除（能正常查到）
	var count int64
	db.Model(&models.Store{}).Where("id = ?", store.ID).Count(&count)
	if count != 1 {
		t.Error("店铺被意外软删除：客户端注入的 deleted_at 未被清除")
	}
}

// TestB2C_CreateListing_ClearsMassAssignment 验证创建上架时客户端注入的 id/created_by 被清除。
func TestB2C_CreateListing_ClearsMassAssignment(t *testing.T) {
	db := setupB2CTestDB(t)
	h, r := setupB2CHandler(db)
	// 先创建一个店铺供 FK 使用
	db.Create(&models.Store{Name: "测试", Platform: "amazon", Status: "active"})
	r.POST("/api/b2c/listings", h.CreateListing)

	body := `{"id":888,"store_id":1,"platform_sku":"SKU001","title":"测试上架","created_by":1}`
	req := httptest.NewRequest("POST", "/api/b2c/listings", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("创建上架: got status %d, want %d, body: %s", w.Code, http.StatusCreated, w.Body.String())
	}

	var resp struct {
		Data models.Listing `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("解析响应失败: %v", err)
	}
	if resp.Data.ID == 888 {
		t.Error("mass assignment 漏洞：客户端注入的 id=888 被接受")
	}
	if resp.Data.CreatedBy != 42 {
		t.Errorf("created_by: got %d, want 42（来自 JWT）", resp.Data.CreatedBy)
	}
}

// TestB2C_CreateOrder_ClearsMassAssignment 验证创建订单时客户端注入的 id 被清除。
func TestB2C_CreateOrder_ClearsMassAssignment(t *testing.T) {
	db := setupB2CTestDB(t)
	h, r := setupB2CHandler(db)
	db.Create(&models.Store{Name: "测试", Platform: "amazon", Status: "active"})
	r.POST("/api/b2c/orders", h.CreateOrder)

	// 客户端尝试注入 id=777 和 deleted_at
	body := `{"id":777,"store_id":1,"platform_order_no":"ORD-001","status":"pending","amount":"99.99","deleted_at":"2025-01-01T00:00:00Z"}`
	req := httptest.NewRequest("POST", "/api/b2c/orders", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("创建订单: got status %d, want %d, body: %s", w.Code, http.StatusCreated, w.Body.String())
	}

	var resp struct {
		Data models.Order `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("解析响应失败: %v", err)
	}
	if resp.Data.ID == 777 {
		t.Error("mass assignment 漏洞：客户端注入的 id=777 被接受")
	}
	if resp.Data.DeletedAt.Valid {
		t.Error("deleted_at 应为空，客户端注入的值未被清除")
	}
}
