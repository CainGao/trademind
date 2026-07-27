// Package handler — 数据备份 HTTP 处理器。
//
// 路由（全部需管理员权限，备份含全量公司数据）：
//   POST   /api/system/backup            生成一份完整备份
//   GET    /api/system/backups           备份列表（按时间倒序）
//   GET    /api/system/backups/:filename 下载备份 zip
//   DELETE /api/system/backups/:filename 删除一份备份
//
// 恢复为 CLI 操作（`./trademind --restore <file>`），不在 HTTP 暴露，
// 因为恢复需替换活动库文件，必须在服务停止时进行。
package handler

import (
	"net/url"

	"github.com/CainGao/trademind/internal/pkg/response"
	"github.com/CainGao/trademind/internal/service"
	"github.com/gin-gonic/gin"
)

type BackupHandler struct {
	svc *service.BackupService
}

func NewBackupHandler(svc *service.BackupService) *BackupHandler {
	return &BackupHandler{svc: svc}
}

func (h *BackupHandler) RegisterRoutes(r *gin.RouterGroup) {
	sys := r.Group("/system")
	sys.POST("/backup", h.Create)
	sys.GET("/backups", h.List)
	sys.GET("/backups/:filename", h.Download)
	sys.DELETE("/backups/:filename", h.Delete)
}

// Create 生成一份完整备份。
func (h *BackupHandler) Create(c *gin.Context) {
	info, err := h.svc.Create()
	if err != nil {
		response.InternalError(c, "备份失败: "+err.Error())
		return
	}
	response.Created(c, info)
}

// List 备份列表（倒序）。
func (h *BackupHandler) List(c *gin.Context) {
	list, err := h.svc.List()
	if err != nil {
		response.InternalError(c, "查询失败: "+err.Error())
		return
	}
	response.Success(c, list)
}

// Download 下载备份 zip。
func (h *BackupHandler) Download(c *gin.Context) {
	filename := c.Param("filename")
	full, err := h.svc.Path(filename)
	if err != nil {
		response.NotFound(c, err.Error())
		return
	}
	// 文件名含中文/特殊字符时，浏览器下载用 RFC 5987 编码兜底
	c.Header("Content-Disposition",
		`attachment; filename="`+filename+`"; filename*=UTF-8''`+url.PathEscape(filename))
	c.File(full)
}

// Delete 删除一份备份。
func (h *BackupHandler) Delete(c *gin.Context) {
	filename := c.Param("filename")
	if err := h.svc.Delete(filename); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	response.Success(c, gin.H{"deleted": true})
}
