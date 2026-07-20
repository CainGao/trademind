// Package repository — 知识库文件 + 切片数据访问（Week 8 RAG）。
//
// 职责：纯 CRUD + 向量检索（余弦相似度）。
// - FileRepo 管理 knowledge_files 表（上传的原始文件元数据）
// - KnowledgeRepo 管理 knowledge_chunks 表（切片 + embedding）
//
// embedding 存储为 JSON 数组字符串（纯 Go，无 sqlite-vec 依赖）。
// 检索时全量加载到内存计算余弦相似度——典型知识库（< 1万切片）毫秒级。
package repository

import (
	"encoding/json"
	"math"
	"sort"

	"github.com/CainGao/trademind/internal/models"
	"gorm.io/gorm"
)

// ============================================================================
// FileRepo — 知识库文件元数据
// ============================================================================

// FileRepo 知识库文件表数据访问。
type FileRepo struct {
	BaseRepo
}

func NewFileRepo(db *gorm.DB) *FileRepo {
	return &FileRepo{BaseRepo{DB: db}}
}

// Create 创建文件记录。
func (r *FileRepo) Create(f *models.KnowledgeFile) error {
	return r.DB.Create(f).Error
}

// FindByID 按主键查。
func (r *FileRepo) FindByID(id uint) (*models.KnowledgeFile, error) {
	var f models.KnowledgeFile
	if err := r.DB.First(&f, id).Error; err != nil {
		return nil, err
	}
	return &f, nil
}

// FileListParams 文件列表查询参数。
type FileListParams struct {
	Page     int
	PageSize int
	Keyword  string // 文件名模糊匹配
	FileType string // 文件类型筛选
}

// FileListResult 分页结果。
type FileListResult struct {
	Total int64
	Items []models.KnowledgeFile
}

// List 分页查询文件列表。
func (r *FileRepo) List(p FileListParams) (*FileListResult, error) {
	if p.Page < 1 {
		p.Page = 1
	}
	if p.PageSize < 1 || p.PageSize > 100 {
		p.PageSize = 20
	}

	q := r.DB.Model(&models.KnowledgeFile{})
	if p.Keyword != "" {
		q = q.Where("filename LIKE ?", "%"+p.Keyword+"%")
	}
	if p.FileType != "" {
		q = q.Where("file_type = ?", p.FileType)
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, err
	}

	var items []models.KnowledgeFile
	err := q.Order("created_at DESC").
		Offset((p.Page - 1) * p.PageSize).
		Limit(p.PageSize).
		Find(&items).Error
	if err != nil {
		return nil, err
	}
	return &FileListResult{Total: total, Items: items}, nil
}

// Update 更新文件记录（如标记 parsed / chunk_count）。
func (r *FileRepo) Update(f *models.KnowledgeFile) error {
	return r.DB.Save(f).Error
}

// SoftDelete 软删除文件（关联切片由 service 层事务删除）。
func (r *FileRepo) SoftDelete(id uint) error {
	return r.DB.Delete(&models.KnowledgeFile{}, id).Error
}

// ============================================================================
// KnowledgeRepo — 知识切片 + 向量检索
// ============================================================================

// KnowledgeRepo 知识切片表数据访问 + 余弦相似度检索。
type KnowledgeRepo struct {
	BaseRepo
}

func NewKnowledgeRepo(db *gorm.DB) *KnowledgeRepo {
	return &KnowledgeRepo{BaseRepo{DB: db}}
}

// CreateBatch 批量创建切片。
func (r *KnowledgeRepo) CreateBatch(chunks []models.KnowledgeChunk) error {
	if len(chunks) == 0 {
		return nil
	}
	return r.DB.Create(&chunks).Error
}

// DeleteByFile 删除指定文件的所有切片（物理删除——切片是文件派生数据，
// 删文件时连带清除，不留软删垃圾；知识库的"删除"语义是彻底遗忘）。
func (r *KnowledgeRepo) DeleteByFile(fileID uint) error {
	return r.DB.Unscoped().Where("file_id = ?", fileID).Delete(&models.KnowledgeChunk{}).Error
}

// CountByFile 统计文件下的切片数。
func (r *KnowledgeRepo) CountByFile(fileID uint) (int64, error) {
	var n int64
	err := r.DB.Model(&models.KnowledgeChunk{}).Where("file_id = ?", fileID).Count(&n).Error
	return n, err
}

// ChunkWithScore 检索结果（切片 + 相似度分数）。
type ChunkWithScore struct {
	models.KnowledgeChunk
	Score float64 `json:"score"` // 余弦相似度 0~1
}

// Search 向量检索：用查询向量在全库（或指定文件）中找余弦相似度最高的 topK 个切片。
//
// 实现：全量加载有 embedding 的切片到内存 → 解析 JSON 向量 → 算余弦相似度 → 排序取 topK。
// 典型知识库（< 1万切片，每向量 1536 维 ≈ 12KB）总量 < 120MB，毫秒级完成。
func (r *KnowledgeRepo) Search(queryVec []float64, fileID uint, topK int) ([]ChunkWithScore, error) {
	if len(queryVec) == 0 {
		return nil, nil
	}
	if topK < 1 || topK > 50 {
		topK = 5
	}

	q := r.DB.Model(&models.KnowledgeChunk{}).Where("embedding != ''")
	if fileID > 0 {
		q = q.Where("file_id = ?", fileID)
	}

	var chunks []models.KnowledgeChunk
	if err := q.Find(&chunks).Error; err != nil {
		return nil, err
	}

	// 计算余弦相似度
	results := make([]ChunkWithScore, 0, len(chunks))
	for i := range chunks {
		vec, err := parseEmbedding(chunks[i].Embedding)
		if err != nil || len(vec) != len(queryVec) {
			continue // 维度不匹配（不同模型 embedding 维度可能不同），跳过
		}
		score := cosineSimilarity(queryVec, vec)
		results = append(results, ChunkWithScore{KnowledgeChunk: chunks[i], Score: score})
	}

	// 按分数降序排序
	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})

	// 取 topK
	if len(results) > topK {
		results = results[:topK]
	}
	return results, nil
}

// AllChunkCount 全库切片总数（统计用）。
func (r *KnowledgeRepo) AllChunkCount() (int64, error) {
	var n int64
	err := r.DB.Model(&models.KnowledgeChunk{}).Count(&n).Error
	return n, err
}

// parseEmbedding 把 JSON 数组字符串解析成 []float64。
func parseEmbedding(s string) ([]float64, error) {
	var vec []float64
	if err := json.Unmarshal([]byte(s), &vec); err != nil {
		return nil, err
	}
	return vec, nil
}

// cosineSimilarity 计算两个向量的余弦相似度（值域 -1~1，越高越相似）。
// cos(A,B) = (A·B) / (|A| × |B|)
func cosineSimilarity(a, b []float64) float64 {
	var dot, normA, normB float64
	for i := 0; i < len(a); i++ {
		dot += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}
	if normA == 0 || normB == 0 {
		return 0
	}
	return dot / (math.Sqrt(normA) * math.Sqrt(normB))
}
