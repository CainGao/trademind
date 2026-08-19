// Package repository — Agent 运行记录数据访问。
//
// 所有 Agent 执行都写入 agent_runs 表，便于：
//   - 老板查看 Agent 工作历史（前端 Agent 任务页）
//   - 成本核算（tokens 累加）
//   - 故障排查（status=failed 的记录）
package repository

import (
	"time"

	"github.com/CainGao/trademind/internal/models"
	"gorm.io/gorm"
)

// AgentRepo Agent 运行记录仓库。
type AgentRepo struct {
	BaseRepo
}

// NewAgentRepo 构造。
func NewAgentRepo(db *gorm.DB) *AgentRepo {
	return &AgentRepo{BaseRepo{DB: db}}
}

// Create 写入一条 Agent 运行记录。
func (r *AgentRepo) Create(run *models.AgentRun) error {
	return r.DB.Create(run).Error
}

// Update 更新（用于运行结束后回写 output/status/finished_at）。
func (r *AgentRepo) Update(run *models.AgentRun) error {
	return r.DB.Save(run).Error
}

// List 列表（按 started_at 倒序）。
func (r *AgentRepo) List(page, pageSize int, agentType string) (items []models.AgentRun, total int64, err error) {
	q := r.DB.Model(&models.AgentRun{})
	if agentType != "" {
		q = q.Where("agent_type = ?", agentType)
	}
	if err = q.Count(&total).Error; err != nil {
		return
	}
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	err = q.Order("started_at DESC").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&items).Error
	return
}

// LatestByType 取某类型最近一次运行（前端"上次运行"展示）。
func (r *AgentRepo) LatestByType(agentType string) (*models.AgentRun, error) {
	var run models.AgentRun
	err := r.DB.Where("agent_type = ?", agentType).
		Order("started_at DESC").
		First(&run).Error
	if err != nil {
		return nil, err
	}
	return &run, nil
}

// CleanupZombieRuns 把所有 status=running 的运行记录标记为 failed（进程启动时调用）。
// 进程重启/崩溃时正在执行的 Agent goroutine 已消失，其运行记录会永久卡在 running
// 状态（僵尸记录）。启动时统一收敛为 failed + 补记 finished_at，保证状态机闭合。
// 返回受影响行数。
func (r *AgentRepo) CleanupZombieRuns() (int64, error) {
	res := r.DB.Model(&models.AgentRun{}).
		Where("status = ?", models.AgentRunRunning).
		Updates(map[string]interface{}{
			"status":      models.AgentRunFailed,
			"output":      "进程重启，任务中断",
			"finished_at": time.Now(),
		})
	return res.RowsAffected, res.Error
}
