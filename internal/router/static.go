// Package router — 静态文件服务，go:embed 前端产物。
//
// 开发模式：前端独立跑 5173，后端 7789 通过 Vite proxy 转发。
// 生产模式：前端构建产物在 web/dist/，编译时 embed 进 Go 二进制。

package router

import (
	"embed"
	"io/fs"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// SetupStatic 注册前端静态资源服务。
// distFS 是 embed.FS，根目录对应前端构建产物。
func SetupStatic(r *gin.Engine, distFS embed.FS) error {
	// 从 embed.FS 中取出 dist 子目录
	subFS, err := fs.Sub(distFS, "dist")
	if err != nil {
		return err
	}

	// 直接用 http.FileServer 提供
	fileServer := http.FileServer(http.FS(subFS))

	// 所有非 /api、/health 的请求都走前端（SPA History 模式由 HashRouter 兜底）
	r.NoRoute(func(c *gin.Context) {
		path := c.Request.URL.Path

		// API/健康检查不在这里处理（应已有专用路由）
		if strings.HasPrefix(path, "/api/") || path == "/health" {
			c.JSON(404, gin.H{"code": 1002, "message": "接口不存在"})
			return
		}

		// 尝试直接读文件（如 /assets/index-xxx.js）
		if path != "/" {
			if _, err := fs.Stat(subFS, strings.TrimPrefix(path, "/")); err == nil {
				fileServer.ServeHTTP(c.Writer, c.Request)
				return
			}
		}

		// 找不到文件，回退到 index.html（SPA 入口）
		c.Request.URL.Path = "/"
		fileServer.ServeHTTP(c.Writer, c.Request)
	})

	return nil
}
