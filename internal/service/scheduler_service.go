// Package service — Agent 定时调度（robfig/cron v3）。
//
// 设计依据：开发方案 V2 §5.3 — 启动时从 settings 表读取 cron 表达式，注册定时 Agent。
// 老板可在前端设置页修改定时（PUT /api/agents/schedule），立即生效（重启调度器）。
//
// 默认 schedule（settings 表无值时）：
//   - selection_cron: "0 9 * * *"   每天早 9 点跑选品 Agent
//   - sourcing_cron:  "0 10 * * *"  每天早 10 点跑采购 Agent
//
// 所有定时任务使用 default_model（AIService 自动路由）。
package service

import (
	"fmt"
	"log"
	"sync"

	"github.com/CainGao/trademind/internal/models"
	"github.com/CainGao/trademind/internal/repository"
	"github.com/robfig/cron/v3"
)

// SchedulerService 定时调度器。
type SchedulerService struct {
	mu             sync.Mutex
	cron           *cron.Cron
	agentSvc       *AgentService
	setting        *repository.SettingRepo
	schedule       map[models.AgentType]string // 当前生效的 cron 表达式
	dailyReportSvc *DailyReportService         // 日报服务（可选，延迟注入）
}

// NewSchedulerService 构造（不立即启动）。
func NewSchedulerService(agentSvc *AgentService, setting *repository.SettingRepo) *SchedulerService {
	return &SchedulerService{
		agentSvc: agentSvc,
		setting:  setting,
		schedule: map[models.AgentType]string{},
	}
}

// SetDailyReportService 延迟注入日报服务（避免 router 装配循环依赖）。
func (s *SchedulerService) SetDailyReportService(svc *DailyReportService) {
	s.dailyReportSvc = svc
}

// DefaultSchedule 默认定时（settings 表无值时用）。
var DefaultSchedule = map[models.AgentType]string{
	models.AgentSelection: "0 9 * * *",  // 每天 9:00
	models.AgentSourcing:  "0 10 * * *", // 每天 10:00
}

// settingKey 返回某 AgentType 在 settings 表里的 key。
func settingKey(t models.AgentType) string {
	return "agent_" + string(t) + "_cron"
}

// GetSchedule 获取当前定时（settings 优先，无值用默认）。
func (s *SchedulerService) GetSchedule(t models.AgentType) string {
	if v, ok := s.schedule[t]; ok {
		return v
	}
	if setting, err := s.setting.Get(settingKey(t)); err == nil && setting != nil && setting.Value != "" {
		return setting.Value
	}
	return DefaultSchedule[t]
}

// AllSchedule 返回所有 Agent 的定时配置（前端展示用）。
func (s *SchedulerService) AllSchedule() []AgentScheduleItem {
	types := []models.AgentType{models.AgentSelection, models.AgentSourcing}
	items := make([]AgentScheduleItem, 0, len(types))
	for _, t := range types {
		items = append(items, AgentScheduleItem{
			AgentType: string(t),
			Cron:      s.GetSchedule(t),
			Enabled:   true,
		})
	}
	return items
}

// AgentScheduleItem 单个调度项。
type AgentScheduleItem struct {
	AgentType string `json:"agent_type"`
	Cron      string `json:"cron"`
	Enabled   bool   `json:"enabled"`
}

// UpdateSchedule 更新某个 Agent 的定时（持久化到 settings + 重启调度器）。
func (s *SchedulerService) UpdateSchedule(t models.AgentType, cronExpr string) error {
	// 1. 校验表达式
	if _, err := cron.ParseStandard(cronExpr); err != nil {
		return fmt.Errorf("cron 表达式非法: %w", err)
	}
	// 2. 持久化到 settings
	if err := s.setting.Set(settingKey(t), cronExpr, false); err != nil {
		return fmt.Errorf("保存调度配置失败: %w", err)
	}
	s.schedule[t] = cronExpr
	// 3. 重启调度器
	return s.Restart()
}

// Start 启动调度器（从 settings 加载所有 cron 并注册）。
func (s *SchedulerService) Start() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.cron != nil {
		return nil // 已启动
	}

	s.cron = cron.New()

	// 注册选品 Agent
	if err := s.register(models.AgentSelection); err != nil {
		return err
	}
	// 注册采购 Agent
	if err := s.register(models.AgentSourcing); err != nil {
		return err
	}

	// 注册日报任务（每天 18:00），仅当注入了 dailyReportSvc
	if s.dailyReportSvc != nil {
		if _, err := s.cron.AddFunc("0 18 * * *", func() {
			log.Println("[scheduler] 触发日报生成")
			report, err := s.dailyReportSvc.Generate("", models.TriggerCron)
			if err != nil {
				log.Printf("[scheduler] 日报生成失败: %v", err)
				return
			}
			// 自动推送（如已配置 webhook）
			if err := s.dailyReportSvc.DeliverToFeishu(report.ID); err != nil {
				log.Printf("[scheduler] 日报飞书推送跳过: %v", err)
			} else {
				log.Printf("[scheduler] 日报 #%d 已推送到飞书", report.ID)
			}
		}); err != nil {
			return fmt.Errorf("注册日报任务失败: %w", err)
		}
		log.Println("[scheduler] 已注册 report | cron=0 18 * * * (每天 18:00)")
	}

	s.cron.Start()
	log.Println("[scheduler] 定时调度器已启动")
	return nil
}

// register 注册单个 Agent 的 cron 任务。
func (s *SchedulerService) register(t models.AgentType) error {
	expr := s.GetSchedule(t)
	entryID, err := s.cron.AddFunc(expr, func() {
		s.runAgent(t)
	})
	if err != nil {
		return fmt.Errorf("注册 %s 失败: %w", t, err)
	}
	log.Printf("[scheduler] 已注册 %s | cron=%s | entryID=%d", t, expr, entryID)
	return nil
}

// runAgent 实际执行 Agent（被 cron 触发）。
func (s *SchedulerService) runAgent(t models.AgentType) {
	log.Printf("[scheduler] 触发 %s Agent", t)
	var err error
	switch t {
	case models.AgentSelection:
		_, _, err = s.agentSvc.RunSelection(14, "", models.TriggerCron)
	case models.AgentSourcing:
		_, _, err = s.agentSvc.RunSourcing("", models.TriggerCron)
	}
	if err != nil {
		log.Printf("[scheduler] %s Agent 执行失败: %v", t, err)
	} else {
		log.Printf("[scheduler] %s Agent 执行完成", t)
	}
}

// Stop 停止调度器。
func (s *SchedulerService) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.cron != nil {
		ctx := s.cron.Stop()
		<-ctx.Done()
		s.cron = nil
		log.Println("[scheduler] 调度器已停止")
	}
}

// Restart 重启调度器（修改定时后调用）。
func (s *SchedulerService) Restart() error {
	s.Stop()
	return s.Start()
}
