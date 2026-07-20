// Package models — RAG 知识库、文件管理、系统配置、Agent Prompt、API 调用日志、审计日志。
package models

import "time"

// KnowledgeChunk RAG 知识库切片。文件解析后切片 + Embedding 存入此表。
type KnowledgeChunk struct {
	BaseModel

	FileID     uint   `gorm:"not null;index" json:"file_id"`
	SourceFile string `gorm:"not null;size:300;index" json:"source_file"` // 冗余存文件名，删除文件后仍可追溯
	ChunkIndex int    `gorm:"not null" json:"chunk_index"`
	Content    string `gorm:"not null;type:text" json:"content"`
	Embedding  string `gorm:"type:text" json:"-"`                      // JSON array（纯 Go 余弦相似度）
	Metadata   string `gorm:"type:text" json:"metadata"`               // JSON: {section, char_start, ...}
}

// KnowledgeFile 知识库文件（Week 8 RAG）。记录上传/粘贴的文档元数据 + 解析状态。
// TableName 指定表名 knowledge_files，区别于通用 files（为未来其他文件用途留扩展空间）。
type KnowledgeFile struct {
	BaseModel
	CreatedByMixin

	Title      string     `gorm:"not null;size:300" json:"title"`        // 显示标题（粘贴文本时可自定义）
	Filename   string     `gorm:"not null;size:300" json:"filename"`     // 原始文件名
	FileType   string     `gorm:"size:20;index" json:"file_type"`        // txt|md|csv|docx|paste
	FileSize   int64      `json:"file_size"`                             // 字节数（粘贴文本按字符数估算）
	StoredPath string     `gorm:"type:text" json:"stored_path"`          // runtime/files/ 下的相对路径；粘贴文本为空
	ChunkCount int        `gorm:"default:0" json:"chunk_count"`          // 生成的切片数
	Status     FileStatus `gorm:"size:20;default:processing;index" json:"status"` // processing|ready|failed
	ParseError string     `gorm:"type:text" json:"parse_error,omitempty"`
}

// TableName 指定表名。
func (KnowledgeFile) TableName() string { return "knowledge_files" }

// File 上传文件管理（通用，保留兼容）。
type File struct {
	BaseModel
	CreatedByMixin

	Filename   string `gorm:"not null;size:300" json:"filename"`
	FileType   string `gorm:"size:20;index" json:"file_type"` // pdf|docx|xlsx|txt|image
	FileSize   int64  `json:"file_size"`
	StoredPath string `gorm:"not null;type:text" json:"stored_path"` // runtime/files/ 下的相对路径
	Parsed     *bool  `gorm:"default:false" json:"parsed"`
}

// Setting 系统配置。敏感字段（AI Key 等）AES 加密存储。
type Setting struct {
	Key       string    `gorm:"primaryKey;size:100" json:"key"`
	Value     string    `gorm:"type:text" json:"value,omitempty"` // 敏感字段 AES 加密
	Encrypted *bool     `gorm:"default:false" json:"encrypted"`
	UpdatedAt time.Time `json:"updated_at"`
}

// TableName Setting 不用 BaseModel（配置表不需要审计字段）。
func (Setting) TableName() string { return "settings" }

// AgentPrompt Agent Prompt 版本管理。同一类型可有多版本，is_active 控制当前生效。
type AgentPrompt struct {
	BaseModel

	AgentType       AgentType `gorm:"not null;type:text;uniqueIndex:uk_agent_prompts_type_version,priority:1" json:"agent_type"`
	Version         string    `gorm:"not null;size:20;uniqueIndex:uk_agent_prompts_type_version,priority:2" json:"version"`
	SystemPrompt    string    `gorm:"not null;type:text" json:"system_prompt"`
	ToolDefinitions string    `gorm:"type:text" json:"tool_definitions"` // JSON: ["query_products","calc_profit"]
	IsActive        *bool     `gorm:"default:false;index:idx_agent_prompts_active" json:"is_active"`
}

// APICallLog API 调用日志，用于 AI 成本追踪和告警。
type APICallLog struct {
	BaseModel

	AgentType    *AgentType `gorm:"type:text;index" json:"agent_type,omitempty"`
	Model        string     `gorm:"size:50;index" json:"model"`
	TokensInput  *int       `json:"tokens_input,omitempty"`
	TokensOutput *int       `json:"tokens_output,omitempty"`
	CostUSD      *float64   `gorm:"type:decimal(6,4)" json:"cost_usd,omitempty"`
	CalledAt     time.Time  `gorm:"index" json:"called_at"`
}

// AuditLog 审计日志。敏感操作（创建/更新/删除/登录/导出）必记。
type AuditLog struct {
	BaseModel

	UserID     uint   `gorm:"not null;index" json:"user_id"`
	Action     string `gorm:"not null;type:text;index" json:"action"` // create|update|delete|login|export
	Resource   string `gorm:"not null;type:text" json:"resource"`     // product|supplier|customer
	ResourceID *uint  `gorm:"index" json:"resource_id,omitempty"`
	Detail     string `gorm:"type:text" json:"detail"`                 // JSON 变更内容
	IP         string `gorm:"size:50" json:"ip"`
}
