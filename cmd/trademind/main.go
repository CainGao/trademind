// TradeMind AI — 企业级私有化 AI 外贸智能操作系统
//
// main 是程序入口。
// 完整启动流程见开发方案 V2 §5.3：
//   加载配置 → 打开数据库 → AutoMigrate → Seed → go:embed 前端 → 启动 HTTP 服务
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/CainGao/trademind/internal/config"
	"github.com/CainGao/trademind/internal/database"
	"github.com/CainGao/trademind/internal/router"
	"github.com/CainGao/trademind/internal/service"
	"github.com/CainGao/trademind/web"
)

func main() {
	// 命令行参数
	configPath := flag.String("config", "", "配置文件路径（默认 runtime/config.yaml）")
	migrateOnly := flag.Bool("migrate-only", false, "只执行数据库迁移后退出")
	restorePath := flag.String("restore", "", "从备份 zip 恢复数据（路径指向 runtime/backups/*.zip；恢复前请先停止正在运行的服务）")
	flag.Parse()

	// 1. 加载配置（支持 --config 自定义 YAML 路径）
	cfg, err := config.LoadFromPath(*configPath)
	if err != nil {
		log.Fatalf("加载配置失败: %v", err)
	}

	// 0. 恢复模式：必须在打开数据库/启动 HTTP 前执行（替换活动库文件不安全）
	if *restorePath != "" {
		dbPath := cfg.Database.Path
		filesDir := filepath.Join(cfg.App.RuntimeDir, "files")
		log.Printf("开始从备份恢复: %s", *restorePath)
		if err := service.RestoreFromZip(*restorePath, dbPath, filesDir); err != nil {
			log.Fatalf("恢复失败: %v", err)
		}
		log.Printf("恢复完成 ✓ 数据库 → %s，附件 → %s", dbPath, filesDir)
		log.Printf("原库已自动备份为 %s.bak.<时间戳>（如存在）。请使用正常模式重新启动。", dbPath)
		return
	}

	log.Printf("[%s v%s] 启动中...", cfg.App.Name, cfg.App.Version)
	log.Printf("数据库: %s", cfg.Database.Path)
	log.Printf("端口: %d", cfg.Server.Port)

	// 2. 打开数据库
	db, err := database.Open(database.Config{Path: cfg.Database.Path})
	if err != nil {
		log.Fatalf("打开数据库失败: %v", err)
	}

	// 3. AutoMigrate（规范 V1.0 §1.6）
	if err := database.AutoMigrate(db); err != nil {
		log.Fatalf("数据库迁移失败: %v", err)
	}
	log.Printf("数据库迁移完成（共 %d 张表）", len(database.AllModels()))

	// 3.5. 种子数据（首次启动注入默认管理员 + JWT Secret，规范 §1.7）
	if err := database.Seed(db); err != nil {
		log.Fatalf("种子数据初始化失败: %v", err)
	}

	if *migrateOnly {
		log.Println("仅迁移模式，退出")
		return
	}

	// 4. 启动 HTTP 服务（含 go:embed 前端）
	server, schedSvc := router.New(cfg, db)
	if err := router.SetupStatic(server, web.DistFS); err != nil {
		log.Fatalf("前端静态资源加载失败: %v", err)
	}

	// 4.5 使用 http.Server 以支持优雅关闭（优于 gin.Engine.Run）
	addr := fmt.Sprintf(":%d", cfg.Server.Port)
	httpServer := &http.Server{
		Addr:    addr,
		Handler: server,
	}
	go func() {
		log.Printf("HTTP 服务启动: http://localhost%s", addr)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("HTTP 服务启动失败: %v", err)
		}
	}()

	// 5. 等待退出信号（支持 Ctrl+C / kill），优雅关闭
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("正在关闭...")

	// 5.1 优雅关闭 HTTP（等待在途请求完成，最多 10s）
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := httpServer.Shutdown(ctx); err != nil {
		log.Printf("HTTP 关闭警告: %v", err)
	}

	// 5.2 停止定时调度器（等待在跑的 cron 任务完成）
	schedSvc.Stop()

	// 5.3 关闭数据库（SQLite WAL 刷盘，防止数据丢失）
	if sqlDB, err := db.DB(); err == nil {
		_ = sqlDB.Close()
	}
	log.Println("已安全退出 ✓")
}
