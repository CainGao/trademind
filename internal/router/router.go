// Package router 注册所有 HTTP 路由（规范 V1.0 §3.1）。
//
// 路由分层：
//   - /api/auth/*    认证（公开）
//   - /api/*         需登录（JWT 中间件）
//   - /api/admin/*   仅管理员
//   - /api/ai/*      AI 相关（限流，待实现）
package router

import (
	"github.com/CainGao/trademind/internal/config"
	"github.com/CainGao/trademind/internal/database"
	"github.com/CainGao/trademind/internal/handler"
	"github.com/CainGao/trademind/internal/middleware"
	"github.com/CainGao/trademind/internal/repository"
	"github.com/CainGao/trademind/internal/service"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// New 创建 Gin 引擎并注册全部路由。
// 所有依赖（Repo/Service/Handler）在此装配（简易 DI）。
func New(cfg *config.Config, db *gorm.DB) *gin.Engine {
	if cfg.App.Production {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()
	r.Use(gin.Logger())
	r.Use(gin.Recovery())

	// CORS（规范 §6.5: 仅 localhost）
	r.Use(cors.New(cors.Config{
		AllowOrigins: []string{
			"http://localhost:7789", "http://127.0.0.1:7789",
			"http://localhost:5173", "http://127.0.0.1:5173", // 前端开发服务器
		},
		AllowMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders: []string{"Authorization", "Content-Type"},
	}))

	// ===== 装配依赖（简易 DI）=====

	// JWT Secret（启动时从 settings 读，已由 Seed 注入）
	jwtSecret, err := database.GetJWTSecret(db)
	if err != nil {
		panic("JWT secret 未初始化，请先调用 database.Seed: " + err.Error())
	}
	jwtMgr := middleware.NewJWTManager(jwtSecret)

	// Repository
	userRepo := repository.NewUserRepo(db)
	auditRepo := repository.NewAuditLogRepo(db)

	// Service
	authSvc := service.NewAuthService(userRepo, auditRepo, jwtMgr)

	// Handler
	authHandler := handler.NewAuthHandler(authSvc)

	// ===== 健康检查（公开）=====
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status":  "ok",
			"name":    cfg.App.Name,
			"version": cfg.App.Version,
		})
	})

	// ===== API 路由 =====
	api := r.Group("/api")

	// 认证（公开）
	authGroup := api.Group("/auth")
	authHandler.RegisterRoutes(authGroup)

	// 需登录（JWT）
	protected := api.Group("")
	protected.Use(middleware.JWT(jwtMgr))
	{
		// 当前用户（注意：路径用 /me 而非 /auth/me，避免与公开 auth 组冲突）
		protected.GET("/me", authHandler.Me)
		// TODO: 后续业务模块在此注册
		// modules/common.RegisterRoutes(protected, ...)
		// modules/b2b.RegisterRoutes(protected, ...)
		// modules/b2c.RegisterRoutes(protected, ...)
	}

	return r
}
