// Package repository — 用户数据访问。
package repository

import (
	"github.com/CainGao/trademind/internal/models"
	"gorm.io/gorm"
)

// UserRepo 用户数据访问。
type UserRepo struct {
	BaseRepo
}

func NewUserRepo(db *gorm.DB) *UserRepo {
	return &UserRepo{BaseRepo{DB: db}}
}

// GetByUsername 按用户名查询。
func (r *UserRepo) GetByUsername(username string) (*models.User, error) {
	var u models.User
	if err := r.DB.Where("username = ?", username).First(&u).Error; err != nil {
		return nil, err
	}
	return &u, nil
}

// GetByID 按 ID 查询。
func (r *UserRepo) GetByID(id uint) (*models.User, error) {
	var u models.User
	if err := r.DB.First(&u, id).Error; err != nil {
		return nil, err
	}
	return &u, nil
}

// Create 创建用户。
func (r *UserRepo) Create(u *models.User) error {
	return r.DB.Create(u).Error
}

// List 分页列表（规范 V1.0 §3.3 分页参数）。
func (r *UserRepo) List(page, size int) ([]models.User, int64, error) {
	var list []models.User
	var total int64
	r.DB.Model(&models.User{}).Count(&total)
	err := r.DB.Order("created_at DESC").
		Offset((page - 1) * size).Limit(size).
		Find(&list).Error
	return list, total, err
}
