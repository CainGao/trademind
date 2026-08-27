package service

import (
	"testing"
	"time"
)

// --- gotcha #91：备份启动补跑精确化（backupCatchUpDue / parseDailyCronTime）---

// TestBackupCatchUpDue 表驱动覆盖补跑判定的全部场景。
// 关键回归用例：「当天 02:00 已备 + 当晚 22 点重启」不应再冗余补一份
// （2026-08-27 实测踩中：20h 阈值踩线导致同一天两份备份）。
func TestBackupCatchUpDue(t *testing.T) {
	// 固定基准时间方便构造：2026-08-27 22:02（周四晚，日常维护重启时刻）
	at := func(y, m, d, h, min int) time.Time {
		return time.Date(y, time.Month(m), d, h, min, 0, 0, time.Local)
	}
	cases := []struct {
		name      string
		latest    time.Time // 最新一份自动备份的时间（hasBackup=false 时忽略）
		hasBackup bool
		cronExpr  string
		now       time.Time
		want      bool
	}{
		// ── 日频 cron "0 2 * * *"（默认）──
		{
			name:      "当天已备_晚上重启不补(回归2026-08-27实测)",
			latest:    at(2026, 8, 27, 2, 0),
			hasBackup: true,
			cronExpr:  "0 2 * * *",
			now:       at(2026, 8, 27, 22, 2),
			want:      false,
		},
		{
			name:      "凌晨漏跑_上午开机补",
			latest:    at(2026, 8, 26, 2, 0),
			hasBackup: true,
			cronExpr:  "0 2 * * *",
			now:       at(2026, 8, 27, 9, 0),
			want:      true,
		},
		{
			name:      "深夜漏跑到晚上才开机_补",
			latest:    at(2026, 8, 26, 2, 0),
			hasBackup: true,
			cronExpr:  "0 2 * * *",
			now:       at(2026, 8, 27, 23, 0),
			want:      true,
		},
		{
			name:      "今天时刻未到_昨天已备_不补(cron半小时后自会跑)",
			latest:    at(2026, 8, 26, 2, 0),
			hasBackup: true,
			cronExpr:  "0 2 * * *",
			now:       at(2026, 8, 27, 1, 30),
			want:      false,
		},
		{
			name:      "今天时刻未到_前天已备_补(昨天也漏了)",
			latest:    at(2026, 8, 25, 2, 0),
			hasBackup: true,
			cronExpr:  "0 2 * * *",
			now:       at(2026, 8, 27, 1, 30),
			want:      true,
		},
		{
			name:      "备份时刻恰好等于计划时刻_不补(边界)",
			latest:    at(2026, 8, 27, 2, 0),
			hasBackup: true,
			cronExpr:  "0 2 * * *",
			now:       at(2026, 8, 27, 2, 0),
			want:      false,
		},
		{
			name:      "补跑晚于计划时刻_不重复补",
			latest:    at(2026, 8, 27, 9, 15),
			hasBackup: true,
			cronExpr:  "0 2 * * *",
			now:       at(2026, 8, 27, 10, 0),
			want:      false,
		},
		// ── 从未备份过 ──
		{
			name:      "从未备份_首启立即补",
			hasBackup: false,
			cronExpr:  "0 2 * * *",
			now:       at(2026, 8, 27, 10, 0),
			want:      true,
		},
		// ── 自定义日频 cron "30 3 * * *" ──
		{
			name:      "自定义cron_当天已备不补",
			latest:    at(2026, 8, 27, 3, 30),
			hasBackup: true,
			cronExpr:  "30 3 * * *",
			now:       at(2026, 8, 27, 15, 0),
			want:      false,
		},
		{
			name:      "自定义cron_当天漏跑补",
			latest:    at(2026, 8, 26, 3, 30),
			hasBackup: true,
			cronExpr:  "30 3 * * *",
			now:       at(2026, 8, 27, 15, 0),
			want:      true,
		},
		{
			name:      "自定义cron_时刻前启动_昨天已备不补",
			latest:    at(2026, 8, 26, 3, 30),
			hasBackup: true,
			cronExpr:  "30 3 * * *",
			now:       at(2026, 8, 27, 2, 0),
			want:      false,
		},
		// ── 非日频 cron：回退 20h 阈值启发式 ──
		{
			name:      "周频cron_21h未备_回退阈值补",
			latest:    at(2026, 8, 26, 19, 0),
			hasBackup: true,
			cronExpr:  "0 2 * * 0",
			now:       at(2026, 8, 27, 16, 0),
			want:      true,
		},
		{
			name:      "步进cron_19h未备_回退阈值不补",
			latest:    at(2026, 8, 26, 21, 0),
			hasBackup: true,
			cronExpr:  "0 */6 * * *",
			now:       at(2026, 8, 27, 16, 0),
			want:      false,
		},
		{
			name:      "非法cron_回退阈值补",
			latest:    at(2026, 8, 26, 10, 0),
			hasBackup: true,
			cronExpr:  "not a cron",
			now:       at(2026, 8, 27, 22, 0),
			want:      true,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := backupCatchUpDue(tc.latest, tc.hasBackup, tc.cronExpr, tc.now)
			if got != tc.want {
				t.Errorf("backupCatchUpDue(latest=%v has=%v cron=%q now=%v) = %v, want %v",
					tc.latest, tc.hasBackup, tc.cronExpr, tc.now, got, tc.want)
			}
		})
	}
}

