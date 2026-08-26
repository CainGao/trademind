// Package database — 种子数据（首次启动注入）。
//
// 规范 V1.0 §1.7: 默认管理员 / JWT Secret / Agent Prompts 等首次启动注入。
package database

import (
	"crypto/rand"
	"encoding/base64"

	"github.com/CainGao/trademind/internal/models"
	"github.com/CainGao/trademind/internal/pkg/crypto"
	"gorm.io/gorm"
)

// Seed 首次启动注入默认数据。幂等：已存在则跳过。
func Seed(db *gorm.DB) error {
	// 1. JWT Secret（运行时随机生成 32 字节，持久化到 settings）
	if err := seedJWTSecret(db); err != nil {
		return err
	}

	// 2. 默认管理员账号
	if err := seedAdmin(db); err != nil {
		return err
	}

	return nil
}

// seedJWTSecret 如果 settings 表没有 jwt_secret 则生成并持久化。
func seedJWTSecret(db *gorm.DB) error {
	var count int64
	db.Model(&models.Setting{}).Where("`key` = ?", "jwt_secret").Count(&count)
	if count > 0 {
		return nil
	}
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return err
	}
	secret := base64.StdEncoding.EncodeToString(raw)
	return db.Create(&models.Setting{
		Key:       "jwt_secret",
		Value:     secret,
		Encrypted: boolPtr(false), // JWT secret 本身就是随机串，不需再加密
	}).Error
}

// DefaultAdminPassword 首启 seed 的默认管理员密码。
// 导出供 auth 层做「仍在使用默认密码」检测（改密提醒横幅），
// 避免魔法字符串在 seed/auth 两处重复后失同步。
const DefaultAdminPassword = "admin123"

// seedAdmin 默认管理员（用户名 admin，密码 admin123，首次登录强制改）。
func seedAdmin(db *gorm.DB) error {
	var count int64
	db.Model(&models.User{}).Count(&count)
	if count > 0 {
		return nil // 已有用户，跳过
	}
	hash, err := crypto.HashPassword(DefaultAdminPassword)
	if err != nil {
		return err
	}
	admin := models.User{
		Username:     "admin",
		PasswordHash: hash,
		Nickname:     "管理员",
		Role:         models.RoleAdmin,
		Status:       models.UserStatusActive,
	}
	return db.Create(&admin).Error
}

// GetJWTSecret 从 settings 读取 JWT Secret。
func GetJWTSecret(db *gorm.DB) ([]byte, error) {
	var s models.Setting
	if err := db.Where("`key` = ?", "jwt_secret").First(&s).Error; err != nil {
		return nil, err
	}
	decoded, err := base64.StdEncoding.DecodeString(s.Value)
	if err != nil {
		return nil, err
	}
	return decoded, nil
}

func boolPtr(b bool) *bool { return &b }
