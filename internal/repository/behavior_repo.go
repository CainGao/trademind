// Package repository — 行为事件数据访问。
//
// Chrome 插件上报的浏览/搜索/采集事件都进这里，是「行为数据资产化」的原始库。
package repository

import (
	"time"

	"github.com/CainGao/trademind/internal/models"
	"gorm.io/gorm"
)

// BehaviorRepo 行为事件数据访问。
type BehaviorRepo struct {
	BaseRepo
}

func NewBehaviorRepo(db *gorm.DB) *BehaviorRepo {
	return &BehaviorRepo{BaseRepo{DB: db}}
}

// Create 写入单条行为事件。
func (r *BehaviorRepo) Create(e *models.BehaviorEvent) error {
	return r.DB.Create(e).Error
}

// CreateBatch 批量写入（插件离线批量上报时用）。
func (r *BehaviorRepo) CreateBatch(events []models.BehaviorEvent) error {
	if len(events) == 0 {
		return nil
	}
	return r.DB.CreateInBatches(events, 100).Error
}

// StatsByType 按事件类型聚合近 N 天的统计（老板驾驶舱用）。
// 返回 [{event_type, cnt}]。
func (r *BehaviorRepo) StatsByType(days int) ([]map[string]interface{}, error) {
	since := time.Now().AddDate(0, 0, -days)
	var rows []map[string]interface{}
	err := r.DB.Model(&models.BehaviorEvent{}).
		Select("event_type, COUNT(*) as cnt").
		Where("occurred_at >= ?", since).
		Group("event_type").
		Order("cnt DESC").
		Find(&rows).Error
	return rows, err
}

// RecentByUser 某用户最近 N 条行为。
func (r *BehaviorRepo) RecentByUser(userID uint, limit int) ([]models.BehaviorEvent, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	var events []models.BehaviorEvent
	err := r.DB.Where("user_id = ?", userID).
		Order("occurred_at DESC").
		Limit(limit).
		Find(&events).Error
	return events, err
}
