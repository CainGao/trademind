package repository

import (
	"testing"
	"time"

	"github.com/CainGao/trademind/internal/models"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

// setupAgentCleanupTestDB 创建隔离的内存 SQLite + 迁移 agent_runs 表。
// DSN 写法注意 gotcha #73：? 必须在 mode=memory 之前，否则会在磁盘产生垃圾文件。
func setupAgentCleanupTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:agent_cleanup_test?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("打开内存数据库失败: %v", err)
	}
	if err := db.AutoMigrate(&models.AgentRun{}); err != nil {
		t.Fatalf("AutoMigrate 失败: %v", err)
	}
	return db
}

// TestCleanupZombieRuns 验证僵尸 running 记录被收敛为 failed，
// 且 done/failed 等已结束的记录不受影响。
func TestCleanupZombieRuns(t *testing.T) {
	db := setupAgentCleanupTestDB(t)
	repo := NewAgentRepo(db)

	// 造 3 条记录：1 条僵尸 running + 1 条 done + 1 条 failed
	zombie := &models.AgentRun{
		AgentType:   models.AgentSelection,
		TriggeredBy: models.TriggerCron,
		Status:      models.AgentRunRunning,
		StartedAt:   time.Now().Add(-time.Hour),
	}
	done := &models.AgentRun{
		AgentType:   models.AgentSourcing,
		TriggeredBy: models.TriggerCron,
		Status:      models.AgentRunDone,
		StartedAt:   time.Now().Add(-2 * time.Hour),
	}
	failed := &models.AgentRun{
		AgentType:   models.AgentSelection,
		TriggeredBy: models.TriggerCron,
		Status:      models.AgentRunFailed,
		StartedAt:   time.Now().Add(-3 * time.Hour),
	}
	for _, r := range []*models.AgentRun{zombie, done, failed} {
		if err := repo.Create(r); err != nil {
			t.Fatalf("造数据失败: %v", err)
		}
	}

	n, err := repo.CleanupZombieRuns()
	if err != nil {
		t.Fatalf("CleanupZombieRuns 失败: %v", err)
	}
	if n != 1 {
		t.Fatalf("应只清理 1 条僵尸记录，实际 %d", n)
	}

	var after models.AgentRun
	if err := db.First(&after, zombie.ID).Error; err != nil {
		t.Fatalf("查询僵尸记录失败: %v", err)
	}
	if after.Status != models.AgentRunFailed {
		t.Fatalf("僵尸记录应变为 failed，got %q", after.Status)
	}
	if after.Output != "进程重启，任务中断" {
		t.Fatalf("僵尸记录应带中断说明，got %q", after.Output)
	}
	if after.FinishedAt == nil {
		t.Fatal("僵尸记录应补记 finished_at")
	}

	// 幂等：再次清理应无变化
	n2, err := repo.CleanupZombieRuns()
	if err != nil {
		t.Fatalf("第二次 CleanupZombieRuns 失败: %v", err)
	}
	if n2 != 0 {
		t.Fatalf("第二次清理应无变化，实际清理 %d 条", n2)
	}

	// done/failed 记录不受影响
	var cnt int64
	db.Model(&models.AgentRun{}).Where("status = ?", models.AgentRunDone).Count(&cnt)
	if cnt != 1 {
		t.Fatalf("done 记录应保留 1 条，got %d", cnt)
	}
	db.Model(&models.AgentRun{}).Where("status = ?", models.AgentRunFailed).Count(&cnt)
	if cnt != 2 { // 原 1 条 failed + 收敛的 1 条
		t.Fatalf("failed 记录应为 2 条（原 1 + 僵尸收敛 1），got %d", cnt)
	}
}
