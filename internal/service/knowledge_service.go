// Package service — RAG 知识库业务（Week 8）。
//
// 功能链路：
//   上传/粘贴文档 → 解析提取文本 → 分片(chunking) → 批量 Embedding → 存 knowledge_chunks
//   检索：query → Embed → 余弦相似度 topK → 拼接上下文
//   RAG Chat：检索上下文 + 用户问题 → AI 生成回答（引用来源）
//
// 设计原则：
//   - Embedding 失败时文件标记 failed，不阻塞其他文件
//   - RAG Chat 的 AI 失败也返回检索到的上下文（降级输出，不阻塞用户）
package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"math"
	"os"
	"path/filepath"
	"strings"

	"github.com/CainGao/trademind/internal/models"
	"github.com/CainGao/trademind/internal/repository"
)

// 知识库默认参数。
const (
	defaultChunkSize = 600 // 每个切片目标字符数
	defaultOverlap   = 100 // 切片重叠字符数
	embedBatchSize   = 20  // 单次 Embedding API 请求的文本数（OpenAI 限制 input 数组长度）
	maxContextChars  = 4000 // RAG 注入上下文的最大字符数
)

// KnowledgeService 知识库业务。
type KnowledgeService struct {
	fileRepo     *repository.FileRepo
	chunkRepo    *repository.KnowledgeRepo
	ai           *AIService
	filesDir     string // runtime/files 绝对路径
}

// NewKnowledgeService 构造。filesDir 是上传文件的存储目录。
func NewKnowledgeService(fr *repository.FileRepo, cr *repository.KnowledgeRepo, ai *AIService, filesDir string) *KnowledgeService {
	return &KnowledgeService{
		fileRepo:  fr,
		chunkRepo: cr,
		ai:        ai,
		filesDir:  filesDir,
	}
}

// ============================================================================
// 上传 / 粘贴
// ============================================================================

// UploadResult 上传结果。
type UploadResult struct {
	File       *models.KnowledgeFile `json:"file"`
	ChunkCount int                   `json:"chunk_count"`
	Warning    string                `json:"warning,omitempty"` // 部分失败的警告
}

// IngestFile 处理上传的文件（multipart 已保存到临时路径）。
// filename 是原始文件名，src 是可读的文件数据。
func (s *KnowledgeService) IngestFile(filename string, fileSize int64, src io.Reader, userID uint) (*UploadResult, error) {
	ext := strings.ToLower(filepath.Ext(filename))
	ft := supportedFileType(ext)
	if ft == "" {
		return nil, fmt.Errorf("暂不支持 .%s 格式（支持 txt/md/csv/docx），可直接粘贴文本", strings.TrimPrefix(ext, "."))
	}

	// 1. 创建文件记录（status=processing）
	file := &models.KnowledgeFile{
		Title:      strings.TrimSuffix(filename, filepath.Ext(filename)),
		Filename:   filename,
		FileType:   strings.TrimPrefix(ext, "."),
		FileSize:   fileSize,
		Status:     models.FileStatusProcessing,
		CreatedByMixin: models.CreatedByMixin{CreatedBy: userID},
	}
	if err := s.fileRepo.Create(file); err != nil {
		return nil, fmt.Errorf("创建文件记录失败: %w", err)
	}

	// 2. 保存到 runtime/files/
	if err := os.MkdirAll(s.filesDir, 0755); err != nil {
		s.markFailed(file, "创建文件目录失败: "+err.Error())
		return nil, fmt.Errorf("创建文件目录失败: %w", err)
	}
	storedName := fmt.Sprintf("%d_%s", file.ID, filename)
	storedPath := filepath.Join(s.filesDir, storedName)
	dst, err := os.Create(storedPath)
	if err != nil {
		s.markFailed(file, "保存文件失败: "+err.Error())
		return nil, fmt.Errorf("保存文件失败: %w", err)
	}
	written, err := io.Copy(dst, src)
	dst.Close()
	if err != nil {
		s.markFailed(file, "写入文件失败: "+err.Error())
		return nil, fmt.Errorf("写入文件失败: %w", err)
	}
	file.FileSize = written
	file.StoredPath = storedPath

	// 3. 解析 + 切片 + 向量化
	return s.processFile(file, storedPath)
}

