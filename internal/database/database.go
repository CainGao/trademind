// Package database 负责 SQLite 连接、AutoMigrate、种子数据。
//
// 驱动选择（规范 V1.0 §1.1）: 用 github.com/glebarez/sqlite（纯 Go modernc 实现），
// 不用 gorm.io/driver/sqlite（基于 mattn/go-sqlite3，需 CGO）。
// 纯 Go 驱动让 Windows 交叉编译零障碍。
package database

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

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
		// ErrRecordNotFoundError 是业务常态（如首次启动 settings 表无 cron 配置、
		// LatestByType 无记录），不应作为错误刷日志。调用方仍会收到 error 正常处理，
		// 这里只是抑制 GORM logger 把它当 SQL 错误打印（启动噪音）。
		Logger: logger.New(
			log.New(os.Stdout, "\r\n", log.LstdFlags),
			logger.Config{
				SlowThreshold:             200 * time.Millisecond,
				LogLevel:                  logger.Warn,
				IgnoreRecordNotFoundError: true,
			},
		),
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

	// WAL 自愈（gotcha #81）：上个会话若被强杀（SIGKILL/断电/直接关机），
	// 优雅关闭（gotcha #49）没有机会执行，-wal 文件会残留并跨会话无限增长
	// （实测 8 周每日 pkill -9 重启后 wal 达 4MB，比主库还大）。
	// 启动时强制 TRUNCATE checkpoint：把残留 WAL 帧刷入主库并将 wal 截断为 0，
	// 保证 WAL 尺寸只受单次会话影响。checkpoint 失败（临时 busy 等）不阻塞启动。
	if res := db.Exec("PRAGMA wal_checkpoint(TRUNCATE)"); res.Error != nil {
		log.Printf("[database] 启动 WAL checkpoint 失败（不阻塞启动）: %v", res.Error)
	}

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
		&models.KnowledgeFile{},
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
