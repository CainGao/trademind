// Package database 负责 SQLite 连接、AutoMigrate、种子数据。
//
// 驱动选择（规范 V1.0 §1.1）: 用 github.com/glebarez/sqlite（纯 Go modernc 实现），
// 不用 gorm.io/driver/sqlite（基于 mattn/go-sqlite3，需 CGO）。
// 纯 Go 驱动让 Windows 交叉编译零障碍。
package database

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/CainGao/trademind/internal/models"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Config 数据库配置。
type Config struct {
	Path string // 数据库文件路径，如 runtime/trademind.db
}

// Open 打开/创建 SQLite 数据库并返回 *gorm.DB。
func Open(cfg Config) (*gorm.DB, error) {
	// 确保目录存在
	if err := os.MkdirAll(filepath.Dir(cfg.Path), 0755); err != nil {
		return nil, fmt.Errorf("创建数据库目录失败 [path=%s]: %w", cfg.Path, err)
	}

	db, err := gorm.Open(sqlite.Open(cfg.Path), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Warn),
	})
	if err != nil {
		return nil, fmt.Errorf("打开数据库失败 [path=%s]: %w", cfg.Path, err)
	}

	// SQLite 性能参数
	sqlDB, _ := db.DB()
	sqlDB.SetMaxOpenConns(1) // SQLite 写并发限制（单机够用）
	sqlDB.SetMaxIdleConns(1)

	// 启用外键约束
	db.Exec("PRAGMA foreign_keys = ON")
	// WAL 模式（提升并发读）
	db.Exec("PRAGMA journal_mode = WAL")

	return db, nil
}

// AllModels 返回所有模块的全部 Model（架构文档 §4.3: 全部模块一次性建表）。
// 即使某模块未启用也建表，避免"启用模块时迁移"的坑。
func AllModels() []interface{} {
	return []interface{}{
		// === 共享底座（common）===
		&models.Company{},
		&models.User{},
		&models.Product{},
		&models.Supplier{},
		&models.SupplierProduct{},
		&models.BehaviorEvent{},
		&models.AIResult{},
		&models.AgentRun{},
		&models.DailyReport{},
		&models.KnowledgeChunk{},
		&models.File{},
		&models.Setting{},
		&models.AgentPrompt{},
		&models.APICallLog{},
		&models.AuditLog{},

		// === B2B 外贸 Pack ===
		&models.Customer{},
		&models.CustomerContact{},
		&models.CustomerCommunication{},
		&models.Inquiry{},
		&models.Quotation{},
		&models.Sample{},
		&models.Contract{},
		&models.EmailThread{},

		// === B2C 跨境 Pack ===
		&models.Store{},
		&models.Listing{},
		&models.Order{},
		&models.Inventory{},
		&models.AdCampaign{},
		&models.Review{},
	}
}

// AutoMigrate 自动建表/加列（规范 V1.0 §1.6）。
//
// 规则：
//   - 新增字段：直接加到 Model，AutoMigrate 自动加列
//   - 删除字段：GORM 不自动删列，字段保留（向后兼容）
//   - 改类型/改字段名：单独写迁移脚本，不用 AutoMigrate
func AutoMigrate(db *gorm.DB) error {
	if err := db.AutoMigrate(AllModels()...); err != nil {
		return fmt.Errorf("AutoMigrate 失败: %w", err)
	}
	return nil
}