// IngestText 处理粘贴的纯文本。
func (s *KnowledgeService) IngestText(title, content string, userID uint) (*UploadResult, error) {
	content = strings.TrimSpace(content)
	if content == "" {
		return nil, errors.New("文本内容不能为空")
	}
	if title == "" {
		title = "粘贴文本"
	}

	file := &models.KnowledgeFile{
		Title:        title,
		Filename:     title + ".txt",
		FileType:     "paste",
		FileSize:     int64(len(content)),
		Status:       models.FileStatusProcessing,
		CreatedByMixin: models.CreatedByMixin{CreatedBy: userID},
	}
	if err := s.fileRepo.Create(file); err != nil {
		return nil, fmt.Errorf("创建文件记录失败: %w", err)
	}

	// 直接用 content 切片（不落盘）
	return s.processText(file, content)
}

// processFile 从磁盘读取文件 → 解析 → 切片 → 向量化。
func (s *KnowledgeService) processFile(file *models.KnowledgeFile, path string) (*UploadResult, error) {
	text, err := extractTextFromFile(path)
	if err != nil {
		s.markFailed(file, err.Error())
		return nil, err
	}
	return s.processText(file, text)
}

// processText 切片 → 批量 embedding → 存库。
func (s *KnowledgeService) processText(file *models.KnowledgeFile, text string) (*UploadResult, error) {
	chunks := chunkText(text, defaultChunkSize, defaultOverlap)
	if len(chunks) == 0 {
		s.markFailed(file, "文档解析后无可用的文本内容")
		return nil, errors.New("文档解析后无可用的文本内容")
	}

	// 批量 embedding（每批 embedBatchSize 条）
	allVecs := make([][]float64, 0, len(chunks))
	for i := 0; i < len(chunks); i += embedBatchSize {
		end := i + embedBatchSize
		if end > len(chunks) {
			end = len(chunks)
		}
		batch := chunks[i:end]
		embResp, err := s.ai.Embed(batch, "")
		if err != nil {
			s.markFailed(file, "生成向量失败: "+err.Error())
			return nil, fmt.Errorf("生成向量失败: %w", err)
		}
		allVecs = append(allVecs, embResp.Vectors...)
	}

	// 构造切片记录
	records := make([]models.KnowledgeChunk, len(chunks))
	for i, content := range chunks {
		embJSON, _ := json.Marshal(allVecs[i])
		records[i] = models.KnowledgeChunk{
			FileID:     file.ID,
			SourceFile: file.Filename,
			ChunkIndex: i,
			Content:    content,
			Embedding:  string(embJSON),
		}
	}
	if err := s.chunkRepo.CreateBatch(records); err != nil {
		s.markFailed(file, "保存切片失败: "+err.Error())
		return nil, fmt.Errorf("保存切片失败: %w", err)
	}

	// 更新文件状态为 ready
	file.Status = models.FileStatusReady
	file.ChunkCount = len(records)
	if err := s.fileRepo.Update(file); err != nil {
		log.Printf("[knowledge] 更新文件状态失败 file_id=%d: %v", file.ID, err)
	}

	return &UploadResult{File: file, ChunkCount: len(records)}, nil
}

// markFailed 标记文件解析失败。
func (s *KnowledgeService) markFailed(file *models.KnowledgeFile, reason string) {
	file.Status = models.FileStatusFailed
	file.ParseError = reason
	_ = s.fileRepo.Update(file)
}

// ============================================================================
// 检索
// ============================================================================

