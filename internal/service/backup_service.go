// Package service — 数据备份与恢复。
//
// 备份策略（规范 §6.7 数据安全）：
//   - 使用 SQLite `VACUUM INTO` 生成一致性快照（自动合并 WAL），无需停服
//   - 将快照 trademind.db + runtime/files/（知识库附件）+ manifest 打包为 zip
//   - 存放于 runtime/backups/
//
// 恢复策略（`--restore <backup.zip>` 标志，HTTP 服务启动前执行）：
//   - 原库自动备份为 trademind.db.bak.<ts> 兜底
//   - 解压 zip → 替换 trademind.db + runtime/files/（含 zip-slip 防护）
//
// 「数据 100% 本地」是产品核心承诺，可备份/可恢复是企业私有化的底线能力。
package service

import (
	"archive/zip"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"gorm.io/gorm"
)

// BackupInfo 备份文件元信息（列表/详情用）。
type BackupInfo struct {
	Filename  string    `json:"filename"`
	Size      int64     `json:"size"`
	CreatedAt time.Time `json:"created_at"`
}

// backupManifest 打包在 zip 内的清单，便于恢复时校验。
type backupManifest struct {
	App       string `json:"app"`
	Version   string `json:"version"`
	CreatedAt string `json:"created_at"`
	DBFile    string `json:"db_file"`
	FileCount int    `json:"file_count"`
}

// manifestName zip 内清单文件名。
const manifestName = "manifest.json"

// 数据库文件在 zip 内的固定名称（恢复时按此查找）。
const zipDBName = "trademind.db"

// 手动备份文件名前缀（用户在设置页点「立即备份」生成）。
const backupPrefix = "trademind-backup-"

// 自动备份文件名前缀（调度器每日生成）。与手动前缀区分开，
// 保留策略（PruneAutoBackups）只清理自动备份，手动备份永不自动删除。
const autoBackupPrefix = "trademind-auto-"

// BackupService 数据备份/恢复业务逻辑。
type BackupService struct {
	db         *gorm.DB
	backupsDir string // runtime/backups 绝对路径
	filesDir   string // runtime/files 绝对路径（附件根目录）
	version    string // 应用版本（写 manifest）
}

// NewBackupService 构造。
func NewBackupService(db *gorm.DB, backupsDir, filesDir, version string) *BackupService {
	return &BackupService{db: db, backupsDir: backupsDir, filesDir: filesDir, version: version}
}

// Create 生成一份完整备份（DB 快照 + 附件），返回元信息。
func (s *BackupService) Create() (*BackupInfo, error) {
	return s.createNamed(backupPrefix)
}

// CreateAuto 生成一份自动备份（与手动备份内容相同，仅文件名前缀不同）。
func (s *BackupService) CreateAuto() (*BackupInfo, error) {
	return s.createNamed(autoBackupPrefix)
}

// createNamed 备份实现：prefix 决定文件名前缀（手动/自动）。
func (s *BackupService) createNamed(prefix string) (*BackupInfo, error) {
	if err := os.MkdirAll(s.backupsDir, 0755); err != nil {
		return nil, fmt.Errorf("创建备份目录失败: %w", err)
	}

	// 1. 生成不冲突的 zip 文件名（人类可读时间戳）
	zipName := s.uniqueName(prefix)
	zipPath := filepath.Join(s.backupsDir, zipName)

	// 2. VACUUM INTO 临时快照（内部路径，非用户输入，安全）
	tmpDB := filepath.Join(s.backupsDir, ".tmp-snapshot.db")
	_ = os.Remove(tmpDB) // 清理可能的残留
	// VACUUM INTO 目标文件必须不存在；用单引号包裹路径。
	// 路径由我们生成（runtime/backups 下），不含单引号。
	if res := s.db.Exec(fmt.Sprintf("VACUUM INTO '%s'", tmpDB)); res.Error != nil {
		_ = os.Remove(tmpDB)
		return nil, fmt.Errorf("生成数据库快照失败: %w", res.Error)
	}

	// 3. 打包 zip（快照 + 附件 + manifest 一次性写入）
	if _, err := s.writeZip(zipPath, tmpDB); err != nil {
		_ = os.Remove(tmpDB)
		_ = os.Remove(zipPath)
		return nil, fmt.Errorf("打包备份失败: %w", err)
	}
	// 无论成功与否都清理临时快照
	_ = os.Remove(tmpDB)

	st, err := os.Stat(zipPath)
	if err != nil {
		return nil, fmt.Errorf("读取备份文件失败: %w", err)
	}

	return &BackupInfo{
		Filename:  zipName,
		Size:      st.Size(),
		CreatedAt: st.ModTime(),
	}, nil
}

