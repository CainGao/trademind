// Package handler — RAG 知识库 HTTP 端点（Week 8）。
//
// 端点：
//   POST   /api/knowledge/upload        上传文件（multipart）
//   POST   /api/knowledge/paste         粘贴文本
//   GET    /api/knowledge/files         文件列表（分页）
//   GET    /api/knowledge/files/:id     文件详情
//   DELETE /api/knowledge/files/:id     删除文件+切片
//   POST   /api/knowledge/search        语义检索
//   POST   /api/knowledge/chat          RAG 对话
//   GET    /api/knowledge/stats         知识库统计
package handler

import (
	"strconv"

	"github.com/CainGao/trademind/internal/pkg/response"
	"github.com/CainGao/trademind/internal/repository"
	"github.com/CainGao/trademind/internal/service"
	"github.com/gin-gonic/gin"
)

// KnowledgeHandler 知识库 HTTP 端点。
type KnowledgeHandler struct {
	svc *service.KnowledgeService
}

// NewKnowledgeHandler 构造。
func NewKnowledgeHandler(svc *service.KnowledgeService) *KnowledgeHandler {
	return &KnowledgeHandler{svc: svc}
}

// RegisterRoutes 注册路由（在 protected 组内调用）。
func (h *KnowledgeHandler) RegisterRoutes(rg *gin.RouterGroup) {
	kb := rg.Group("/knowledge")
	kb.POST("/upload", h.Upload)
	kb.POST("/paste", h.Paste)
	kb.GET("/files", h.ListFiles)
	kb.GET("/files/:id", h.GetFile)
	kb.DELETE("/files/:id", h.DeleteFile)
	kb.POST("/search", h.Search)
	kb.POST("/chat", h.Chat)
	kb.GET("/stats", h.Stats)
}

// Upload 上传文件。
// POST /api/knowledge/upload (multipart, field: "file")
func (h *KnowledgeHandler) Upload(c *gin.Context) {
	fileHeader, err := c.FormFile("file")
	if err != nil {
		response.BadRequest(c, "请上传文件（字段名 file）")
		return
	}
	// 限制文件大小 10MB
	if fileHeader.Size > 10*1024*1024 {
		response.BadRequest(c, "文件大小不能超过 10MB")
		return
	}

	src, err := fileHeader.Open()
	if err != nil {
		response.InternalError(c, "打开上传文件失败: "+err.Error())
		return
	}
	defer src.Close()

	userID := c.GetUint("user_id")
	result, err := h.svc.IngestFile(fileHeader.Filename, fileHeader.Size, src, userID)
	if err != nil {
		response.InternalError(c, "文件处理失败: "+err.Error())
		return
	}
	response.Created(c, result)
}

// PasteRequest 粘贴文本请求体。
type PasteRequest struct {
	Title   string `json:"title"`
	Content string `json:"content" binding:"required"`
}

// Paste 粘贴纯文本入库。
// POST /api/knowledge/paste
func (h *KnowledgeHandler) Paste(c *gin.Context) {
	var req PasteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误: "+err.Error())
		return
	}
	userID := c.GetUint("user_id")
	result, err := h.svc.IngestText(req.Title, req.Content, userID)
	if err != nil {
		response.InternalError(c, "文本处理失败: "+err.Error())
		return
	}
	response.Created(c, result)
}

// FileListQuery 文件列表查询参数。
type FileListQuery struct {
	Page     int    `form:"page"`
	PageSize int    `form:"page_size"`
	Keyword  string `form:"keyword"`
	FileType string `form:"file_type"`
}

// ListFiles 文件列表。
// GET /api/knowledge/files?page=1&page_size=20&keyword=xxx
func (h *KnowledgeHandler) ListFiles(c *gin.Context) {
	var q FileListQuery
	_ = c.ShouldBindQuery(&q)
	if q.Page < 1 {
		q.Page = 1
	}
	if q.PageSize < 1 || q.PageSize > 100 {
		q.PageSize = 20
	}

	result, err := h.svc.ListFiles(repository.FileListParams{
		Page:     q.Page,
		PageSize: q.PageSize,
		Keyword:  q.Keyword,
		FileType: q.FileType,
	})
	if err != nil {
		response.InternalError(c, "查询失败: "+err.Error())
		return
	}
	response.SuccessPage(c, result.Items, result.Total, q.Page, q.PageSize)
}

// GetFile 文件详情。
// GET /api/knowledge/files/:id
func (h *KnowledgeHandler) GetFile(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "id 非法")
		return
	}
	file, chunkCount, err := h.svc.GetFile(uint(id))
	if err != nil {
		response.NotFound(c, "文件不存在")
		return
	}
	response.Success(c, gin.H{"file": file, "chunk_count": chunkCount})
}

// DeleteFile 删除文件 + 切片。
// DELETE /api/knowledge/files/:id
func (h *KnowledgeHandler) DeleteFile(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "id 非法")
		return
	}
	if err := h.svc.DeleteFile(uint(id)); err != nil {
		response.InternalError(c, "删除失败: "+err.Error())
		return
	}
	response.Success(c, gin.H{"deleted": true})
}

// SearchRequest 语义检索请求体。
type SearchRequest struct {
	Query  string `json:"query" binding:"required"`
	FileID uint   `json:"file_id,omitempty"` // 可选：限定在某文件内检索
	TopK   int    `json:"top_k,omitempty"`   // 默认 5
}

// Search 语义检索。
// POST /api/knowledge/search
func (h *KnowledgeHandler) Search(c *gin.Context) {
	var req SearchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误: "+err.Error())
		return
	}
	results, err := h.svc.Search(req.Query, req.FileID, req.TopK)
	if err != nil {
		response.InternalError(c, "检索失败: "+err.Error())
		return
	}
	response.Success(c, gin.H{"results": results, "count": len(results)})
}

// ChatRequest RAG 对话请求体。
type ChatRequest struct {
	Query    string `json:"query" binding:"required"`
	History  string `json:"history,omitempty"` // 之前的对话内容（纯文本）
	Provider string `json:"provider,omitempty"`
}

// Chat RAG 检索增强对话。
// POST /api/knowledge/chat
func (h *KnowledgeHandler) Chat(c *gin.Context) {
	var req ChatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误: "+err.Error())
		return
	}
	provider := service.AIProvider(c.DefaultQuery("provider", req.Provider))
	answer, err := h.svc.RAGChat(req.Query, req.History, provider)
	if err != nil {
		response.InternalError(c, "RAG 对话失败: "+err.Error())
		return
	}
	response.Success(c, answer)
}

// Stats 知识库统计。
// GET /api/knowledge/stats
func (h *KnowledgeHandler) Stats(c *gin.Context) {
	stats, err := h.svc.Stats()
	if err != nil {
		response.InternalError(c, "统计失败: "+err.Error())
		return
	}
	response.Success(c, stats)
}