// SearchResult 检索结果项。
type SearchResult struct {
	FileID    uint    `json:"file_id"`
	SourceFile string  `json:"source_file"`
	Content   string  `json:"content"`
	Score     float64 `json:"score"` // 余弦相似度 0~1
}

// Search 语义检索：把 query 向量化后在知识库中找最相关的 topK 个切片。
func (s *KnowledgeService) Search(query string, fileID uint, topK int) ([]SearchResult, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, errors.New("查询不能为空")
	}

	// 1. query 向量化
	embResp, err := s.ai.Embed([]string{query}, "")
	if err != nil {
		return nil, fmt.Errorf("查询向量化失败: %w", err)
	}
	if len(embResp.Vectors) == 0 {
		return nil, errors.New("查询向量化返回空")
	}

	// 2. 余弦相似度检索
	results, err := s.chunkRepo.Search(embResp.Vectors[0], fileID, topK)
	if err != nil {
		return nil, fmt.Errorf("检索失败: %w", err)
	}

	// 3. 转换为 API 响应
	out := make([]SearchResult, 0, len(results))
	for _, r := range results {
		out = append(out, SearchResult{
			FileID:    r.FileID,
			SourceFile: r.SourceFile,
			Content:   r.Content,
			Score:     math.Round(r.Score*1000) / 1000, // 保留3位小数
		})
	}
	return out, nil
}

// RetrieveContext 为 RAG Chat 检索上下文文本（拼接多个切片，限制总长度）。
func (s *KnowledgeService) RetrieveContext(query string, maxChars int) (string, []SearchResult, error) {
	if maxChars <= 0 {
		maxChars = maxContextChars
	}
	results, err := s.Search(query, 0, 5)
	if err != nil {
		return "", nil, err
	}
	if len(results) == 0 {
		return "", nil, nil
	}

	var b strings.Builder
	for _, r := range results {
	 snippet := fmt.Sprintf("【来源: %s】\n%s\n\n", r.SourceFile, r.Content)
		if b.Len()+len(snippet) > maxChars {
			break
		}
		b.WriteString(snippet)
	}
	return b.String(), results, nil
}

// ============================================================================
// RAG Chat（检索增强生成）
// ============================================================================

// RAGChatAnswer RAG 对话回答。
type RAGChatAnswer struct {
	Answer   string        `json:"answer"`    // AI 生成的回答
	Sources  []SearchResult `json:"sources"`  // 引用的知识来源
	Provider AIProvider    `json:"provider"`  // 使用的 AI 厂商
	HasContext bool        `json:"has_context"` // 是否检索到相关知识
}

// RAGChat 检索增强对话：query → 检索相关知识 → 拼入 prompt → AI 生成回答。
//
// 容错策略（gotcha #39 延伸）：AI 失败时降级返回检索到的原始上下文片段，不阻塞用户。
func (s *KnowledgeService) RAGChat(query, history string, provider AIProvider) (*RAGChatAnswer, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, errors.New("问题不能为空")
	}

	// 1. 检索相关知识
	ctx, sources, err := s.RetrieveContext(query, maxContextChars)
	if err != nil {
		return nil, fmt.Errorf("知识检索失败: %w", err)
	}

	hasContext := len(sources) > 0

	// 2. 构造 prompt
	system := `你是外贸企业的智能助手，基于企业知识库回答问题。
规则：
1. 优先使用提供的「参考资料」回答问题
2. 如果资料中没有相关信息，明确说明"知识库中暂无相关内容"，再凭通用知识回答
3. 回答简洁专业，适当标注信息来源
4. 用中文回答`

	var userPrompt string
	if hasContext {
		userPrompt = fmt.Sprintf("# 参考资料\n%s\n\n# 历史对话\n%s\n\n# 当前问题\n%s", ctx, history, query)
	} else {
		userPrompt = fmt.Sprintf("（知识库中暂无相关内容，请凭通用知识回答）\n\n# 历史对话\n%s\n\n# 问题\n%s", history, query)
	}

	// 3. 调 AI
	resp, err := s.ai.Chat(ChatRequest{
		Provider: provider,
		Messages: []ChatMessage{
			{Role: "system", Content: system},
			{Role: "user", Content: userPrompt},
		},
		Temperature: 0.4,
	})
	if err != nil {
		// AI 失败降级：返回检索到的原始上下文
		if hasContext {
			var b strings.Builder
			b.WriteString("AI 服务暂时不可用，以下是从知识库检索到的相关内容：\n\n")
			for i, src := range sources {
				b.WriteString(fmt.Sprintf("%d. 【%s】\n%s\n\n", i+1, src.SourceFile, src.Content))
			}
			return &RAGChatAnswer{
				Answer:    b.String(),
				Sources:   sources,
				HasContext: true,
			}, nil
		}
		return nil, fmt.Errorf("AI 回答失败: %w", err)
	}

	return &RAGChatAnswer{
		Answer:    resp.Content,
		Sources:   sources,
		Provider:  resp.Provider,
		HasContext: hasContext,
	}, nil
}

