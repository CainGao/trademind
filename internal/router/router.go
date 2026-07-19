// Package router 注册所有 HTTP 路由（规范 V1.0 §3.1）。
//
// 路由分层：
//   - /api/auth/*    认证（公开）
//   - /api/*         需登录（JWT 中间件）
//   - /api/ai/*      AI 相关（限流）
package router

import (
	"github.com/CainGao/trademind/internal/config"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// New 创建 Gin 引擎并注册路由。
func New(cfg *config.Config, db *gorm.DB) *gin.Engine {
	if cfg.App.Production {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()
	r.Use(gin.Logger())
	r.Use(gin.Recovery())

	// 健康检查
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status":  "ok",
			"name":    cfg.App.Name,
			"version": cfg.App.Version,
		})
	})

	// API 路由组
	api := r.Group("/api")

	// === 认证路由（公开）===
	// TODO: auth.RegisterRoutes(api.Group("/auth"))

	// === 业务路由（需登录）===
	// TODO: protected := api.Group("")
	// TODO: protected.Use(middleware.JWT())
	// TODO: modules/common, modules/b2b, modules/b2c 注册路由

	_ = api // 占位，后续模块注册时使用

	return r
}
