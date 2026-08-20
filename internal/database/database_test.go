package database

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	_ "github.com/glebarez/sqlite"
)

// TestOpen_TruncatesLeftoverWAL 验证启动时 WAL 自愈（gotcha #81）。
//
// 场景：上个会话被强杀（SIGKILL/断电），-wal 文件残留未 checkpoint。
// 模拟方式：用原始 database/sql 连接写入一行且**不关闭连接**（连接关闭时
// SQLite 会自动 checkpoint 并删除 wal 文件，所以必须保持打开来模拟残留）。
// 期望：database.Open 后 wal 文件被 TRUNCATE 截断为 0，且残留数据已恢复可见。
func TestOpen_TruncatesLeftoverWAL(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "wal_test.db")

	// 1. 用原始连接制造一个带残留 WAL 的库
	raw, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("打开原始连接失败: %v", err)
	}
	defer raw.Close()
	if _, err := raw.Exec("PRAGMA journal_mode = WAL"); err != nil {
		t.Fatalf("设置 WAL 模式失败: %v", err)
	}
	if _, err := raw.Exec("CREATE TABLE leftover (id INTEGER PRIMARY KEY, note TEXT)"); err != nil {
		t.Fatalf("建表失败: %v", err)
	}
	if _, err := raw.Exec("INSERT INTO leftover (note) VALUES ('from-killed-session')"); err != nil {
		t.Fatalf("插入失败: %v", err)
	}

	// 2. 断言 wal 文件存在且有内容（模拟强杀残留）
	walPath := dbPath + "-wal"
	walSize := func() int64 {
		info, err := os.Stat(walPath)
		if err != nil {
			return -1
		}
		return info.Size()
	}
	if walSize() <= 0 {
		t.Fatalf("前置条件不满足: wal 文件应存在且有内容，实际 size=%d", walSize())
	}

	// 3. database.Open 应触发 TRUNCATE checkpoint
	gormDB, err := Open(Config{Path: dbPath})
	if err != nil {
		t.Fatalf("Open 失败: %v", err)
	}
	sqlDB, _ := gormDB.DB()
	defer sqlDB.Close()

	if got := walSize(); got != 0 {
		t.Errorf("启动 checkpoint 后 wal 应截断为 0，实际 size=%d", got)
	}

	// 4. 残留数据应已恢复（checkpoint 把 wal 帧刷入主库）
	var note string
	if err := gormDB.Raw("SELECT note FROM leftover WHERE id = 1").Scan(&note).Error; err != nil {
		t.Fatalf("读取残留数据失败: %v", err)
	}
	if note != "from-killed-session" {
		t.Errorf("残留数据不符: got %q", note)
	}
}

// TestOpen_正常打开 验证空库/新库路径下 Open + checkpoint 不报错。
func TestOpen_正常打开(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "fresh.db")

	gormDB, err := Open(Config{Path: dbPath})
	if err != nil {
		t.Fatalf("Open 失败: %v", err)
	}
	sqlDB, _ := gormDB.DB()
	defer sqlDB.Close()

	// journal_mode 应为 wal
	var mode string
	if err := gormDB.Raw("PRAGMA journal_mode").Scan(&mode).Error; err != nil {
		t.Fatalf("查询 journal_mode 失败: %v", err)
	}
	if mode != "wal" {
		t.Errorf("journal_mode 应为 wal，实际 %q", mode)
	}
}