// ============================================================================
// 文件管理
// ============================================================================

// ListFiles 文件列表。
func (s *KnowledgeService) ListFiles(p repository.FileListParams) (*repository.FileListResult, error) {
	return s.fileRepo.List(p)
}

// GetFile 文件详情（含切片预览）。
func (s *KnowledgeService) GetFile(id uint) (*models.KnowledgeFile, int64, error) {
	f, err := s.fileRepo.FindByID(id)
	if err != nil {
		return nil, 0, err
	}
	count, _ := s.chunkRepo.CountByFile(id)
	return f, count, nil
}

// DeleteFile 删除文件 + 关联切片 + 物理文件。
func (s *KnowledgeService) DeleteFile(id uint) error {
	f, err := s.fileRepo.FindByID(id)
	if err != nil {
		return fmt.Errorf("文件不存在: %w", err)
	}

	// 1. 删除关联切片（物理删除——切片是派生数据）
	if err := s.chunkRepo.DeleteByFile(id); err != nil {
		return fmt.Errorf("删除切片失败: %w", err)
	}
	// 2. 软删除文件记录
	if err := s.fileRepo.SoftDelete(id); err != nil {
		return fmt.Errorf("删除文件记录失败: %w", err)
	}
	// 3. 删除物理文件（失败不影响数据已删，只记日志）
	if f.StoredPath != "" {
		if err := os.Remove(f.StoredPath); err != nil && !os.IsNotExist(err) {
			log.Printf("[knowledge] 删除物理文件失败 path=%s: %v", f.StoredPath, err)
		}
	}
	return nil
}

// KnowledgeStats 知识库总览统计。
type KnowledgeStats struct {
	FileCount   int64 `json:"file_count"`   // 文件总数
	ChunkCount  int64 `json:"chunk_count"`  // 切片总数
	ReadyCount  int64 `json:"ready_count"`  // 可检索文件数
	FailedCount int64 `json:"failed_count"` // 失败文件数
}

// Stats 知识库统计。
func (s *KnowledgeService) Stats() (*KnowledgeStats, error) {
	var stats KnowledgeStats
	var err error

	stats.FileCount, err = s.countByStatus("")
	if err != nil {
		return nil, err
	}
	stats.ReadyCount, _ = s.countByStatus(models.FileStatusReady)
	stats.FailedCount, _ = s.countByStatus(models.FileStatusFailed)
	stats.ChunkCount, _ = s.chunkRepo.AllChunkCount()
	return &stats, nil
}

func (s *KnowledgeService) countByStatus(status models.FileStatus) (int64, error) {
	q := s.fileRepo.DB.Model(&models.KnowledgeFile{})
	if status != "" {
		q = q.Where("status = ?", status)
	}
	var n int64
	return n, q.Count(&n).Error
}
