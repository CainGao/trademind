// Package repository — 审计日志数据访问。
package repository

import (
	"time"

	"github.com/CainGao/trademind/internal/models"
	"gorm.io/gorm"
)

// AuditLogRepo 审计日志。规范 §6.7: 敏感操作必记。
type AuditLogRepo struct {
	BaseRepo
}

func NewAuditLogRepo(db *gorm.DB) *AuditLogRepo {
	return &AuditLogRepo{BaseRepo{DB: db}}
}

// Log 写入审计日志（异步友好：失败只影响日志，不阻断主流程）。
func (r *AuditLogRepo) Log(userID uint, action, resource string, resourceID *uint, detail, ip string) {
	log := models.AuditLog{
		UserID:     userID,
		Action:     action,
		Resource:   resource,
		ResourceID: resourceID,
		Detail:     detail,
		IP:         ip,
	}
	// 容错：失败仅打印不抛错
	if err := r.DB.Create(&log).Error; err != nil {
		// TODO: 接 logger 打 warn
		_ = err
	}
}

// ListByUser 查某用户的审计记录。
func (r *AuditLogRepo) ListByUser(userID uint, page, size int) ([]models.AuditLog, int64, error) {
	var logs []models.AuditLog
	var total int64
	q := r.DB.Model(&models.AuditLog{}).Where("user_id = ?", userID)
	q.Count(&total)
	err := q.Order("created_at DESC").
		Offset((page - 1) * size).Limit(size).Find(&logs).Error
	return logs, total, err
}

// ListByAction 查某类操作。
func (r *AuditLogRepo) ListByAction(action string, since time.Time, page, size int) ([]models.AuditLog, int64, error) {
	var logs []models.AuditLog
	var total int64
	q := r.DB.Model(&models.AuditLog{}).Where("action = ? AND created_at >= ?", action, since)
	q.Count(&total)
	err := q.Order("created_at DESC").
		Offset((page - 1) * size).Limit(size).Find(&logs).Error
	return logs, total, err
}
