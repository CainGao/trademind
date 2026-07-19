// Package repository — 企业信息 + 系统配置（settings）数据访问。
package repository

import (
	"github.com/CainGao/trademind/internal/models"
	"gorm.io/gorm"
)

// CompanyRepo 企业信息（单条记录）。
type CompanyRepo struct {
	BaseRepo
}

func NewCompanyRepo(db *gorm.DB) *CompanyRepo {
	return &CompanyRepo{BaseRepo{DB: db}}
}

// Get 读取企业信息（第一条记录）。
func (r *CompanyRepo) Get() (*models.Company, error) {
	var c models.Company
	if err := r.DB.First(&c).Error; err != nil {
		return nil, err
	}
	return &c, nil
}

// Save 创建或更新企业信息（单条记录 upsert）。
func (r *CompanyRepo) Save(c *models.Company) error {
	var existing models.Company
	err := r.DB.First(&existing).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return r.DB.Create(c).Error
		}
		return err
	}
	c.ID = existing.ID
	return r.DB.Save(c).Error
}

// SettingRepo 系统配置。规范 §6.2: 敏感字段（AI Key）AES 加密。
type SettingRepo struct {
	BaseRepo
}

func NewSettingRepo(db *gorm.DB) *SettingRepo {
	return &SettingRepo{BaseRepo{DB: db}}
}

// Get 按 key 读配置。不存在返回空字符串 + ErrRecordNotFound。
func (r *SettingRepo) Get(key string) (*models.Setting, error) {
	var s models.Setting
	if err := r.DB.Where("`key` = ?", key).First(&s).Error; err != nil {
		return nil, err
	}
	return &s, nil
}

// Set 写配置（upsert）。
func (r *SettingRepo) Set(key, value string, encrypted bool) error {
	var existing models.Setting
	err := r.DB.Where("`key` = ?", key).First(&existing).Error
	if err == gorm.ErrRecordNotFound {
		return r.DB.Create(&models.Setting{
			Key: key, Value: value, Encrypted: &encrypted,
		}).Error
	}
	if err != nil {
		return err
	}
	return r.DB.Model(&existing).Updates(map[string]interface{}{
		"value":     value,
		"encrypted": encrypted,
	}).Error
}

// GetMany 批量读。
func (r *SettingRepo) GetMany(keys []string) (map[string]string, error) {
	var settings []models.Setting
	if err := r.DB.Where("`key` IN ?", keys).Find(&settings).Error; err != nil {
		return nil, err
	}
	m := make(map[string]string, len(settings))
	for _, s := range settings {
		m[s.Key] = s.Value
	}
	return m, nil
}

// ListAll 全部配置（用于设置页面展示，敏感字段会过滤）。
func (r *SettingRepo) ListAll() ([]models.Setting, error) {
	var list []models.Setting
	if err := r.DB.Order("`key`").Find(&list).Error; err != nil {
		return nil, err
	}
	return list, nil
}
