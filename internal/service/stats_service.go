// Package service — 行为数据分析（老板驾驶舱数据源）。
//
// 数据来源：Chrome 插件上报的员工行为（浏览/搜索/采集/收藏）。
// 职责：把这些"行为资产"聚合成老板看得懂的指标。
package service

import (
	"github.com/CainGao/trademind/internal/repository"
)

type StatsService struct {
	behaviorRepo *repository.BehaviorRepo
	productRepo  *repository.ProductRepo
	supplierRepo *repository.SupplierRepo
}

func NewStatsService(br *repository.BehaviorRepo, pr *repository.ProductRepo, sr *repository.SupplierRepo) *StatsService {
	return &StatsService{behaviorRepo: br, productRepo: pr, supplierRepo: sr}
}

// Dashboard 老板驾驶舱全量数据（一次请求拿全）。
type Dashboard struct {
	BehaviorOverview  *repository.BehaviorOverview  `json:"behavior_overview"`
	SupplierOverview  *repository.SupplierOverview  `json:"supplier_overview"`
	ProductTotal      int64                         `json:"product_total"`
	DailyTrend        []map[string]interface{}      `json:"daily_trend"`
	TopKeywords       []map[string]interface{}      `json:"top_keywords"`
	StatsByType       []map[string]interface{}      `json:"stats_by_type"`
}

// Dashboard 驾驶舱全量数据（默认近 14 天趋势）。
func (s *StatsService) Dashboard(trendDays int) (*Dashboard, error) {
	if trendDays < 1 || trendDays > 90 {
		trendDays = 14
	}
	behOv, err := s.behaviorRepo.Overview()
	if err != nil {
		return nil, err
	}
	supOv, err := s.supplierRepo.Overview()
	if err != nil {
		return nil, err
	}
	// 商品总数（List 取 total）
	prodList, err := s.productRepo.List(repository.ListParams{Page: 1, PageSize: 1})
	if err != nil {
		return nil, err
	}
	trend, err := s.behaviorRepo.DailyTrend(trendDays)
	if err != nil {
		return nil, err
	}
	kw, err := s.behaviorRepo.TopKeywords(14, 10)
	if err != nil {
		return nil, err
	}
	byType, err := s.behaviorRepo.StatsByType(14)
	if err != nil {
		return nil, err
	}
	return &Dashboard{
		BehaviorOverview: behOv,
		SupplierOverview: supOv,
		ProductTotal:     prodList.Total,
		DailyTrend:       ensureSlice(trend),
		TopKeywords:      ensureSlice(kw),
		StatsByType:      ensureSlice(byType),
	}, nil
}

// ensureSlice converts a nil slice to an empty slice (prevents JSON null).
func ensureSlice(rows []map[string]interface{}) []map[string]interface{} {
	if rows == nil {
		return []map[string]interface{}{}
	}
	return rows
}

// DailyTrend 趋势图单独接口（前端可指定天数）。
func (s *StatsService) DailyTrend(days int) ([]map[string]interface{}, error) {
	return s.behaviorRepo.DailyTrend(days)
}

// TopKeywords Top 搜索词。
func (s *StatsService) TopKeywords(days, limit int) ([]map[string]interface{}, error) {
	return s.behaviorRepo.TopKeywords(days, limit)
}
