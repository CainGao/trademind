// Package database — AES 加密密钥管理。
//
// 规范 §6.2: AI Key 等敏感字段 AES-256 加密存储。
// 密钥运行时随机生成 32 字节，持久化到 settings 表（aes_key 字段）。

package database

import (
	"crypto/rand"
	"encoding/base64"

	"github.com/CainGao/trademind/internal/models"
	"gorm.io/gorm"
)

const keyAESKey = "aes_key"

// GetOrCreateAESKey 从 settings 读 AES 密钥，不存在则生成并持久化。
// 返回 32 字节密钥。
func GetOrCreateAESKey(db *gorm.DB) ([]byte, error) {
	var s models.Setting
	err := db.Where("`key` = ?", keyAESKey).First(&s).Error
	if err == nil {
		// 已存在
		return base64.StdEncoding.DecodeString(s.Value)
	}
	if err != gorm.ErrRecordNotFound {
		return nil, err
	}
	// 不存在，生成新的
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return nil, err
	}
	encoded := base64.StdEncoding.EncodeToString(raw)
	t := true
	if err := db.Create(&models.Setting{
		Key: keyAESKey, Value: encoded, Encrypted: &t,
	}).Error; err != nil {
		return nil, err
	}
	return raw, nil
}
