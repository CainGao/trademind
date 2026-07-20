// Package service — AI 网关：多模型路由 + AES Key 解密 + 统一 OpenAI 兼容协议。
//
// 设计原则（架构文档 §3.3 AI 网关）：
//   - 用户首启时配置的 AI Key 用 AES-256-GCM 加密存到 settings 表
//   - 运行时按需解密，绝不打印/返回 Key 明文
//   - 统一走 OpenAI Chat Completions 协议（DeepSeek/Qwen/Tongyi 原生兼容；
//     Anthropic 有独立协议但社区有兼容层；本期先用 OpenAI 协议覆盖 3 家）
//   - 零云端中转：Go 服务直连 AI 厂商，老板的 Key 不经过我们
//
// 路由策略：
//   1. 用户指定 provider → 用该 provider
//   2. 未指定 → 用 settings 中的 default_model
//   3. 默认 provider 的 Key 没配 → 按优先级 deepseek > openai > qwen > anthropic 找第一个有 Key 的
package service

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/CainGao/trademind/internal/pkg/crypto"
	"github.com/CainGao/trademind/internal/repository"
)

// AIProvider 厂商标识。
type AIProvider string

const (
	ProviderDeepSeek  AIProvider = "deepseek"
	ProviderOpenAI    AIProvider = "openai"
	ProviderQwen      AIProvider = "qwen"
	ProviderAnthropic AIProvider = "anthropic"
)

// 各厂商的 OpenAI 兼容端点。
var providerEndpoints = map[AIProvider]string{
	ProviderDeepSeek:  "https://api.deepseek.com/v1/chat/completions",
	ProviderOpenAI:    "https://api.openai.com/v1/chat/completions",
	ProviderQwen:      "https://dashscope.aliyuncs.com/compatible-mode/v1/chat/completions",
	ProviderAnthropic: "https://api.anthropic.com/v1/chat/completions", // 兼容层
}

// 各厂商默认模型名。
var providerDefaultModel = map[AIProvider]string{
	ProviderDeepSeek:  "deepseek-chat",
	ProviderOpenAI:    "gpt-4o-mini",
	ProviderQwen:      "qwen-plus",
	ProviderAnthropic: "claude-3-5-haiku-20241022",
}

// settings 表中的 Key 名（与 setup_service 保持一致）。
const (
	settingKeyDeepSeekKey   = "ai_key_deepseek"
	settingKeyOpenAIKey     = "ai_key_openai"
	settingKeyQwenKey       = "ai_key_qwen"
	settingKeyAnthropicKey  = "ai_key_anthropic"
	settingKeyDefaultModel  = "ai_default_model"
)

// AIService AI 网关业务。
type AIService struct {
	settingRepo *repository.SettingRepo
	encryptor   *crypto.Encryptor
	httpClient  *http.Client
}

func NewAIService(sr *repository.SettingRepo, enc *crypto.Encryptor) *AIService {
	return &AIService{
		settingRepo: sr,
		encryptor:   enc,
		httpClient: &http.Client{
			Timeout: 60 * time.Second, // AI 调用可能较慢
		},
	}
}

// ChatMessage OpenAI 协议消息格式。
type ChatMessage struct {
	Role    string `json:"role"`    // system|user|assistant
	Content string `json:"content"`
}

// ChatRequest 内部 chat 请求。
type ChatRequest struct {
	Provider AIProvider   `json:"provider,omitempty"` // 可选，未指定走默认
	Model    string       `json:"model,omitempty"`    // 可选，未指定走厂商默认
	Messages []ChatMessage `json:"messages"`
	Temperature float64   `json:"temperature,omitempty"` // 0-2，默认 0.7
	MaxTokens  int        `json:"max_tokens,omitempty"`  // 默认不限制
}

// ChatResponse 统一响应。
type ChatResponse struct {
	Provider AIProvider `json:"provider"`
	Model    string     `json:"model"`
	Content  string     `json:"content"`
	Usage    *TokenUsage `json:"usage,omitempty"`
}

