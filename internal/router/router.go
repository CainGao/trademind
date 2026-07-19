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
	customerRepo := repository.NewCustomerRepo(db)
	inquiryRepo := repository.NewInquiryRepo(db)
	quotationRepo := repository.NewQuotationRepo(db)
	storeRepo := repository.NewStoreRepo(db)
	listingRepo := repository.NewListingRepo(db)
	orderRepo := repository.NewOrderRepo(db)
	agentRepo := repository.NewAgentRepo(db)

	// Service
	authSvc := service.NewAuthService(userRepo, auditRepo, jwtMgr)
	setupSvc := service.NewSetupService(companyRepo, settingRepo, userRepo, encryptor)
	productSvc := service.NewProductService(productRepo)
	extensionSvc := service.NewExtensionService(productRepo, supplierRepo, behaviorRepo)
	supplierSvc := service.NewSupplierService(supplierRepo)
	statsSvc := service.NewStatsService(behaviorRepo, productRepo, supplierRepo)
	aiSvc := service.NewAIService(settingRepo, encryptor)
	agentSvc := service.NewAgentService(aiSvc, productRepo, supplierRepo, behaviorRepo, agentRepo)
	schedSvc := service.NewSchedulerService(agentSvc, settingRepo)
	b2cSvc := service.NewB2CService(storeRepo, listingRepo, orderRepo)
	dailyReportSvc := service.NewDailyReportService(db, aiSvc, statsSvc, productRepo, supplierRepo, behaviorRepo, settingRepo)

	// 把日报服务注入调度器（每天 18:00 触发）
	schedSvc.SetDailyReportService(dailyReportSvc)
	customerSvc := service.NewCustomerService(customerRepo)
	inquirySvc := service.NewInquiryService(inquiryRepo)
	quotationSvc := service.NewQuotationService(quotationRepo)

	// Handler
	authHandler := handler.NewAuthHandler(authSvc)
	setupHandler := handler.NewSetupHandler(setupSvc)
	productHandler := handler.NewProductHandler(productSvc)
	extensionHandler := handler.NewExtensionHandler(extensionSvc)
	supplierHandler := handler.NewSupplierHandler(supplierSvc)
	statsHandler := handler.NewStatsHandler(statsSvc)
	aiHandler := handler.NewAIHandler(aiSvc, agentSvc)
	b2bHandler := handler.NewB2BHandler(customerSvc, inquirySvc, quotationSvc)
	agentHandler := handler.NewAgentHandler(agentSvc, schedSvc)
	b2cHandler := handler.NewB2CHandler(b2cSvc)
	dailyReportHandler := handler.NewDailyReportHandler(dailyReportSvc)

	// 启动 Agent 定时调度器（选品/采购 Agent，默认每天 9:00 / 10:00）
	// 即使没有配置 AI Key 也启动——任务会失败但记录到 agent_runs，老板能在前端看到。
	go func() {
		if err := schedSvc.Start(); err != nil {
			// 调度器启动失败不阻塞 HTTP 服务，只记录日志
			_ = err // log 会在 scheduler_service.go 里打
		}
	}()

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

		// AI 网关 + Agent
		aiHandler.RegisterRoutes(protected)

		// B2B 客户/询盘/报价单
		b2bHandler.RegisterRoutes(protected)

		// Agent 任务管理（选品/采购 Agent + 运行历史 + 定时配置）
		agents := protected.Group("/agents")
		agents.POST("/run", agentHandler.Run)
		agents.POST("/analyze-product", agentHandler.AnalyzeProduct)
		agents.GET("/runs", agentHandler.ListRuns)
		agents.GET("/runs/:id", agentHandler.GetRun)
		agents.GET("/schedule", agentHandler.GetSchedule)
		agents.PUT("/schedule", agentHandler.UpdateSchedule)
		// Week 6 专用 Agent
		agents.POST("/analyze-email", agentHandler.AnalyzeEmail)
		agents.POST("/analyze-inquiry", agentHandler.AnalyzeInquiry)
		agents.POST("/advise-quotation", agentHandler.AdviseQuotation)
		agents.POST("/optimize-listing", agentHandler.OptimizeListing)
		agents.POST("/analyze-reviews", agentHandler.AnalyzeReviews)

		// B2C 跨境电商（店铺/上架/订单）
		b2c := protected.Group("/b2c")
		b2c.GET("/stores", b2cHandler.ListStores)
		b2c.POST("/stores", b2cHandler.CreateStore)
		b2c.PUT("/stores/:id", b2cHandler.UpdateStore)
		b2c.DELETE("/stores/:id", b2cHandler.DeleteStore)
		b2c.GET("/listings", b2cHandler.ListListings)
		b2c.POST("/listings", b2cHandler.CreateListing)
		b2c.PUT("/listings/:id", b2cHandler.UpdateListing)
		b2c.GET("/orders", b2cHandler.ListOrders)
		b2c.POST("/orders", b2cHandler.CreateOrder)
		b2c.PUT("/orders/:id/status", b2cHandler.UpdateOrderStatus)
		b2c.GET("/overview", b2cHandler.Overview)

		// 老板日报 + 飞书 webhook 推送
		dr := protected.Group("/daily-reports")
		dr.POST("/generate", dailyReportHandler.Generate)
		dr.GET("", dailyReportHandler.List)
		dr.GET("/today", dailyReportHandler.GetToday)
		dr.GET("/feishu-config", dailyReportHandler.GetFeishuConfig)
		dr.PUT("/feishu-config", dailyReportHandler.UpdateFeishuConfig)
		dr.GET("/:id", dailyReportHandler.GetByID)
		dr.POST("/:id/deliver-feishu", dailyReportHandler.DeliverToFeishu)

		// Chrome 插件对接（采集 / 行为 / 状态）
		extensionHandler.RegisterRoutes(protected)
	}

	return r
}