// writeZip 将快照 db + 附件目录 + manifest 写入 zipPath。
// 返回纳入备份的附件文件数。
func (s *BackupService) writeZip(zipPath, snapshotDB string) (int, error) {
	f, err := os.Create(zipPath)
	if err != nil {
		return 0, err
	}
	defer f.Close()

	zw := zip.NewWriter(f)
	defer zw.Close()

	// 写数据库快照（固定名 trademind.db）
	if err := addFileToZip(zw, snapshotDB, zipDBName); err != nil {
		return 0, fmt.Errorf("写入数据库快照失败: %w", err)
	}

	// 写附件目录（files/**）
	count := 0
	if info, err := os.Stat(s.filesDir); err == nil && info.IsDir() {
		err := filepath.Walk(s.filesDir, func(path string, fi os.FileInfo, err error) error {
			if err != nil || fi.IsDir() {
				return err
			}
			rel, err := filepath.Rel(s.filesDir, path)
			if err != nil {
				return err
			}
			// zip 内统一用正斜杠
			arc := filepath.ToSlash(filepath.Join("files", rel))
			if err := addFileToZip(zw, path, arc); err != nil {
				return err
			}
			count++
			return nil
		})
		if err != nil {
			return count, fmt.Errorf("打包附件失败: %w", err)
		}
	}

	// 写 manifest
	manifest := backupManifest{
		App:       "TradeMind AI",
		Version:   s.version,
		CreatedAt: time.Now().Format(time.RFC3339),
		DBFile:    zipDBName,
		FileCount: count,
	}
	mb, _ := json.MarshalIndent(manifest, "", "  ")
	mw, err := zw.Create(manifestName)
	if err != nil {
		return count, err
	}
	if _, err := mw.Write(mb); err != nil {
		return count, err
	}

	return count, nil
}

// addFileToZip 将磁盘文件 src 加入 zip，归档名为 arcName。
func addFileToZip(zw *zip.Writer, src, arcName string) error {
	r, err := os.Open(src)
	if err != nil {
		return err
	}
	defer r.Close()

	w, err := zw.Create(arcName)
	if err != nil {
		return err
	}
	_, err = io.Copy(w, r)
	return err
}

// uniqueName 生成不冲突的备份文件名 <prefix>YYYYMMDD-HHMMSS.zip。
func (s *BackupService) uniqueName(prefix string) string {
	base := prefix + time.Now().Format("20060102-150405")
	name := base + ".zip"
	for i := 2; ; i++ {
		if _, err := os.Stat(filepath.Join(s.backupsDir, name)); os.IsNotExist(err) {
			return name
		}
		name = fmt.Sprintf("%s-%d.zip", base, i)
	}
}

// List 列出全部备份，按时间倒序（最新在前）。
func (s *BackupService) List() ([]BackupInfo, error) {
	entries, err := os.ReadDir(s.backupsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []BackupInfo{}, nil
		}
		return nil, fmt.Errorf("读取备份目录失败: %w", err)
	}
	var out []BackupInfo
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".zip") {
			continue
		}
		// 跳过临时文件
		if strings.HasPrefix(e.Name(), ".") {
			continue
		}
		fi, err := e.Info()
		if err != nil {
			continue
		}
		out = append(out, BackupInfo{
			Filename:  e.Name(),
			Size:      fi.Size(),
			CreatedAt: fi.ModTime(),
		})
	}
	if out == nil {
		out = []BackupInfo{}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.After(out[j].CreatedAt) })
	return out, nil
}

// PruneAutoBackups 保留策略：删除「自动备份」中修改时间早于 retentionDays 天前的文件。
// 手动备份（trademind-backup-*）永不自动删除——那是老板亲手点的。
// retentionDays <= 0 视为删除全部自动备份。返回删除的文件数。
func (s *BackupService) PruneAutoBackups(retentionDays int) (int, error) {
	entries, err := os.ReadDir(s.backupsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil // 目录都没有，无事可做
		}
		return 0, fmt.Errorf("读取备份目录失败: %w", err)
	}
	if retentionDays < 0 {
		retentionDays = 0
	}
	cutoff := time.Now().AddDate(0, 0, -retentionDays)
	pruned := 0
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasPrefix(name, autoBackupPrefix) || !strings.HasSuffix(name, ".zip") {
			continue
		}
		fi, err := e.Info()
		if err != nil {
			continue // 单个文件 stat 失败跳过，不阻塞整体清理
		}
		if fi.ModTime().Before(cutoff) {
			if err := os.Remove(filepath.Join(s.backupsDir, name)); err != nil {
				log.Printf("[backup] 清理过期自动备份失败 [%s]: %v", name, err)
				continue
			}
			pruned++
		}
	}
	return pruned, nil
}

// LatestAutoBackup 返回最新一份自动备份。没有自动备份时 ok=false。
// 调度器启动时用它判断是否需要补跑（桌面应用深夜大概率没开机，cron 常漏跑）。
func (s *BackupService) LatestAutoBackup() (BackupInfo, bool, error) {
	list, err := s.List()
	if err != nil {
		return BackupInfo{}, false, err
	}
	// List 已按时间倒序，取第一个自动备份即可
	for _, b := range list {
		if strings.HasPrefix(b.Filename, autoBackupPrefix) {
			return b, true, nil
		}
	}
	return BackupInfo{}, false, nil
}