// TokenUsage token 用量。
type TokenUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// Chat 执行 AI 对话。
// 流程：解密 Key → 选 provider → 构造 OpenAI 请求 → 转发 → 解析响应。
func (s *AIService) Chat(req ChatRequest) (*ChatResponse, error) {
	if len(req.Messages) == 0 {
		return nil, errors.New("messages 不能为空")
	}

	// 1. 解析 provider 和 apiKey
	provider, apiKey, err := s.resolveProvider(req.Provider)
	if err != nil {
		return nil, err
	}

	// 2. 决定模型名
	model := req.Model
	if model == "" {
		model = providerDefaultModel[provider]
	}

	// 3. 构造 OpenAI 兼容请求
	temp := req.Temperature
	if temp == 0 {
		temp = 0.7
	}
	body := map[string]interface{}{
		"model":       model,
		"messages":    req.Messages,
		"temperature": temp,
		"stream":      false,
	}
	if req.MaxTokens > 0 {
		body["max_tokens"] = req.MaxTokens
	}
	bodyBytes, _ := json.Marshal(body)

	// 4. 发起请求
	url := providerEndpoints[provider]
	httpReq, _ := http.NewRequest("POST", url, bytes.NewReader(bodyBytes))
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+apiKey)

	resp, err := s.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("调用 %s 失败: %w", provider, err)
	}
	defer resp.Body.Close()

	respBytes, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		// 截取错误信息前 500 字符，避免日志过长
		errMsg := string(respBytes)
		if len(errMsg) > 500 {
			errMsg = errMsg[:500]
		}
		return nil, fmt.Errorf("%s 返回 %d: %s", provider, resp.StatusCode, errMsg)
	}

	// 5. 解析 OpenAI 协议响应
	var raw struct {
		Choices []struct {
			Message struct {
				Role    string `json:"role"`
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
		Usage struct {
			PromptTokens     int `json:"prompt_tokens"`
			CompletionTokens int `json:"completion_tokens"`
			TotalTokens      int `json:"total_tokens"`
		} `json:"usage"`
		Model string `json:"model"`
	}
	if err := json.Unmarshal(respBytes, &raw); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}
	if len(raw.Choices) == 0 {
		return nil, errors.New("AI 返回空响应（无 choices）")
	}

	return &ChatResponse{
		Provider: provider,
		Model:    raw.Model,
		Content:  raw.Choices[0].Message.Content,
		Usage: &TokenUsage{
			PromptTokens:     raw.Usage.PromptTokens,
			CompletionTokens: raw.Usage.CompletionTokens,
			TotalTokens:      raw.Usage.TotalTokens,
		},
	}, nil
}

// resolveProvider 解析 provider + 解密对应 Key。
// 优先级：用户指定 > default_model > 第一个有 Key 的。
func (s *AIService) resolveProvider(specified AIProvider) (AIProvider, string, error) {
	// 读 default_model
	defaultModel := s.getSetting(settingKeyDefaultModel)

	// 候选顺序
	candidates := []AIProvider{specified}
	if specified == "" {
		if defaultModel != "" {
			candidates = []AIProvider{AIProvider(defaultModel)}
		}
		// 默认 provider 没配 Key 时按优先级 fallback
		candidates = append(candidates, ProviderDeepSeek, ProviderOpenAI, ProviderQwen, ProviderAnthropic)
	}

	// 去重
	seen := map[AIProvider]bool{}
	for _, p := range candidates {
		if p == "" || seen[p] {
			continue
		}
		seen[p] = true
		key := s.getDecryptedKey(p)
		if key != "" {
			return p, key, nil
		}
	}
	return "", "", errors.New("未找到可用的 AI Key，请在系统设置中配置")
}

// getDecryptedKey 从 settings 读取并解密指定 provider 的 Key。
func (s *AIService) getDecryptedKey(p AIProvider) string {
	keyName := map[AIProvider]string{
		ProviderDeepSeek:  settingKeyDeepSeekKey,
		ProviderOpenAI:    settingKeyOpenAIKey,
		ProviderQwen:      settingKeyQwenKey,
		ProviderAnthropic: settingKeyAnthropicKey,
	}[p]
	encrypted := s.getSetting(keyName)
	if encrypted == "" {
		return ""
	}
	plain, err := s.encryptor.Decrypt(encrypted)
	if err != nil {
		return ""
	}
	return plain
}

// getSetting 读 settings 表，失败返回空字符串。
func (s *AIService) getSetting(key string) string {
	setting, err := s.settingRepo.Get(key)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(setting.Value)
}

// ConfiguredProviders 返回当前已配置的 provider 列表（前端显示用，不返回 Key）。
type ProviderInfo struct {
	Name      AIProvider `json:"name"`
	Configured bool      `json:"configured"`
	IsDefault  bool      `json:"is_default"`
}

// ConfiguredProviders 已配置的 provider 列表。
func (s *AIService) ConfiguredProviders() []ProviderInfo {
	defaultModel := s.getSetting(settingKeyDefaultModel)
	result := []ProviderInfo{}
	for _, p := range []AIProvider{ProviderDeepSeek, ProviderOpenAI, ProviderQwen, ProviderAnthropic} {
		result = append(result, ProviderInfo{
			Name:       p,
			Configured: s.getDecryptedKey(p) != "",
			IsDefault:  string(p) == defaultModel,
		})
	}
	return result
}

