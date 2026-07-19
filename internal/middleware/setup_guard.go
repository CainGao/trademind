// Package middleware — 首启守卫。
//
// 在 setup 未完成时，除 setup/auth 外的所有请求都返回 403 + setup_required。
// 前端拦截此响应自动跳转 /setup。
//
// 为避免循环引用（service ↔ middleware），通过 SetupChecker 接口解耦。

package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"
)

// SetupChecker 由 service 层实现，用于判断首启是否完成。
type SetupChecker interface {
	IsSetupRequired() bool
}

// SetupGuard 首启守卫中间件。
// checker 由 SetupService 实现（避免循环引用）。
func SetupGuard(checker SetupChecker) gin.HandlerFunc {
	return func(c *gin.Context) {
		// setup 自己的端点放行（POST /api/setup/*）
		if strings.HasPrefix(c.Request.URL.Path, "/api/setup/") {
			c.Next()
			return
		}

		if checker.IsSetupRequired() {
			c.AbortWithStatusJSON(403, gin.H{
				"code":    4001,
				"message": "首次启动向导未完成",
				"data":    gin.H{"setup_required": true},
			})
			return
		}
		c.Next()
	}
}
