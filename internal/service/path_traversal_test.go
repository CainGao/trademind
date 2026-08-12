package service

import (
	"strings"
	"testing"
)

// ===== sanitizeFilename 测试（gotcha #67：知识库上传路径穿越防护）=====

func TestSanitizeFilename_NormalName(t *testing.T) {
	got := sanitizeFilename("产品手册.txt")
	if got != "产品手册.txt" {
		t.Errorf("正常文件名被修改: got %q", got)
	}
}

func TestSanitizeFilename_RelativePathTraversal(t *testing.T) {
	// ../../etc/passwd.txt → basename 去掉目录部分
	got := sanitizeFilename("../../etc/passwd.txt")
	if strings.Contains(got, "..") || strings.Contains(got, "/") {
		t.Errorf("路径穿越未拦截: got %q", got)
	}
	t.Logf("净化结果: %q", got)
}

func TestSanitizeFilename_AbsolutePath(t *testing.T) {
	got := sanitizeFilename("/etc/shadow.txt")
	if strings.HasPrefix(got, "/") {
		t.Errorf("绝对路径未拦截: got %q", got)
	}
	t.Logf("净化结果: %q", got)
}

func TestSanitizeFilename_BackslashPath(t *testing.T) {
	got := sanitizeFilename("..\\..\\windows\\system32\\evil.txt")
	// 关键检查：结果不含路径分隔符（\ 或 /），即无法穿越目录
	if strings.Contains(got, "\\") || strings.Contains(got, "/") {
		t.Errorf("Windows 路径穿越未拦截: got %q", got)
	}
	t.Logf("净化结果: %q", got)
}

func TestSanitizeFilename_NullBytes(t *testing.T) {
	got := sanitizeFilename("test\x00.txt")
	if strings.Contains(got, "\x00") {
		t.Errorf("空字节未移除: got %q", got)
	}
}

func TestSanitizeFilename_EmptyString(t *testing.T) {
	got := sanitizeFilename("")
	if got == "" {
		t.Error("空字符串应被替换为 unnamed")
	}
}

func TestSanitizeFilename_DotOnly(t *testing.T) {
	got := sanitizeFilename(".")
	if got == "." {
		t.Error("点号应被替换")
	}
}

func TestSanitizeFilename_DoubleDot(t *testing.T) {
	got := sanitizeFilename("..")
	if got == ".." {
		t.Error("双点号应被替换")
	}
}

func TestSanitizeFilename_LongName(t *testing.T) {
	longName := strings.Repeat("a", 300) + ".txt"
	got := sanitizeFilename(longName)
	if len(got) > 200 {
		t.Errorf("超长文件名未截断: len=%d", len(got))
	}
	if !strings.HasSuffix(got, ".txt") {
		t.Errorf("截断后扩展名丢失: got %q", got)
	}
}

// ===== validateCollectInput 测试（gotcha #67：采集输入长度上限）=====

func TestValidateCollectInput_NormalInput(t *testing.T) {
	in := CollectProductInput{
		Name:        "蓝牙耳机",
		Description: "高品质无线蓝牙耳机",
		SourceURL:   "https://detail.1688.com/offer/123.html",
		Category:    "电子产品",
		SourceID:    "offer_123",
	}
	if err := validateCollectInput(&in); err != nil {
		t.Errorf("正常输入不应报错: %v", err)
	}
}

func TestValidateCollectInput_OversizedName(t *testing.T) {
	in := CollectProductInput{
		Name: strings.Repeat("x", maxCollectNameLen+1),
	}
	err := validateCollectInput(&in)
	if err == nil {
		t.Error("超长商品名称应被拒绝")
	}
	if !strings.Contains(err.Error(), "商品名称过长") {
		t.Errorf("错误信息不匹配: %v", err)
	}
}

func TestValidateCollectInput_OversizedDescription(t *testing.T) {
	in := CollectProductInput{
		Name:        "ok",
		Description: strings.Repeat("x", maxCollectDescLen+1),
	}
	err := validateCollectInput(&in)
	if err == nil {
		t.Error("超长描述应被拒绝")
	}
}

func TestValidateCollectInput_OversizedSourceURL(t *testing.T) {
	in := CollectProductInput{
		Name:      "ok",
		SourceURL: strings.Repeat("x", maxCollectURLLen+1),
	}
	err := validateCollectInput(&in)
	if err == nil {
		t.Error("超长 URL 应被拒绝")
	}
}

func TestValidateCollectInput_TooManyImages(t *testing.T) {
	urls := make([]string, maxCollectImageURLCount+1)
	for i := range urls {
		urls[i] = "https://example.com/img.jpg"
	}
	in := CollectProductInput{
		Name:      "ok",
		ImageURLs: urls,
	}
	err := validateCollectInput(&in)
	if err == nil {
		t.Error("过多图片应被拒绝")
	}
}

func TestValidateCollectInput_OversizedSupplierName(t *testing.T) {
	in := CollectProductInput{
		Name: "ok",
		Supplier: SupplierInfo{
			Name: strings.Repeat("x", maxCollectSupplierNameLen+1),
		},
	}
	err := validateCollectInput(&in)
	if err == nil {
		t.Error("超长供应商名称应被拒绝")
	}
}

func TestValidateCollectInput_BoundaryValues(t *testing.T) {
	// 恰好等于上限应通过
	in := CollectProductInput{
		Name:        strings.Repeat("x", maxCollectNameLen),
		Description: strings.Repeat("x", maxCollectDescLen),
		SourceURL:   strings.Repeat("x", maxCollectURLLen),
		Category:    strings.Repeat("x", maxCollectCategoryLen),
		SourceID:    strings.Repeat("x", maxCollectSourceIDLen),
	}
	if err := validateCollectInput(&in); err != nil {
		t.Errorf("恰好等于上限应通过: %v", err)
	}
}