// TestReportCatchUpDue 日报启动补跑判定（gotcha #92）。
// 回归场景：2026-08-27 老板 18:00 前关机，当晚 22 点开机启动应用 → 当天无日报 → 应补跑。
func TestReportCatchUpDue(t *testing.T) {
	at := func(y, m, d, h, min int) time.Time {
		return time.Date(y, time.Month(m), d, h, min, 0, 0, time.Local)
	}
	cases := []struct {
		name           string
		now            time.Time
		hasTodayReport bool
		want           bool
	}{
		{
			name:           "18点后启动_当天无日报_补(回归2026-08-27实测)",
			now:            at(2026, 8, 27, 22, 6),
			hasTodayReport: false,
			want:           true,
		},
		{
			name:           "18点后启动_当天已有日报_不补",
			now:            at(2026, 8, 27, 22, 6),
			hasTodayReport: true,
			want:           false,
		},
		{
			name:           "18点前启动_不补(等cron正常跑)",
			now:            at(2026, 8, 27, 9, 0),
			hasTodayReport: false,
			want:           false,
		},
		{
			name:           "恰好18点整_不补(边界,留给cron)",
			now:            at(2026, 8, 27, 18, 0),
			hasTodayReport: false,
			want:           false,
		},
		{
			name:           "18点01分_当天无日报_补",
			now:            at(2026, 8, 27, 18, 1),
			hasTodayReport: false,
			want:           true,
		},
		{
			name:           "次日凌晨启动_当天(新的一天)18点未到_不补",
			now:            at(2026, 8, 28, 8, 0),
			hasTodayReport: false,
			want:           false,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := reportCatchUpDue(tc.now, tc.hasTodayReport)
			if got != tc.want {
				t.Errorf("reportCatchUpDue(now=%v hasToday=%v) = %v, want %v",
					tc.now, tc.hasTodayReport, got, tc.want)
			}
		})
	}
}

// TestParseDailyCronTime 验证日频 cron 解析的接受与拒绝面。
func TestParseDailyCronTime(t *testing.T) {
	cases := []struct {
		expr   string
		wantH  int
		wantM  int
		wantOK bool
	}{
		{"0 2 * * *", 2, 0, true},
		{"30 3 * * *", 3, 30, true},
		{"59 23 * * *", 23, 59, true},
		{"0 */6 * * *", 0, 0, false}, // 步进
		{"0 2,4 * * *", 0, 0, false}, // 列表
		{"0 2-6 * * *", 0, 0, false}, // 范围
		{"0 2 * * 0", 0, 0, false},   // 周频
		{"0 0 1 * *", 0, 0, false},   // 月频
		{"", 0, 0, false},
		{"garbage", 0, 0, false},
		{"60 2 * * *", 0, 0, false}, // 分钟越界
		{"0 24 * * *", 0, 0, false}, // 小时越界
		{"-1 2 * * *", 0, 0, false}, // 负数
	}
	for _, tc := range cases {
		h, m, ok := parseDailyCronTime(tc.expr)
		if ok != tc.wantOK || (ok && (h != tc.wantH || m != tc.wantM)) {
			t.Errorf("parseDailyCronTime(%q) = (%d,%d,%v), want (%d,%d,%v)",
				tc.expr, h, m, ok, tc.wantH, tc.wantM, tc.wantOK)
		}
	}
}