// ============================================================================
// Embedding（Week 8 RAG）
//
// OpenAI 兼容的 /v1/embeddings 协议。OpenAI 和 Qwen(DashScope 兼容模式)均原生支持。
// DeepSeek / Anthropic 无 Embedding API —— 按优先级 openai > qwen 找第一个有 Key 的。
// ============================================================================

// 支持嵌入的厂商及其端点。
var providerEmbeddingEndpoint = map[AIProvider]string{
	ProviderOpenAI: "https://api.openai.com/v1/embeddings",
	ProviderQwen:   "https://dashscope.aliyuncs.com/compatible-mode/v1/embeddings",
}

// 各厂商默认嵌入模型。
var providerEmbeddingModel = map[AIProvider]string{
	ProviderOpenAI: "text-embedding-3-small", // 1536 维，便宜
	ProviderQwen:   "text-embedding-v2",      // 1536 维
}

// EmbeddingResponse 嵌入结果。
type EmbeddingResponse struct {
	Provider AIProvider `json:"provider"`
	Model    string     `json:"model"`
	Vectors  [][]float64 `json:"vectors"` // 与输入文本一一对应
	Dims     int        `json:"dims"`     // 向量维度
}

// Embed 批量生成文本的嵌入向量。
// 优先级：指定 provider > openai > qwen（DeepSeek/Anthropic 无 embedding API）。
func (s *AIService) Embed(texts []string, specified AIProvider) (*EmbeddingResponse, error) {
	if len(texts) == 0 {
		return nil, errors.New("嵌入文本不能为空")
	}

	// 选支持 embedding 的 provider
	provider, apiKey, model, err := s.resolveEmbeddingProvider(specified)
	if err != nil {
		return nil, err
	}

	// 构造 OpenAI 兼容请求
	body := map[string]interface{}{
		"model": model,
		"input": texts,
	}
	bodyBytes, _ := json.Marshal(body)

	url := providerEmbeddingEndpoint[provider]
	httpReq, _ := http.NewRequest("POST", url, bytes.NewReader(bodyBytes))
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+apiKey)

	resp, err := s.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("调用 %s 嵌入失败: %w", provider, err)
	}
	defer resp.Body.Close()

	respBytes, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		errMsg := string(respBytes)
		if len(errMsg) > 500 {
			errMsg = errMsg[:500]
		}
		return nil, fmt.Errorf("%s 嵌入返回 %d: %s", provider, resp.StatusCode, errMsg)
	}

	// 解析 OpenAI 协议嵌入响应
	var raw struct {
		Data []struct {
			Embedding []float64 `json:"embedding"`
			Index     int       `json:"index"`
		} `json:"data"`
		Model string `json:"model"`
	}
	if err := json.Unmarshal(respBytes, &raw); err != nil {
		return nil, fmt.Errorf("解析嵌入响应失败: %w", err)
	}
	if len(raw.Data) == 0 {
		return nil, errors.New("嵌入返回空数据")
	}

	// 按 index 排序（OpenAI 规范保证顺序，但保险起见）
	vectors := make([][]float64, len(raw.Data))
	for _, d := range raw.Data {
		if d.Index >= 0 && d.Index < len(vectors) {
			vectors[d.Index] = d.Embedding
		}
	}

	return &EmbeddingResponse{
		Provider: provider,
		Model:    raw.Model,
		Vectors:  vectors,
		Dims:     len(vectors[0]),
	}, nil
}

// resolveEmbeddingProvider 选支持 embedding 的 provider（DeepSeek/Anthropic 跳过）。
func (s *AIService) resolveEmbeddingProvider(specified AIProvider) (AIProvider, string, string, error) {
	// 候选顺序：指定 > openai > qwen
	candidates := []AIProvider{}
	if specified != "" {
		candidates = append(candidates, specified)
	}
	candidates = append(candidates, ProviderOpenAI, ProviderQwen)

	seen := map[AIProvider]bool{}
	for _, p := range candidates {
		if seen[p] || providerEmbeddingEndpoint[p] == "" {
			continue
		}
		seen[p] = true
		key := s.getDecryptedKey(p)
		if key != "" {
			return p, key, providerEmbeddingModel[p], nil
		}
	}
	return "", "", "", errors.New("无可用的嵌入模型（需配置 OpenAI 或 Qwen 的 AI Key；DeepSeek/Anthropic 不支持嵌入）")
}
