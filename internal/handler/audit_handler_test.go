package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/CainGao/trademind/internal/middleware"
	"github.com/CainGao/trademind/internal/models"
	"github.com/CainGao/trademind/internal/repository"
	"github.com/CainGao/trademind/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

// setupAuditTestDB 创建命名内存库（gotcha #73：按测试名隔离，避免同进程数据共享）。
func setupAuditTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("打开内存数据库失败: %v", err)
	}
	if err := db.AutoMigrate(&models.User{}, &models.AuditLog{}); err != nil {
		t.Fatalf("AutoMigrate 失败: %v", err)
	}
	return db
}

// setupAuditEngine 构造与 router.go 相同的 admin 鉴权链路。
func setupAuditEngine(t *testing.T, db *gorm.DB) (*gin.Engine, *middleware.JWTManager) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	jwtMgr := middleware.NewJWTManager([]byte("test-secret-0123456789abcdef0123456789"))
	svc := service.NewAuditService(repository.NewAuditLogRepo(db), repository.NewUserRepo(db))
	h := NewAuditHandler(svc)

	r := gin.New()
	api := r.Group("/api")
	protected := api.Group("")
	protected.Use(middleware.JWT(jwtMgr))
	admin := protected.Group("")
	admin.Use(middleware.RequireRole(models.RoleAdmin))
	h.RegisterRoutes(admin)
	return r, jwtMgr
}

func doGet(r *gin.Engine, path, token string) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, path, nil)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	r.ServeHTTP(w, req)
	return w
}

func auditBody(t *testing.T, w *httptest.ResponseRecorder) map[string]interface{} {
	t.Helper()
	var out map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &out); err != nil {
		t.Fatalf("解析响应失败: %v", err)
	}
	return out
}

// TestAuditLogs_RequiresAuth 未认证请求必须 401。
func TestAuditLogs_RequiresAuth(t *testing.T) {
	db := setupAuditTestDB(t)
	r, _ := setupAuditEngine(t, db)

	w := doGet(r, "/api/audit/logs", "")
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("未认证应 401，实际 %d", w.Code)
	}
}

// TestAuditLogs_RequiresAdmin 非 admin 角色必须 403。
func TestAuditLogs_RequiresAdmin(t *testing.T) {
	db := setupAuditTestDB(t)
	r, mgr := setupAuditEngine(t, db)

	staffTok := tokenFor(t, mgr, 2, "staff", models.RoleSales)
	w := doGet(r, "/api/audit/logs", staffTok)
	if w.Code != http.StatusForbidden {
		t.Fatalf("sales 角色应 403，实际 %d", w.Code)
	}
}

// TestAuditLogs_AdminListAndFilter admin 可查全量 + action 筛选 + username 映射。
func TestAuditLogs_AdminListAndFilter(t *testing.T) {
	db := setupAuditTestDB(t)
	r, mgr := setupAuditEngine(t, db)

	// 造数据：1 个用户 + 3 条审计日志（2 login + 1 login_failed）
	user := models.User{BaseModel: models.BaseModel{ID: 1}, Username: "boss_test", PasswordHash: "x", Role: models.RoleBoss}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("创建用户失败: %v", err)
	}
	auditRepo := repository.NewAuditLogRepo(db)
	auditRepo.Log(1, "login", "user", nil, "登录成功", "127.0.0.1")
	auditRepo.Log(1, "login_failed", "user", nil, "密码错误", "127.0.0.1")
	auditRepo.Log(1, "login", "user", nil, "登录成功", "192.168.1.5")

	adminTok := tokenFor(t, mgr, 1, "admin", models.RoleAdmin)

	// 全量
	w := doGet(r, "/api/audit/logs?page=1&page_size=10", adminTok)
	if w.Code != http.StatusOK {
		t.Fatalf("admin 查询应 200，实际 %d body=%s", w.Code, w.Body.String())
	}
	body := auditBody(t, w)
	data := body["data"].(map[string]interface{})
	if data["total"].(float64) != 3 {
		t.Fatalf("total 应 3，实际 %v", data["total"])
	}
	list := data["list"].([]interface{})
	first := list[0].(map[string]interface{})
	if first["username"] != "boss_test" {
		t.Fatalf("username 映射失败，实际 %v", first["username"])
	}

	// action 筛选
	w = doGet(r, "/api/audit/logs?action=login_failed", adminTok)
	body = auditBody(t, w)
	data = body["data"].(map[string]interface{})
	if data["total"].(float64) != 1 {
		t.Fatalf("action=login_failed 应 1 条，实际 %v", data["total"])
	}

	// user_id 筛选（不存在的用户 → 空列表）
	w = doGet(r, "/api/audit/logs?user_id=999", adminTok)
	body = auditBody(t, w)
	data = body["data"].(map[string]interface{})
	if data["total"].(float64) != 0 {
		t.Fatalf("user_id=999 应 0 条，实际 %v", data["total"])
	}
}

