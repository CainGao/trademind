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

// DailyTrend 近 N 天每日事件数趋势（驾驶舱折线图用）。
// 返回 [{date: "2026-07-19", total: 42, browse: 20, search: 15, collect: 5, favorite: 2}, ...]
func (r *BehaviorRepo) DailyTrend(days int) ([]map[string]interface{}, error) {
	if days < 1 || days > 90 {
		days = 14
	}
	since := time.Now().AddDate(0, 0, -(days - 1))
	var rows []map[string]interface{}
	// SQLite date() 函数截取到天
	err := r.DB.Model(&models.BehaviorEvent{}).
		Select(`date(occurred_at) as date, COUNT(*) as total,
			SUM(CASE WHEN event_type='browse' THEN 1 ELSE 0 END) as browse,
			SUM(CASE WHEN event_type='search' THEN 1 ELSE 0 END) as search,
			SUM(CASE WHEN event_type='collect' THEN 1 ELSE 0 END) as collect,
			SUM(CASE WHEN event_type='favorite' THEN 1 ELSE 0 END) as favorite,
			SUM(CASE WHEN event_type='compare' THEN 1 ELSE 0 END) as compare,
			SUM(CASE WHEN event_type='export' THEN 1 ELSE 0 END) as export`).
		Where("occurred_at >= ?", since).
		Group("date(occurred_at)").
		Order("date ASC").
		Find(&rows).Error
	return rows, err
}

// TopKeywords 近 N 天最热门的搜索关键词 Top N（target_id 是关键词）。
func (r *BehaviorRepo) TopKeywords(days, limit int) ([]map[string]interface{}, error) {
	if limit < 1 || limit > 50 {
		limit = 10
	}
	since := time.Now().AddDate(0, 0, -days)
	var rows []map[string]interface{}
	err := r.DB.Model(&models.BehaviorEvent{}).
		Select("target_id as keyword, COUNT(*) as cnt").
		Where("event_type = 'search' AND occurred_at >= ?", since).
		Group("target_id").
		Order("cnt DESC").
		Limit(limit).
		Find(&rows).Error
	return rows, err
}

// Overview 全局统计概览（驾驶舱卡片用）。
type BehaviorOverview struct {
	Total       int64 `json:"total"`
	Last7Days   int64 `json:"last_7_days"`
	Last30Days  int64 `json:"last_30_days"`
	BrowseCnt   int64 `json:"browse_cnt"`
	SearchCnt   int64 `json:"search_cnt"`
	CollectCnt  int64 `json:"collect_cnt"`
}

// Overview 聚合。
func (r *BehaviorRepo) Overview() (*BehaviorOverview, error) {
	var ov BehaviorOverview
	r.DB.Model(&models.BehaviorEvent{}).Count(&ov.Total)
	r.DB.Model(&models.BehaviorEvent{}).Where("occurred_at >= ?", time.Now().AddDate(0, 0, -7)).Count(&ov.Last7Days)
	r.DB.Model(&models.BehaviorEvent{}).Where("occurred_at >= ?", time.Now().AddDate(0, 0, -30)).Count(&ov.Last30Days)
	r.DB.Model(&models.BehaviorEvent{}).Where("event_type = 'browse'").Count(&ov.BrowseCnt)
	r.DB.Model(&models.BehaviorEvent{}).Where("event_type = 'search'").Count(&ov.SearchCnt)
	r.DB.Model(&models.BehaviorEvent{}).Where("event_type = 'collect'").Count(&ov.CollectCnt)
	return &ov, nil
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
