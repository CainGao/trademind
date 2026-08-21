package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/CainGao/trademind/internal/middleware"
	"github.com/CainGao/trademind/internal/models"
	"github.com/CainGao/trademind/internal/pkg/crypto"
	"github.com/CainGao/trademind/internal/repository"
	"github.com/CainGao/trademind/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

// setupAuthTestDB 创建命名内存库（gotcha #73：DSN ? 位置要正确，避免磁盘垃圾文件）。
func setupAuthTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:auth_register_test?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("打开内存数据库失败: %v", err)
	}
	if err := db.AutoMigrate(&models.User{}, &models.AuditLog{}); err != nil {
		t.Fatalf("AutoMigrate 失败: %v", err)
	}
	return db
}

// setupAuthEngine 构造与 router.go 相同的公开/受保护双链路由由。
// 公开组：login/refresh；受保护组：JWT + RequireRole(admin) + register。
func setupAuthEngine(t *testing.T, db *gorm.DB) (*gin.Engine, *service.AuthService, *middleware.JWTManager) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	jwtMgr := middleware.NewJWTManager([]byte("test-secret-0123456789abcdef0123456789"))
	svc := service.NewAuthService(repository.NewUserRepo(db), repository.NewAuditLogRepo(db), jwtMgr)
	h := NewAuthHandler(svc, middleware.NewLoginLimiter(5, 5*time.Minute, 5*time.Minute))

	r := gin.New()
	api := r.Group("/api")

	// 公开组（与 router.go 一致：仅 login/refresh）
	authGroup := api.Group("/auth")
	h.RegisterRoutes(authGroup)

	// 受保护组（与 router.go 一致：register 需 JWT + admin）
	protected := api.Group("")
	protected.Use(middleware.JWT(jwtMgr))
	authAdmin := protected.Group("/auth")
	authAdmin.Use(middleware.RequireRole(models.RoleAdmin))
	h.RegisterAdminRoutes(authAdmin)

	return r, svc, jwtMgr
}

// tokenFor 用 JWTManager 直接签发测试 token（绕过登录）。
func tokenFor(t *testing.T, mgr *middleware.JWTManager, id uint, name string, role models.UserRole) string {
	t.Helper()
	tok, err := mgr.GenerateAccessToken(&models.User{BaseModel: models.BaseModel{ID: id}, Username: name, Role: role})
	if err != nil {
		t.Fatalf("签发 token 失败: %v", err)
	}
	return tok
}

// mustHash 测试辅助：生成 bcrypt 密码哈希。
func mustHash(t *testing.T, pwd string) string {
	t.Helper()
	h, err := crypto.HashPassword(pwd)
	if err != nil {
		t.Fatalf("哈希密码失败: %v", err)
	}
	return h
}

func doPost(r *gin.Engine, path, token, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest("POST", path, bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

// TestAuth_Register_NotPublic 验证未认证请求被 401 拒绝（gotcha #82 核心：register 不再公开）。
func TestAuth_Register_NotPublic(t *testing.T) {
	db := setupAuthTestDB(t)
	r, _, _ := setupAuthEngine(t, db)

	w := doPost(r, "/api/auth/register", "", `{"username":"evil","password":"123456","role":"admin"}`)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("无 token 注册: got status %d, want 401, body: %s", w.Code, w.Body.String())
	}
	// 确认没有用户被创建
	var cnt int64
	db.Model(&models.User{}).Where("username = ?", "evil").Count(&cnt)
	if cnt != 0 {
		t.Fatalf("未认证注册不应创建用户，但发现 %d 条", cnt)
	}
}

// TestAuth_Register_StaffForbidden 验证非 admin 角色被 403 拒绝。
func TestAuth_Register_StaffForbidden(t *testing.T) {
	db := setupAuthTestDB(t)
	r, _, mgr := setupAuthEngine(t, db)

	staffTok := tokenFor(t, mgr, 2, "staffer", models.RoleStaff)
	w := doPost(r, "/api/auth/register", staffTok, `{"username":"newbie","password":"123456"}`)
	if w.Code != http.StatusForbidden {
		t.Fatalf("staff 注册: got status %d, want 403, body: %s", w.Code, w.Body.String())
	}
}

// TestAuth_Register_AdminCreatesUser 验证 admin 可以正常创建用户。
func TestAuth_Register_AdminCreatesUser(t *testing.T) {
	db := setupAuthTestDB(t)
	r, _, mgr := setupAuthEngine(t, db)

	adminTok := tokenFor(t, mgr, 1, "admin", models.RoleAdmin)
	w := doPost(r, "/api/auth/register", adminTok, `{"username":"newbie","password":"123456","nickname":"新人"}`)
	if w.Code != http.StatusCreated {
		t.Fatalf("admin 注册: got status %d, want %d, body: %s", w.Code, http.StatusCreated, w.Body.String())
	}
	var u models.User
	if res := db.Where("username = ?", "newbie").First(&u); res.Error != nil {
		t.Fatalf("用户应已入库: %v", res.Error)
	}
	if u.Role != models.RoleStaff {
		t.Fatalf("未指定角色应默认 staff, got %s", u.Role)
	}
}

// TestAuth_Register_RejectsInvalidRole 验证 role 白名单（binding oneof）仍生效。
func TestAuth_Register_RejectsInvalidRole(t *testing.T) {
	db := setupAuthTestDB(t)
	r, _, mgr := setupAuthEngine(t, db)

	adminTok := tokenFor(t, mgr, 1, "admin", models.RoleAdmin)
	w := doPost(r, "/api/auth/register", adminTok, `{"username":"hacker","password":"123456","role":"superadmin"}`)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("非法 role: got status %d, want 400, body: %s", w.Code, w.Body.String())
	}
}

// TestAuth_LoginStillPublic 验证公开组 login/refresh 未被误伤（真实用户可无 token 登录成功）。
func TestAuth_LoginStillPublic(t *testing.T) {
	db := setupAuthTestDB(t)
	r, _, _ := setupAuthEngine(t, db)

	// 预置一个真实用户（密码 123456）
	u := &models.User{Username: "alice", PasswordHash: mustHash(t, "123456"), Role: models.RoleStaff, Status: models.UserStatusActive}
	if err := db.Create(u).Error; err != nil {
		t.Fatalf("预置用户失败: %v", err)
	}

	w := doPost(r, "/api/auth/login", "", `{"username":"alice","password":"123456"}`)
	if w.Code != http.StatusOK {
		t.Fatalf("login 公开可达: got status %d, want 200, body: %s", w.Code, w.Body.String())
	}
	var resp struct {
		Data struct {
			AccessToken string `json:"access_token"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("解析响应失败: %v", err)
	}
	if resp.Data.AccessToken == "" {
		t.Fatalf("login 应返回 access_token, body: %s", w.Body.String())
	}
}