// TestAuditLogs_InputValidation 非法参数必须 400。
func TestAuditLogs_InputValidation(t *testing.T) {
	db := setupAuditTestDB(t)
	r, mgr := setupAuditEngine(t, db)
	adminTok := tokenFor(t, mgr, 1, "admin", models.RoleAdmin)

	cases := []struct {
		name string
		path string
	}{
		{"非法 user_id", "/api/audit/logs?user_id=abc"},
		{"零 user_id", "/api/audit/logs?user_id=0"},
		{"负 user_id", "/api/audit/logs?user_id=-5"},
		{"非法 start_date", "/api/audit/logs?start_date=2026/08/23"},
		{"非法 end_date", "/api/audit/logs?end_date=not-a-date"},
		{"action 超长", "/api/audit/logs?action=" + strings.Repeat("a", 51)},
	}
	for _, tc := range cases {
		w := doGet(r, tc.path, adminTok)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("%s 应 400，实际 %d body=%s", tc.name, w.Code, w.Body.String())
		}
	}
}

// TestAuditLogs_DateRange 日期闭区间筛选（end_date 含当天）。
func TestAuditLogs_DateRange(t *testing.T) {
	db := setupAuditTestDB(t)
	r, mgr := setupAuditEngine(t, db)

	// 直接造两条指定时间的日志（绕过 Log 的 time.Now）
	today := time.Now()
	yesterday := today.AddDate(0, 0, -1)
	old := yesterday.AddDate(0, 0, -30)
	db.Create(&models.AuditLog{UserID: 1, Action: "login", Resource: "user", Detail: "today"})
	db.Create(&models.AuditLog{UserID: 1, Action: "login", Resource: "user", Detail: "yesterday"})
	db.Create(&models.AuditLog{UserID: 1, Action: "login", Resource: "user", Detail: "old"})
	// 修正 created_at（Create 后手动 Updates）
	db.Model(&models.AuditLog{}).Where("detail = ?", "today").Update("created_at", today)
	db.Model(&models.AuditLog{}).Where("detail = ?", "yesterday").Update("created_at", yesterday)
	db.Model(&models.AuditLog{}).Where("detail = ?", "old").Update("created_at", old)

	adminTok := tokenFor(t, mgr, 1, "admin", models.RoleAdmin)

	// start_date=昨天 → 应含 today + yesterday 两条（old 被 30 天前排除）
	path := fmt.Sprintf("/api/audit/logs?start_date=%s", yesterday.Format("2006-01-02"))
	w := doGet(r, path, adminTok)
	if w.Code != http.StatusOK {
		t.Fatalf("日期筛选应 200，实际 %d", w.Code)
	}
	body := auditBody(t, w)
	data := body["data"].(map[string]interface{})
	if data["total"].(float64) != 2 {
		t.Fatalf("start_date=昨天 应 2 条，实际 %v", data["total"])
	}
}
