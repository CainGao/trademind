package service

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"

	"github.com/CainGao/trademind/internal/models"
	"github.com/CainGao/trademind/internal/repository"
)

// setupKnowledgeTestDB 按测试函数隔离的内存库（gotcha #85）。
func setupKnowledgeTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("打开内存库失败: %v", err)
	}
	if err := db.AutoMigrate(&models.KnowledgeFile{}, &models.KnowledgeChunk{}); err != nil {
		t.Fatalf("迁移失败: %v", err)
	}
	return db
}

// newTestKnowledgeService 构造测试用 KnowledgeService（AI 为 nil，测试场景均不触达 Embedding）。
func newTestKnowledgeService(t *testing.T) (*KnowledgeService, *gorm.DB, string) {
	t.Helper()
	db := setupKnowledgeTestDB(t)
	filesDir := t.TempDir()
	svc := NewKnowledgeService(
		repository.NewFileRepo(db),
		repository.NewKnowledgeRepo(db),
		nil, // AIService —— 测试场景不到 Embedding 路径
		filesDir,
	)
	return svc, db, filesDir
}

// TestReembedFile_FileNotFound 不存在的文件 ID 应报错。
func TestReembedFile_FileNotFound(t *testing.T) {
	svc, _, _ := newTestKnowledgeService(t)
	if _, err := svc.ReembedFile(999); err == nil {
		t.Fatal("不存在的文件应返回错误")
	}
}

// TestReembedFile_PasteNoPhysicalFile 粘贴文本（StoredPath 为空）应拒绝重试。
func TestReembedFile_PasteNoPhysicalFile(t *testing.T) {
	svc, db, _ := newTestKnowledgeService(t)
	paste := &models.KnowledgeFile{
		Title:    "粘贴文本",
		Filename: "粘贴文本.txt",
		FileType: "paste",
		Status:   models.FileStatusFailed,
	}
	if err := db.Create(paste).Error; err != nil {
		t.Fatalf("创建粘贴记录失败: %v", err)
	}
	_, err := svc.ReembedFile(paste.ID)
	if err == nil {
		t.Fatal("粘贴文本重试应返回错误")
	}
	if want := "无物理文件"; !strings.Contains(err.Error(), want) {
		t.Errorf("错误信息应包含 %q，实际: %v", want, err)
	}
}

// TestReembedFile_PhysicalFileMissing 物理文件丢失应 markFailed 并报错。
func TestReembedFile_PhysicalFileMissing(t *testing.T) {
	svc, db, filesDir := newTestKnowledgeService(t)
	ghost := &models.KnowledgeFile{
		Title:      "丢失文件",
		Filename:   "ghost.txt",
		FileType:   "txt",
		Status:     models.FileStatusFailed,
		StoredPath: filepath.Join(filesDir, "9999_ghost.txt"), // 不存在
	}
	if err := db.Create(ghost).Error; err != nil {
		t.Fatalf("创建记录失败: %v", err)
	}
	if _, err := svc.ReembedFile(ghost.ID); err == nil {
		t.Fatal("物理文件丢失应返回错误")
	}
	// 验证 markFailed 已写入 parse_error
	var after models.KnowledgeFile
	if err := db.First(&after, ghost.ID).Error; err != nil {
		t.Fatalf("回查失败: %v", err)
	}
	if after.ParseError == "" {
		t.Error("物理文件丢失后 parse_error 应有值")
	}
}

// TestReembedFile_ClearsOldChunks 重试前应清除旧切片（幂等），
// 且空内容文件重试后仍为 failed 并带上新的失败原因（不触达 nil AI）。
func TestReembedFile_ClearsOldChunks(t *testing.T) {
	svc, db, filesDir := newTestKnowledgeService(t)
	// 物理文件：内容为空（解析后无可用文本 → markFailed，不经过 Embedding）
	path := filepath.Join(filesDir, "1_empty.txt")
	if err := os.WriteFile(path, []byte("   "), 0644); err != nil {
		t.Fatalf("写测试文件失败: %v", err)
	}
	file := &models.KnowledgeFile{
		Title:      "空文件",
		Filename:   "empty.txt",
		FileType:   "txt",
		Status:     models.FileStatusFailed,
		StoredPath: path,
	}
	if err := db.Create(file).Error; err != nil {
		t.Fatalf("创建记录失败: %v", err)
	}
	// 残留的旧切片（模拟上次失败前的遗留）
	old := []models.KnowledgeChunk{
		{FileID: file.ID, SourceFile: file.Filename, ChunkIndex: 0, Content: "old", Embedding: "[1]"},
		{FileID: file.ID, SourceFile: file.Filename, ChunkIndex: 1, Content: "old2", Embedding: "[2]"},
	}
	if err := db.Create(&old).Error; err != nil {
		t.Fatalf("创建旧切片失败: %v", err)
	}
	if _, err := svc.ReembedFile(file.ID); err == nil {
		t.Fatal("空内容文件重试应返回错误")
	}
	var chunkCount int64
	db.Model(&models.KnowledgeChunk{}).Where("file_id = ?", file.ID).Count(&chunkCount)
	if chunkCount != 0 {
		t.Errorf("重试后旧切片应被清除，实际剩余 %d 条", chunkCount)
	}
	var after models.KnowledgeFile
	if err := db.First(&after, file.ID).Error; err != nil {
		t.Fatalf("回查失败: %v", err)
	}
	if after.Status != models.FileStatusFailed {
		t.Errorf("空文件重试后状态应为 failed，实际: %s", after.Status)
	}
	if after.ParseError == "" {
		t.Error("空文件重试后 parse_error 应有值")
	}
}