// Path 返回备份文件的绝对路径。filename 必须是纯文件名（防目录穿越）。
func (s *BackupService) Path(filename string) (string, error) {
	if !s.isSafeName(filename) {
		return "", errors.New("非法的备份文件名")
	}
	full := filepath.Join(s.backupsDir, filename)
	if _, err := os.Stat(full); err != nil {
		return "", errors.New("备份文件不存在")
	}
	return full, nil
}

// Delete 删除一份备份。filename 必须是纯文件名。
func (s *BackupService) Delete(filename string) error {
	if !s.isSafeName(filename) {
		return errors.New("非法的备份文件名")
	}
	full := filepath.Join(s.backupsDir, filename)
	if _, err := os.Stat(full); err != nil {
		return errors.New("备份文件不存在")
	}
	return os.Remove(full)
}

// isSafeName 校验文件名：非空、无路径分隔符、无 .. 、以 .zip 结尾、不以 . 开头。
func (s *BackupService) isSafeName(name string) bool {
	if name == "" || strings.HasPrefix(name, ".") {
		return false
	}
	if !strings.HasSuffix(name, ".zip") {
		return false
	}
	if strings.ContainsAny(name, `/\`) {
		return false
	}
	if name == ".." || strings.Contains(name, "..") {
		return false
	}
	return true
}

// RestoreFromZip 从备份 zip 恢复数据。
// ⚠️ 必须在 HTTP 服务启动、数据库打开前调用（替换活动 db 文件不安全）。
// 参数：
//   - zipPath   备份 zip 路径
//   - dbPath    目标数据库路径（runtime/trademind.db）
//   - filesDir  目标附件目录（runtime/files）
//
// 恢复前自动把现有 db 备份为 dbPath.bak.<ts>。
func RestoreFromZip(zipPath, dbPath, filesDir string) error {
	if _, err := os.Stat(zipPath); err != nil {
		return fmt.Errorf("备份文件不存在: %w", err)
	}

	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return fmt.Errorf("打开备份失败: %w", err)
	}
	defer r.Close()

	// 定位 db 与 manifest
	var dbFound bool
	for _, f := range r.File {
		if f.Name == zipDBName {
			dbFound = true
			break
		}
	}
	if !dbFound {
		return errors.New("备份文件损坏：缺少 trademind.db")
	}

	// 1. 现有库兜底备份（若存在）
	if _, err := os.Stat(dbPath); err == nil {
		bak := dbPath + ".bak." + time.Now().Format("20060102-150405")
		if err := copyFile(dbPath, bak); err != nil {
			return fmt.Errorf("备份现有库失败: %w", err)
		}
		// 清理 WAL/SHM（旧库的，新库会自己重建）
		_ = os.Remove(dbPath + "-wal")
		_ = os.Remove(dbPath + "-shm")
	}

	// 确保目录存在
	if err := os.MkdirAll(filepath.Dir(dbPath), 0755); err != nil {
		return err
	}
	if err := os.MkdirAll(filesDir, 0755); err != nil {
		return err
	}

	// 2. 解压全部条目
	for _, f := range r.File {
		if f.FileInfo().IsDir() {
			continue
		}
		// zip-slip 防护：目标必须落在 filesDir 下（manifest/db 特殊处理）
		switch f.Name {
		case zipDBName:
			if err := extractZipEntry(f, dbPath); err != nil {
				return fmt.Errorf("恢复数据库失败: %w", err)
			}
		case manifestName:
			// manifest 不落盘，跳过
			continue
		default:
			// 附件：files/** → filesDir/**
			if !strings.HasPrefix(f.Name, "files/") {
				continue // 忽略未知条目
			}
			rel := strings.TrimPrefix(f.Name, "files/")
			// 防 zip-slip
			target := filepath.Join(filesDir, rel)
			if !isWithin(filesDir, target) {
				return fmt.Errorf("非法路径的附件条目: %s", f.Name)
			}
			if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
				return err
			}
			if err := extractZipEntry(f, target); err != nil {
				return fmt.Errorf("恢复附件失败 [%s]: %w", f.Name, err)
			}
		}
	}

	return nil
}

// extractZipEntry 把一个 zip 条目解压到 dstPath。
func extractZipEntry(f *zip.File, dstPath string) error {
	rc, err := f.Open()
	if err != nil {
		return err
	}
	defer rc.Close()

	w, err := os.Create(dstPath)
	if err != nil {
		return err
	}
	defer w.Close()

	_, err = io.Copy(w, rc)
	return err
}

// copyFile 拷贝单个文件。
func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	return err
}

// isWithin 判断 target 是否在 base 目录内（防 zip-slip）。
func isWithin(base, target string) bool {
	absBase, _ := filepath.Abs(base)
	absTarget, _ := filepath.Abs(target)
	rel, err := filepath.Rel(absBase, absTarget)
	if err != nil {
		return false
	}
	return rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}
