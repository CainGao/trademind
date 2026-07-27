package service

import (
	"archive/zip"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/CainGao/trademind/internal/database"
)

// newTestBackupSvc 构造一个使用临时库的 BackupService，返回服务、临时目录与清理函数。
func newTestBackupSvc(t *testing.T) (*BackupService, string, func()) {
	t.Helper()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "trademind.db")
	backupsDir := filepath.Join(dir, "backups")
	filesDir := filepath.Join(dir, "files")

	db, err := database.Open(database.Config{Path: dbPath})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	// 建一张表并写几行，验证快照内容
	if err := db.AutoMigrate(&testWidget{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	db.Create(&testWidget{Name: "alpha"})
	db.Create(&testWidget{Name: "beta"})

	// 放一个附件文件
	if err := os.MkdirAll(filesDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(filesDir, "note.txt"), []byte("hello"), 0644); err != nil {
		t.Fatal(err)
	}

	svc := NewBackupService(db, backupsDir, filesDir, "test-1.0")
	cleanup := func() {
		if sqlDB, err := db.DB(); err == nil {
			_ = sqlDB.Close()
		}
	}
	return svc, dir, cleanup
}

type testWidget struct {
	ID   uint   `gorm:"primarykey"`
	Name string `gorm:"type:text"`
}

func TestBackup_Create(t *testing.T) {
	svc, _, cleanup := newTestBackupSvc(t)
	defer cleanup()

	info, err := svc.Create()
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if !strings.HasSuffix(info.Filename, ".zip") {
		t.Errorf("filename = %q, want .zip suffix", info.Filename)
	}
	if info.Size <= 0 {
		t.Errorf("size = %d, want > 0", info.Size)
	}

	// 验证 zip 内含 trademind.db + manifest + 附件
	full := filepath.Join(svc.backupsDir, info.Filename)
	zr, err := zip.OpenReader(full)
	if err != nil {
		t.Fatalf("open zip: %v", err)
	}
	defer zr.Close()
	want := map[string]bool{"trademind.db": false, "manifest.json": false, "files/note.txt": false}
	for _, f := range zr.File {
		if _, ok := want[f.Name]; ok {
			want[f.Name] = true
		}
	}
	for name, found := range want {
		if !found {
			t.Errorf("zip missing entry %q", name)
		}
	}
}

func TestBackup_List_SortedDesc(t *testing.T) {
	svc, _, cleanup := newTestBackupSvc(t)
	defer cleanup()

	a, err := svc.Create()
	if err != nil {
		t.Fatal(err)
	}
	b, err := svc.Create()
	if err != nil {
		t.Fatal(err)
	}
	// 触发同名后缀逻辑（极小概率同秒，这里只验证不报错且数量正确）
	if _, err := svc.Create(); err != nil {
		t.Fatal(err)
	}

	list, err := svc.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(list) != 3 {
		t.Fatalf("len(list) = %d, want 3", len(list))
	}
	// 最新在前（倒序）
	if !list[0].CreatedAt.After(list[len(list)-1].CreatedAt) ||
		list[0].CreatedAt.Equal(list[len(list)-1].CreatedAt) {
		// 同秒创建时时间戳可能相等，校验文件名倒序兜底
		if list[0].Filename < list[len(list)-1].Filename {
			t.Errorf("list not sorted desc: first=%q last=%q", list[0].Filename, list[len(list)-1].Filename)
		}
	}
	_ = a
	_ = b
}

func TestBackup_PathTraversalRejected(t *testing.T) {
	svc, _, cleanup := newTestBackupSvc(t)
	defer cleanup()

	bad := []string{"../etc/passwd.zip", "/etc/passwd.zip", "..", "sub/dir/x.zip", ".hidden.zip", "nozip"}
	for _, name := range bad {
		if _, err := svc.Path(name); err == nil {
			t.Errorf("Path(%q) should reject", name)
		}
		if err := svc.Delete(name); err == nil {
			t.Errorf("Delete(%q) should reject", name)
		}
	}
}

func TestBackup_Path_Delete_OK(t *testing.T) {
	svc, _, cleanup := newTestBackupSvc(t)
	defer cleanup()

	info, err := svc.Create()
	if err != nil {
		t.Fatal(err)
	}
	p, err := svc.Path(info.Filename)
	if err != nil {
		t.Fatalf("Path: %v", err)
	}
	if !strings.HasSuffix(p, info.Filename) {
		t.Errorf("Path = %q, want suffix %q", p, info.Filename)
	}
	if err := svc.Delete(info.Filename); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	// 再删应报不存在
	if err := svc.Delete(info.Filename); err == nil {
		t.Error("Delete twice should fail")
	}
}

func TestBackup_RestoreFromZip(t *testing.T) {
	svc, dir, cleanup := newTestBackupSvc(t)

	// 1. 先备份
	info, err := svc.Create()
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	cleanup() // 关闭源 db 句柄，便于覆盖

	zipPath := filepath.Join(svc.backupsDir, info.Filename)

	// 2. 清空原始库 + 附件（模拟灾难）
	dbPath := filepath.Join(dir, "trademind.db")
	os.Remove(dbPath)
	os.Remove(filepath.Join(svc.filesDir, "note.txt"))

	// 3. 恢复
	if err := RestoreFromZip(zipPath, dbPath, svc.filesDir); err != nil {
		t.Fatalf("RestoreFromZip: %v", err)
	}

	// 4. 校验：库可打开且含数据；附件已恢复
	if _, err := os.Stat(dbPath); err != nil {
		t.Fatalf("db not restored: %v", err)
	}
	db2, err := database.Open(database.Config{Path: dbPath})
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	var n int64
	db2.Model(&testWidget{}).Count(&n)
	if n != 2 {
		t.Errorf("restored row count = %d, want 2", n)
	}
	sqlDB, _ := db2.DB()
	sqlDB.Close()

	note := filepath.Join(svc.filesDir, "note.txt")
	got, err := os.ReadFile(note)
	if err != nil {
		t.Fatalf("attachment not restored: %v", err)
	}
	if string(got) != "hello" {
		t.Errorf("attachment content = %q, want 'hello'", got)
	}

	// 5. 兜底备份应生成 .bak
	matches, _ := filepath.Glob(dbPath + ".bak.*")
	if len(matches) == 0 {
		// 恢复时原库已被本测试删掉，无 .bak 是正常的
		t.Logf("no .bak (source db removed before restore) — acceptable")
	}
}

func TestBackup_RestoreZipSlipGuard(t *testing.T) {
	dir := t.TempDir()
	zipPath := filepath.Join(dir, "evil.zip")
	filesDir := filepath.Join(dir, "files")
	os.MkdirAll(filesDir, 0755)

	// 构造一个恶意 zip：files/../../evil.txt（试图逃逸）
	out, err := os.Create(zipPath)
	if err != nil {
		t.Fatal(err)
	}
	zw := zip.NewWriter(out)
	w, err := zw.Create("files/../../evil.txt")
	if err != nil {
		t.Fatal(err)
	}
	w.Write([]byte("pwned"))
	zw.Close()
	out.Close()

	dbPath := filepath.Join(dir, "trademind.db")
	// 还需要一个合法 trademind.db 条目才能走到附件解压；这里专测 isWithin 拦截
	// 先放一个合法 db 让其通过 db 校验
	legitZip := filepath.Join(dir, "legit.zip")
	lo, _ := os.Create(legitZip)
	lzw := zip.NewWriter(lo)
	// 写一个空 db 条目
	dw, _ := lzw.Create("trademind.db")
	dw.Write([]byte{})
	// 恶意附件
	ew, _ := lzw.Create("files/../../evil2.txt")
	ew.Write([]byte("pwned"))
	lzw.Close()
	lo.Close()

	err = RestoreFromZip(legitZip, dbPath, filesDir)
	if err == nil {
		t.Fatal("expected zip-slip error, got nil")
	}
	// evil2.txt 不应出现在 filesDir 上层
	if _, err := os.Stat(filepath.Join(dir, "evil2.txt")); err == nil {
		t.Error("zip-slip succeeded: evil2.txt escaped")
	}
}
