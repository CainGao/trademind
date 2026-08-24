package handler

// gotcha #87 测试：/api/setup/* 写操作端点必须 JWT + Admin（设计文档声明）。
// 此前只挂了 JWT + SetupGuard，staff 角色可越权覆盖全局配置（AI Key/场景/企业信息）。
// 测试引擎镜像 router.go 的装配：公开组 GET /setup/status；受保护组 JWT → SetupGuard → RequireRole(admin) → 写操作。

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/CainGao/trademind/internal/middleware"
	"github.com/CainGao/trademind/internal/models"
	"github.com/CainGao/trademind/internal/pkg/crypto"
	"github.com/CainGao/trademind/internal/repository"
	"github.com/CainGao/trademind/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

// setupWizardTestDB 创建按测试函数隔离的命名内存库（gotcha #85）。
func setupWizardTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := "file:" + t.Name() + "?mode=memory&cache=shared"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("打开内存数据库失败: %v", err)
	}
	if err := db.AutoMigrate(&models.User{}, &models.AuditLog{}, &models.Setting{}, &models.Company{}); err != nil {
		t.Fatalf("AutoMigrate 失败: %v", err)
	}
	return db
}

// setupWizardEngine 构造与 router.go 一致的双链路引擎。
func setupWizardEngine(t *testing.T, db *gorm.DB) (*gin.Engine, *middleware.JWTManager) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	jwtMgr := middleware.NewJWTManager([]byte("test-secret-0123456789abcdef0123456789"))
	encryptor, err := crypto.NewEncryptor(make([]byte, 32))
	if err != nil {
		t.Fatalf("创建 Encryptor 失败: %v", err)
	}
	svc := service.NewSetupService(
		repository.NewCompanyRepo(db),
		repository.NewSettingRepo(db),
		repository.NewUserRepo(db),
		encryptor,
	)
	h := NewSetupHandler(svc)

	r := gin.New()
	api := r.Group("/api")

	// 公开组（与 router.go 一致：仅 status 查询）
	api.GET("/setup/status", h.Status)

	// 受保护组（与 router.go 一致：JWT → SetupGuard → RequireRole(admin) → 写操作）
	protected := api.Group("")
	protected.Use(middleware.JWT(jwtMgr))
	protected.Use(middleware.SetupGuard(svc))
	setupAdmin := protected.Group("")
	setupAdmin.Use(middleware.RequireRole(models.RoleAdmin))
	h.RegisterRoutes(setupAdmin)

	return r, jwtMgr
}

// TestSetup_Write_Unauthenticated401 未认证请求被 401 拒绝。
func TestSetup_Write_Unauthenticated401(t *testing.T) {
	db := setupWizardTestDB(t)
	r, _ := setupWizardEngine(t, db)

	w := doPost(r, "/api/setup/company", "", `{"name":"Evil Corp"}`)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("无 token 写 setup: got status %d, want 401, body: %s", w.Code, w.Body.String())
	}
	var cnt int64
	db.Model(&models.Company{}).Count(&cnt)
	if cnt != 0 {
		t.Fatalf("未认证请求不应写入企业信息，但发现 %d 条", cnt)
	}
}

// TestSetup_Write_StaffForbidden403 staff 角色被 403 拒绝（gotcha #87 核心：
// 修复前 staff 可覆盖 AI Key / 场景 / 企业信息等全局配置）。
func TestSetup_Write_StaffForbidden403(t *testing.T) {
	db := setupWizardTestDB(t)
	r, mgr := setupWizardEngine(t, db)

	staffTok := tokenFor(t, mgr, 2, "staffer", models.RoleStaff)
	cases := []struct {
		path string
		body string
	}{
		{"/api/setup/company", `{"name":"Evil Corp"}`},
		{"/api/setup/scenario", `{"scenario":"b2c"}`},
		{"/api/setup/ai-key", `{"deepseek_api_key":"sk-evil"}`},
		{"/api/setup/complete", `{}`},
	}
	for _, tc := range cases {
		w := doPost(r, tc.path, staffTok, tc.body)
		if w.Code != http.StatusForbidden {
			t.Fatalf("staff POST %s: got status %d, want 403, body: %s", tc.path, w.Code, w.Body.String())
		}
	}
	// 确认没有任何 settings 被写入
	var cnt int64
	db.Model(&models.Setting{}).Where("`key` = ?", "setup_company_done").Count(&cnt)
	if cnt != 0 {
		t.Fatalf("staff 请求不应写入 setup 标记，但发现 %d 条", cnt)
	}
}

// TestSetup_Write_AdminAllowed admin 角色通过角色检查并正常写入
//（同时验证首启未完成时 SetupGuard 放行 /api/setup/* 前缀，向导流程不受影响）。
func TestSetup_Write_AdminAllowed(t *testing.T) {
	db := setupWizardTestDB(t)
	r, mgr := setupWizardEngine(t, db)

	adminTok := tokenFor(t, mgr, 1, "admin", models.RoleAdmin)
	w := doPost(r, "/api/setup/company", adminTok, `{"name":"测试贸易公司","industry":"五金","country":"中国"}`)
	if w.Code != http.StatusOK {
		t.Fatalf("admin 写 setup/company: got status %d, want 200, body: %s", w.Code, w.Body.String())
	}
	var c models.Company
	if res := db.Where("name = ?", "测试贸易公司").First(&c); res.Error != nil {
		t.Fatalf("企业信息应已入库: %v", res.Error)
	}
	// 标记也应写入
	var s models.Setting
	if res := db.Where("`key` = ?", "setup_company_done").First(&s); res.Error != nil {
		t.Fatalf("setup_company_done 标记应已写入: %v", res.Error)
	}
}

// TestSetup_StatusStillPublic status 查询保持公开（前端守卫依赖，首启未完成时也要能查）。
func TestSetup_StatusStillPublic(t *testing.T) {
	db := setupWizardTestDB(t)
	r, _ := setupWizardEngine(t, db)

	req := httptest.NewRequest("GET", "/api/setup/status", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("GET /api/setup/status 公开可达: got status %d, want 200, body: %s", w.Code, w.Body.String())
	}
}
