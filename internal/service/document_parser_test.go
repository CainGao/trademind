package service

import (
	"archive/zip"
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

// ===== supportedFileType =====

func TestSupportedFileType(t *testing.T) {
	cases := []struct {
		ext  string
		want string
	}{
		{"txt", "txt"},
		{".txt", "txt"},
		{"md", "txt"},
		{"markdown", "txt"},
		{"csv", "txt"},
		{"log", "txt"},
		{".CSV", "txt"}, // case insensitive
		{"docx", "docx"},
		{"DOCX", "docx"},
		{"pdf", ""},
		{"xlsx", ""},
		{"", ""},
		{"xyz", ""},
	}
	for _, c := range cases {
		got := supportedFileType(c.ext)
		if got != c.want {
			t.Errorf("supportedFileType(%q) = %q, want %q", c.ext, got, c.want)
		}
	}
}

// ===== stripXMLTags =====

func TestStripXMLTags_SimpleTags(t *testing.T) {
	input := `<w:p>Hello <w:b>World</w:b></w:p>`
	got := stripXMLTags(input)
	want := "Hello World"
	if got != want {
		t.Errorf("stripXMLTags(%q) = %q, want %q", input, got, want)
	}
}

func TestStripXMLTags_PreservesContent(t *testing.T) {
	input := `<w:p>第一段</w:p><w:p>第二段</w:p>`
	got := stripXMLTags(input)
	// Should contain both texts
	if !contains(got, "第一段") || !contains(got, "第二段") {
		t.Errorf("stripXMLTags should preserve text content, got: %q", got)
	}
}

func TestStripXMLTags_NoTags(t *testing.T) {
	input := "plain text no tags"
	got := stripXMLTags(input)
	if got != input {
		t.Errorf("stripXMLTags on plain text = %q, want %q", got, input)
	}
}

func TestStripXMLTags_EmptyString(t *testing.T) {
	got := stripXMLTags("")
	if got != "" {
		t.Errorf("stripXMLTags('') = %q, want empty", got)
	}
}

func TestStripXMLTags_OnlyTags(t *testing.T) {
	got := stripXMLTags("<root></root>")
	if got != "" {
		t.Errorf("stripXMLTags with only tags = %q, want empty", got)
	}
}

// ===== chunkText =====

func TestChunkText_Empty(t *testing.T) {
	result := chunkText("", 500, 100)
	if result != nil {
		t.Errorf("chunkText('') should return nil, got %v", result)
	}
}

func TestChunkText_WhitespaceOnly(t *testing.T) {
	result := chunkText("   \n  \t  ", 500, 100)
	if result != nil {
		t.Errorf("chunkText whitespace-only should return nil, got %v", result)
	}
}

func TestChunkText_ShortText(t *testing.T) {
	text := "This is a short paragraph."
	chunks := chunkText(text, 500, 100)
	if len(chunks) != 1 {
		t.Fatalf("Short text should produce 1 chunk, got %d", len(chunks))
	}
	if chunks[0] != text {
		t.Errorf("Single chunk content mismatch: got %q, want %q", chunks[0], text)
	}
}

func TestChunkText_MultipleParagraphs(t *testing.T) {
	// targetSize must be >= 200 (enforced by chunkText); use 200 with enough text
	lines := []string{
		"AABBCCDDEEFFGGHHIIJJKKLLMMNNOOPPQQRRSSTTUUVVWWXXYYZZ", // 52 chars each
		"AABBCCDDEEFFGGHHIIJJKKLLMMNNOOPPQQRRSSTTUUVVWWXXYYZZ",
		"AABBCCDDEEFFGGHHIIJJKKLLMMNNOOPPQQRRSSTTUUVVWWXXYYZZ",
		"AABBCCDDEEFFGGHHIIJJKKLLMMNNOOPPQQRRSSTTUUVVWWXXYYZZ",
		"AABBCCDDEEFFGGHHIIJJKKLLMMNNOOPPQQRRSSTTUUVVWWXXYYZZ",
		"AABBCCDDEEFFGGHHIIJJKKLLMMNNOOPPQQRRSSTTUUVVWWXXYYZZ",
	}
	text := joinLines(lines)
	chunks := chunkText(text, 200, 40)
	if len(chunks) < 2 {
		t.Errorf("Long text should produce multiple chunks, got %d", len(chunks))
	}
}

func TestChunkText_DefaultTargetSize(t *testing.T) {
	// targetSize < 200 should default to 500
	text := stringsRepeat("a", 250)
	chunks := chunkText(text, 100, 10)
	if len(chunks) != 1 {
		t.Errorf("With default targetSize 500, 250-char text should be 1 chunk, got %d", len(chunks))
	}
}

func TestChunkText_Overlap(t *testing.T) {
	// targetSize must be >= 200; use 200 with enough paragraphs to trigger chunking
	lines := make([]string, 0, 12)
	for i := 0; i < 12; i++ {
		lines = append(lines, stringsRepeat("x", 40))
	}
	text := joinLines(lines)
	chunks := chunkText(text, 200, 40)
	if len(chunks) < 2 {
		t.Fatalf("Expected 2+ chunks, got %d", len(chunks))
	}
	// All chunks should be non-empty
	for i, c := range chunks {
		if c == "" {
			t.Errorf("Chunk %d is empty", i)
		}
	}
}

// ===== extractTextFromFile =====

func TestExtractTextFromFile_TxtFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.txt")
	content := "Hello TradeMind!\nLine 2 content."
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	got, err := extractTextFromFile(path)
	if err != nil {
		t.Fatalf("extractTextFromFile(txt) error: %v", err)
	}
	if got != content {
		t.Errorf("Got %q, want %q", got, content)
	}
}

