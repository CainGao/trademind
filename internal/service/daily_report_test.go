package service

import (
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/CainGao/trademind/internal/database"
	"github.com/CainGao/trademind/internal/models"
	"github.com/CainGao/trademind/internal/repository"
)

// newDailyReportTestSvc 构造一个仅用于测试 DeliverToFeishu 的服务，
// 用临时库 + 畸形 webhook URL 验证 http.NewRequest 错误处理（gotcha #63）。
func newDailyReportTestSvc(t *testing.T) (*DailyReportService, func()) {
	t.Helper()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "trademind.db")

	db, err := database.Open(database.Config{Path: dbPath})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.AutoMigrate(&models.Setting{}, &models.DailyReport{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	settingRepo := repository.NewSettingRepo(db)
	svc := NewDailyReportService(db, nil, nil, nil, nil, nil, settingRepo)

	cleanup := func() {
		if sqlDB, err := db.DB(); err == nil {
			_ = sqlDB.Close()
		}
	}
	return svc, cleanup
}

// TestDeliverToFeishu_MalformedWebhookURL 验证畸形 webhook URL 返回明确错误而非 panic。
// 修复前 http.NewRequest 的 err 被忽略（_），req 为 nil 时下一行 Header.Set 会 panic。
func TestDeliverToFeishu_MalformedWebhookURL(t *testing.T) {
	svc, cleanup := newDailyReportTestSvc(t)
	defer cleanup()

	// 1. 写入畸形 webhook URL（包含控制字符，触发 url.Parse 失败）
	if err := svc.setting.Set("feishu_webhook_url", "ht\x00tp://invalid", false); err != nil {
		t.Fatalf("set malformed webhook url: %v", err)
	}

	// 2. 插入一条日报（DeliverToFeishu 需要先取到 report）
	now := time.Now()
	report := &models.DailyReport{
		ReportDate:  time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local),
		Summary:     `{"test": true}`,
		AINarrative: "测试日报",
	}
	if err := svc.db.Create(report).Error; err != nil {
		t.Fatalf("create report: %v", err)
	}

	// 3. 调用 DeliverToFeishu，期望返回明确错误（而非 panic）
	err := svc.DeliverToFeishu(report.ID)
	if err == nil {
		t.Fatal("期望畸形 URL 返回错误，实际返回 nil")
	}
	if !strings.Contains(err.Error(), "URL 无效") {
		t.Errorf("期望错误包含 'URL 无效'，实际: %v", err)
	}
}

// TestDeliverToFeishu_NoWebhookConfigured 验证未配置 webhook 时返回友好提示。
func TestDeliverToFeishu_NoWebhookConfigured(t *testing.T) {
	svc, cleanup := newDailyReportTestSvc(t)
	defer cleanup()

	err := svc.DeliverToFeishu(1)
	if err == nil {
		t.Fatal("期望未配置 webhook 时返回错误")
	}
	if !strings.Contains(err.Error(), "未配置") {
		t.Errorf("期望错误包含 '未配置'，实际: %v", err)
	}
}
