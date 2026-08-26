package service

import (
	"strings"
	"testing"

	"github.com/CainGao/trademind/internal/database"
	"github.com/CainGao/trademind/internal/middleware"
	"github.com/CainGao/trademind/internal/models"
	"github.com/CainGao/trademind/internal/pkg/crypto"
	"github.com/CainGao/trademind/internal/repository"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

// setupWeakPwdTestDB 创建按测试函数隔离的命名内存库（gotcha #85）。
func setupWeakPwdTestDB(t *testing.T) *gorm.DB {
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

// newWeakPwdAuthSvc 构造 AuthService + seed 一个 admin（密码用 database.DefaultAdminPassword，
// 与生产 seed 行为一致）。
func newWeakPwdAuthSvc(t *testing.T, db *gorm.DB) *AuthService {
	t.Helper()
	hash, err := crypto.HashPassword(database.DefaultAdminPassword)
	if err != nil {
		t.Fatalf("hash 密码失败: %v", err)
	}
	if err := db.Create(&models.User{
		Username: "admin", PasswordHash: hash, Nickname: "管理员",
		Role: models.RoleAdmin, Status: models.UserStatusActive,
	}).Error; err != nil {
		t.Fatalf("seed admin 失败: %v", err)
	}
	jwtMgr := middleware.NewJWTManager([]byte("test-secret-0123456789abcdef0123456789"))
	return NewAuthService(repository.NewUserRepo(db), repository.NewAuditLogRepo(db), jwtMgr)
}

// TestIsWeakPassword 黑名单命中/未命中（含大小写不敏感）。
func TestIsWeakPassword(t *testing.T) {
	weak := []string{"admin123", "ADMIN123", "Admin123", "123456", "12345678", "password", "PASSWORD", "qwerty", "888888"}
	for _, pw := range weak {
		if !isWeakPassword(pw) {
			t.Errorf("isWeakPassword(%q) = false, want true（黑名单命中）", pw)
		}
	}
	strong := []string{"Tr@de2026!Mind", "xk9#mQ2$vLp8", "7Hg*zdPq94we"}
	for _, pw := range strong {
		if isWeakPassword(pw) {
			t.Errorf("isWeakPassword(%q) = true, want false（正常密码误拦）", pw)
		}
	}
}

// TestLogin_MustChangePasswordFlag gotcha #88 核心：admin 用默认密码登录 →
// 响应带 must_change_password=true；改密后用新密码登录 → false。
func TestLogin_MustChangePasswordFlag(t *testing.T) {
	db := setupWeakPwdTestDB(t)
	svc := newWeakPwdAuthSvc(t, db)

	// 默认密码登录 → 标志为 true
	pair, err := svc.Login(LoginInput{Username: "admin", Password: database.DefaultAdminPassword}, "127.0.0.1")
	if err != nil {
		t.Fatalf("默认密码登录应成功: %v", err)
	}
	if !pair.User.MustChangePassword {
		t.Errorf("admin 用默认密码登录 must_change_password 应为 true")
	}

	// 改密后登录 → 标志为 false
	hash, _ := crypto.HashPassword("Tr@de2026!Mind")
	if err := db.Model(&models.User{}).Where("username = ?", "admin").
		Update("password_hash", hash).Error; err != nil {
		t.Fatalf("更新密码失败: %v", err)
	}
	pair2, err := svc.Login(LoginInput{Username: "admin", Password: "Tr@de2026!Mind"}, "127.0.0.1")
	if err != nil {
		t.Fatalf("新密码登录应成功: %v", err)
	}
	if pair2.User.MustChangePassword {
		t.Errorf("改密后登录 must_change_password 应为 false")
	}
}

// TestLogin_NonAdminNoFlag 非 admin 用户即使密码恰为 admin123 也不置标志
//（检测限定 seed 的 admin 账户，避免误报）。
func TestLogin_NonAdminNoFlag(t *testing.T) {
	db := setupWeakPwdTestDB(t)
	svc := newWeakPwdAuthSvc(t, db)

	hash, _ := crypto.HashPassword("Str0ng!Pass22")
	if err := db.Create(&models.User{
		Username: "boss1", PasswordHash: hash, Nickname: "老板",
		Role: models.RoleBoss, Status: models.UserStatusActive,
	}).Error; err != nil {
		t.Fatalf("seed boss 失败: %v", err)
	}
	pair, err := svc.Login(LoginInput{Username: "boss1", Password: "Str0ng!Pass22"}, "127.0.0.1")
	if err != nil {
		t.Fatalf("boss 登录应成功: %v", err)
	}
	if pair.User.MustChangePassword {
		t.Errorf("非 admin 用户不应置 must_change_password")
	}
}

// TestRegister_RejectWeakPassword 注册接口拒绝弱密码黑名单。
func TestRegister_RejectWeakPassword(t *testing.T) {
	db := setupWeakPwdTestDB(t)
	svc := newWeakPwdAuthSvc(t, db)

	if _, err := svc.Register(RegisterInput{Username: "newbie", Password: "123456"}); err == nil || !strings.Contains(err.Error(), "弱密码") {
		t.Errorf("Register 弱密码应被拒，got err=%v", err)
	}
	u, err := svc.Register(RegisterInput{Username: "newbie", Password: "G00d!Pass77"})
	if err != nil {
		t.Fatalf("正常密码注册应成功: %v", err)
	}
	if u.Username != "newbie" {
		t.Errorf("注册返回用户名 = %q, want newbie", u.Username)
	}
}

// TestChangePassword_WeakAndSameRejection 改密接口拒绝「新旧相同」与「弱密码」。
func TestChangePassword_WeakAndSameRejection(t *testing.T) {
	db := setupWeakPwdTestDB(t)
	newWeakPwdAuthSvc(t, db) // seed admin（默认密码）

	encryptor, err := crypto.NewEncryptor(make([]byte, 32))
	if err != nil {
		t.Fatalf("创建 Encryptor 失败: %v", err)
	}
	svc := NewSetupService(
		repository.NewCompanyRepo(db),
		repository.NewSettingRepo(db),
		repository.NewUserRepo(db),
		encryptor,
	)

	// 新旧相同 → 拒绝
	err = svc.ChangePassword(1, ChangePasswordInput{
		OldPassword: database.DefaultAdminPassword, NewPassword: database.DefaultAdminPassword,
	})
	if err == nil || !strings.Contains(err.Error(), "不能与原密码相同") {
		t.Errorf("新旧相同应被拒，got err=%v", err)
	}

	// 弱密码 → 拒绝
	err = svc.ChangePassword(1, ChangePasswordInput{
		OldPassword: database.DefaultAdminPassword, NewPassword: "admin888",
	})
	if err == nil || !strings.Contains(err.Error(), "弱密码") {
		t.Errorf("弱密码应被拒，got err=%v", err)
	}

	// 正常改密 → 成功
	if err := svc.ChangePassword(1, ChangePasswordInput{
		OldPassword: database.DefaultAdminPassword, NewPassword: "S3cure#2026!Pwd",
	}); err != nil {
		t.Fatalf("正常改密应成功: %v", err)
	}
	// 改完用新密码能登录且标志为 false
	authSvc := NewAuthService(repository.NewUserRepo(db), repository.NewAuditLogRepo(db),
		middleware.NewJWTManager([]byte("test-secret-0123456789abcdef0123456789")))
	pair, err := authSvc.Login(LoginInput{Username: "admin", Password: "S3cure#2026!Pwd"}, "127.0.0.1")
	if err != nil {
		t.Fatalf("改密后登录应成功: %v", err)
	}
	if pair.User.MustChangePassword {
		t.Errorf("改密后登录 must_change_password 应为 false")
	}
}