func TestExtractTextFromFile_MdFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "readme.md")
	content := "# Title\n\nSome **bold** text."
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	got, err := extractTextFromFile(path)
	if err != nil {
		t.Fatalf("extractTextFromFile(md) error: %v", err)
	}
	if got != content {
		t.Errorf("Got %q, want %q", got, content)
	}
}

func TestExtractTextFromFile_UnsupportedFormat(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "doc.pdf")
	if err := os.WriteFile(path, []byte("%PDF-1.4"), 0644); err != nil {
		t.Fatal(err)
	}
	_, err := extractTextFromFile(path)
	if err == nil {
		t.Error("extractTextFromFile on .pdf should return error")
	}
}

func TestExtractTextFromFile_NonExistent(t *testing.T) {
	_, err := extractTextFromFile("/nonexistent/path/file.txt")
	if err == nil {
		t.Error("extractTextFromFile on nonexistent file should return error")
	}
}

// ===== extractDocxText =====

func TestExtractDocxText_ValidDocx(t *testing.T) {
	// Create a minimal valid .docx (zip with word/document.xml)
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)

	docContent := `<?xml version="1.0" encoding="UTF-8"?>
<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
  <w:body>
    <w:p><w:r><w:t>TradeMind AI Knowledge Base</w:t></w:r></w:p>
    <w:p><w:r><w:t>Second paragraph here.</w:t></w:r></w:p>
  </w:body>
</w:document>`

	f, err := w.Create("word/document.xml")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.Write([]byte(docContent)); err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}

	got, err := extractDocxText(buf.Bytes())
	if err != nil {
		t.Fatalf("extractDocxText error: %v", err)
	}
	if !contains(got, "TradeMind AI Knowledge Base") {
		t.Errorf("Should contain document text, got: %q", got)
	}
	if !contains(got, "Second paragraph here.") {
		t.Errorf("Should contain second paragraph, got: %q", got)
	}
}

func TestExtractDocxText_NoDocumentXml(t *testing.T) {
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	f, _ := w.Create("other.txt")
	f.Write([]byte("not the document"))
	w.Close()

	_, err := extractDocxText(buf.Bytes())
	if err == nil {
		t.Error("extractDocxText without word/document.xml should return error")
	}
}

func TestExtractDocxText_InvalidZip(t *testing.T) {
	_, err := extractDocxText([]byte("not a zip file at all"))
	if err == nil {
		t.Error("extractDocxText on invalid zip should return error")
	}
}

// ===== helpers =====

func contains(s, substr string) bool {
	return bytes.Contains([]byte(s), []byte(substr))
}

func joinLines(lines []string) string {
	var buf bytes.Buffer
	for i, l := range lines {
		if i > 0 {
			buf.WriteByte('\n')
		}
		buf.WriteString(l)
	}
	return buf.String()
}

func stringsRepeat(s string, n int) string {
	var buf bytes.Buffer
	for i := 0; i < n; i++ {
		buf.WriteString(s)
	}
	return buf.String()
}
