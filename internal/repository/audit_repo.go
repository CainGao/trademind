// Package repository — 审计日志数据访问。
package repository

import (
	"log"
	"time"

	"github.com/CainGao/trademind/internal/models"
	"github.com/CainGao/trademind/internal/pkg/pagination"
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
	entry := models.AuditLog{
		UserID:     userID,
		Action:     action,
		Resource:   resource,
		ResourceID: resourceID,
		Detail:     detail,
		IP:         ip,
	}
	// 容错：失败仅打印不抛错
	if err := r.DB.Create(&entry).Error; err != nil {
		log.Printf("[audit] 写入审计日志失败 user_id=%d action=%s: %v", userID, action, err)
	}
}

// ListByUser 查某用户的审计记录。
func (r *AuditLogRepo) ListByUser(userID uint, page, size int) ([]models.AuditLog, int64, error) {
	page, size = pagination.Normalize(page, size)
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
	page, size = pagination.Normalize(page, size)
	var logs []models.AuditLog
	var total int64
	q := r.DB.Model(&models.AuditLog{}).Where("action = ? AND created_at >= ?", action, since)
	q.Count(&total)
	err := q.Order("created_at DESC").
		Offset((page - 1) * size).Limit(size).Find(&logs).Error
	return logs, total, err
}

// AuditLogFilter 审计日志查询筛选（全部字段可空 = 不过滤）。
type AuditLogFilter struct {
	UserID  *uint
	Action  string // 精确匹配（login/login_failed/create/...）
	StartAt *time.Time
	EndAt   *time.Time
}

// ListAll 管理员查看全量审计日志（筛选 + 分页，按时间倒序）。
func (r *AuditLogRepo) ListAll(f AuditLogFilter, page, size int) ([]models.AuditLog, int64, error) {
	page, size = pagination.Normalize(page, size)
	var logs []models.AuditLog
	var total int64
	q := r.DB.Model(&models.AuditLog{})
	if f.UserID != nil {
		q = q.Where("user_id = ?", *f.UserID)
	}
	if f.Action != "" {
		q = q.Where("action = ?", f.Action)
	}
	if f.StartAt != nil {
		q = q.Where("created_at >= ?", *f.StartAt)
	}
	if f.EndAt != nil {
		q = q.Where("created_at < ?", *f.EndAt)
	}
	q.Count(&total)
	err := q.Order("created_at DESC").
		Offset((page - 1) * size).Limit(size).Find(&logs).Error
	return logs, total, err
}
