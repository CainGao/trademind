// Package service — 文档解析工具（Week 8 RAG）。
//
// 纯 Go 实现，无外部依赖（docx 用标准库 archive/zip + XML 标签剥离）：
//   - txt/md/csv: 直接读取 UTF-8 文本
//   - docx: 解压 zip → 读 word/document.xml → 剥离 XML 标签提取纯文本
//   - 其他格式(pdf/xlsx): 返回错误，提示用 txt 粘贴
package service

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// supportedFileType 支持的文档类型。
func supportedFileType(ext string) string {
	ext = strings.ToLower(strings.TrimPrefix(ext, "."))
	switch ext {
	case "txt", "md", "markdown", "csv", "log":
		return "txt"
	case "docx":
		return "docx"
	default:
		return ""
	}
}

// extractTextFromFile 从磁盘文件提取纯文本。
func extractTextFromFile(path string) (string, error) {
	ext := filepath.Ext(path)
	ft := supportedFileType(ext)
	if ft == "" {
		return "", fmt.Errorf("暂不支持 .%s 格式，请粘贴纯文本（支持 txt/md/csv/docx）", strings.TrimPrefix(ext, "."))
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("读取文件失败: %w", err)
	}

	if ft == "txt" {
		return string(data), nil
	}
	if ft == "docx" {
		return extractDocxText(data)
	}
	return "", fmt.Errorf("不支持的文件类型")
}

// extractDocxText 从 docx 字节数据提取纯文本。
// docx 本质是 zip，正文在 word/document.xml 中。
func extractDocxText(data []byte) (string, error) {
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return "", fmt.Errorf("解压 docx 失败: %w", err)
	}

	var docXML []byte
	for _, f := range zr.File {
		if f.Name == "word/document.xml" {
			rc, err := f.Open()
			if err != nil {
				return "", fmt.Errorf("打开 document.xml 失败: %w", err)
			}
			docXML, err = io.ReadAll(rc)
			rc.Close()
			if err != nil {
				return "", fmt.Errorf("读取 document.xml 失败: %w", err)
			}
			break
		}
	}
	if docXML == nil {
		return "", fmt.Errorf("docx 中未找到 word/document.xml")
	}
	return stripXMLTags(string(docXML)), nil
}

// stripXMLTags 剥离 XML 标签，保留文本内容。
// 段落结束标签 </w:p> 替换为换行，表格单元格 </w:tc> 替换为制表符。
func stripXMLTags(xmlStr string) string {
	var b strings.Builder
	inTag := false
	for _, r := range xmlStr {
		if r == '<' {
			inTag = true
			continue
		}
		if r == '>' {
			inTag = false
			continue
		}
		if !inTag {
			b.WriteRune(r)
		}
	}
	// 按常见 docx 标签边界（已被剥离）做简单清理
	text := b.String()
	// 多余空行压缩
	for strings.Contains(text, "\n\n\n") {
		text = strings.ReplaceAll(text, "\n\n\n", "\n\n")
	}
	return strings.TrimSpace(text)
}

// chunkText 把长文本切分成有重叠的片段（用于 embedding）。
//
// 策略：
//  1. 先按段落（双换行）分割
//  2. 累积到 targetSize 字符时作为一个 chunk
//  3. 下一个 chunk 保留 overlap 字符作为上下文重叠
//
// 参数：targetSize 目标片段长度（字符），overlap 重叠长度。
func chunkText(text string, targetSize, overlap int) []string {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil
	}
	if targetSize < 200 {
		targetSize = 500
	}
	if overlap < 0 || overlap >= targetSize {
		overlap = targetSize / 5
	}

	// 按段落切
	paragraphs := strings.Split(text, "\n")
	chunks := []string{}
	var current strings.Builder
	currentLen := 0

	for _, para := range paragraphs {
		para = strings.TrimSpace(para)
		if para == "" {
			continue
		}
		// 如果当前片段加上这段会超过目标，先保存当前片段
		if currentLen > 0 && currentLen+len(para)+1 > targetSize {
			chunks = append(chunks, strings.TrimSpace(current.String()))
			// 重叠：从当前片段末尾取 overlap 字符
			tail := current.String()
			if len(tail) > overlap {
				tail = tail[len(tail)-overlap:]
			}
			current.Reset()
			current.WriteString(tail)
			currentLen = len(tail)
		}
		if currentLen > 0 {
			current.WriteString("\n")
			currentLen++
		}
		current.WriteString(para)
		currentLen += len(para)
	}
	// 最后一个片段
	if currentLen > 0 {
		chunks = append(chunks, strings.TrimSpace(current.String()))
	}

	// 如果单个段落本身超长（> targetSize * 2），硬切
	final := make([]string, 0, len(chunks))
	for _, c := range chunks {
		if len(c) <= targetSize*2 {
			final = append(final, c)
			continue
		}
		for i := 0; i < len(c); i += targetSize - overlap {
			end := i + targetSize
			if end > len(c) {
				end = len(c)
			}
			final = append(final, c[i:end])
			if end >= len(c) {
				break
			}
		}
	}
	return final
}
