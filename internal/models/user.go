// Package models — 用户、角色、企业信息。
package models

import "time"

// Company 企业信息（单条记录，首次启动向导创建）。
type Company struct {
	BaseModel
	Name     string `gorm:"not null" json:"name"`
	Industry string `json:"industry"`
	Country  string `json:"country"`
	Contact  string `json:"contact"`
	Logo     string `json:"logo"` // base64 或 runtime/files/ 下的相对路径
}

// User 用户。共享底座，五角色适配 B2B/B2C 两个场景。
type User struct {
	BaseModel
	Username     string     `gorm:"uniqueIndex:uk_users_username;not null;size:50" json:"username"`
	PasswordHash string     `gorm:"not null;size:60" json:"-"` // bcrypt hash，禁止返回前端
	Nickname     string     `gorm:"size:50" json:"nickname"`
	Department   string     `gorm:"size:50" json:"department"`
	Role         UserRole   `gorm:"type:text;default:'staff';index" json:"role"`
	Status       UserStatus `gorm:"type:text;default:'active'" json:"status"`
	Avatar       string     `json:"avatar"`
	LastLoginAt  *time.Time `gorm:"index" json:"last_login_at,omitempty"`
}

// TableName 指定表名（规范 V1.0 §1.3: 复数下划线）。
func (User) TableName() string { return "users" }
