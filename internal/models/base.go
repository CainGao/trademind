// Package models 定义所有 GORM Model（纯数据结构，无业务逻辑）。
//
// 所有业务表必须内嵌 BaseModel，自动获得 id/created_at/updated_at/deleted_at/created_by 五要素。
// 详见开发规范 V1.0 §1.1。
package models

import (
	"time"

	"gorm.io/gorm"
)

// BaseModel 是所有业务表的基类，按开发规范 V1.0 §1.2 提供表五要素：
//   - ID         主键（不用 UUID，SQLite 性能考虑）
//   - CreatedAt  创建时间（带索引）
//   - UpdatedAt  更新时间
//   - DeletedAt  软删除（带索引，禁止物理删除）
//   - CreatedBy  创建人（审计追踪，子表按需内嵌 CreatedByMixin）
//
// 使用方式：
//
//	type Product struct {
//	    BaseModel
//	    Name string
//	}
type BaseModel struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	CreatedAt time.Time      `gorm:"index" json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`
}

// CreatedByMixin 用于需要审计追踪的表（大多数业务表都应该用）。
// 内嵌位置：BaseModel 之后，业务字段之前。
type CreatedByMixin struct {
	CreatedBy uint `gorm:"index" json:"created_by"` // 创建人 user_id
}
