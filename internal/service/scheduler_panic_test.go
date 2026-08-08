package service

import (
	"testing"

	"github.com/CainGao/trademind/internal/models"
)

// TestRunAgent_PanicRecovery 验证 runAgent 的 defer/recover 保护：
// Agent 执行 panic 时不应崩溃测试进程（gotcha #42 延伸）。
func TestRunAgent_PanicRecovery(t *testing.T) {
	// 构造一个 agentSvc 为 nil 的 scheduler，调用 RunSelection 时会 nil dereference panic
	s := &SchedulerService{
		agentSvc: nil,
	}

	// runAgent 应该 recover 而非 panic 传播
	// 如果没有 recover，这行会导致 test 崩溃（fatal error）
	s.runAgent(models.AgentSelection)

	// 如果执行到这里，说明 panic 被成功 recover
	t.Log("runAgent panic recovery 正常工作")
}

// TestRunAgent_PanicRecoverySourcing 验证 sourcing Agent 的 recover。
func TestRunAgent_PanicRecoverySourcing(t *testing.T) {
	s := &SchedulerService{
		agentSvc: nil,
	}

	s.runAgent(models.AgentSourcing)

	t.Log("runAgent (sourcing) panic recovery 正常工作")
}
