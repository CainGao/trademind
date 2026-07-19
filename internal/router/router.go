// Package router 注册所有 HTTP 路由（规范 V1.0 §3.1）。
//
// 路由分层：
//   /api/setup/*    首启向导（status 公开；写操作需 admin）
//   /api/auth/*     认证（公开）
//   /api/*          需登录（JWT）+ 首启完成
//
// 首启未完成时，除 setup/auth 外的所有请求都会被中间件拒绝（403 + setup_required）。
package router

import (
	"github.com/CainGao/trademind/internal/config"
	"github.com/CainGao/trademind/internal/database"
	"github.com/CainGao/trademind/internal/handler"
	"github.com/CainGao/trademind/internal/middleware"
	"github.com/CainGao/trademind/internal/pkg/crypto"
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
			"http://localhost:5173", "http://127.0.0.1:5173",
		},
		AllowMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders: []string{"Authorization", "Content-Type"},
	}))

	// ===== 装配依赖（简易 DI）=====

	// JWT Secret
	jwtSecret, err := database.GetJWTSecret(db)
	if err != nil {
		panic("JWT secret 未初始化: " + err.Error())
	}
	jwtMgr := middleware.NewJWTManager(jwtSecret)

	// AES 加密密钥（规范 §6.2）
	aesKey, err := database.GetOrCreateAESKey(db)
	if err != nil {
		panic("AES key 初始化失败: " + err.Error())
	}
	encryptor, err := crypto.NewEncryptor(aesKey)
	if err != nil {
		panic("Encryptor 创建失败: " + err.Error())
	}

	// Repository
	userRepo := repository.NewUserRepo(db)
	auditRepo := repository.NewAuditLogRepo(db)
	companyRepo := repository.NewCompanyRepo(db)
	settingRepo := repository.NewSettingRepo(db)
	productRepo := repository.NewProductRepo(db)
	supplierRepo := repository.NewSupplierRepo(db)
	behaviorRepo := repository.NewBehaviorRepo(db)

	// Service
	authSvc := service.NewAuthService(userRepo, auditRepo, jwtMgr)
	setupSvc := service.NewSetupService(companyRepo, settingRepo, userRepo, encryptor)
	productSvc := service.NewProductService(productRepo)
	extensionSvc := service.NewExtensionService(productRepo, supplierRepo, behaviorRepo)
	supplierSvc := service.NewSupplierService(supplierRepo)
	statsSvc := service.NewStatsService(behaviorRepo, productRepo, supplierRepo)

	// Handler
	authHandler := handler.NewAuthHandler(authSvc)
	setupHandler := handler.NewSetupHandler(setupSvc)
	productHandler := handler.NewProductHandler(productSvc)
	extensionHandler := handler.NewExtensionHandler(extensionSvc)
	supplierHandler := handler.NewSupplierHandler(supplierSvc)
	statsHandler := handler.NewStatsHandler(statsSvc)

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

	// 首启状态查询（公开，前端守卫用）
	api.GET("/setup/status", setupHandler.Status)

	// 认证（公开）
	authGroup := api.Group("/auth")
	authHandler.RegisterRoutes(authGroup)

	// 需登录 + 首启完成（除 setup 写操作外）
	protected := api.Group("")
	protected.Use(middleware.JWT(jwtMgr))
	protected.Use(middleware.SetupGuard(setupSvc))
	{
		// 首启向导写操作
		setupHandler.RegisterRoutes(protected)

		// 当前用户
		protected.GET("/me", authHandler.Me)

		// 商品中心（CRUD）
		productHandler.RegisterRoutes(protected)

		// 供应商（列表/详情/评分/总览）
		supplierHandler.RegisterRoutes(protected)

		// 行为统计（驾驶舱数据源）
		statsHandler.RegisterRoutes(protected)

		// Chrome 插件对接（采集 / 行为 / 状态）
		extensionHandler.RegisterRoutes(protected)
	}

	return r
}
