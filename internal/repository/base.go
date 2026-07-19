// Package repository 数据访问层（规范 V1.0 §2.2）。
//
// 职责：只做 CRUD + 查询，不写业务逻辑。
// 业务逻辑在 service 层；数据结构在 models 层。
package repository

import "gorm.io/gorm"

// BaseRepo 所有 Repo 内嵌，获得通用 DB 句柄。
type BaseRepo struct {
	DB *gorm.DB
}

// Transaction 包装 GORM 事务（规范 V1.0 §1.8: 多表操作必须用事务）。
func (r *BaseRepo) Transaction(fn func(tx *gorm.DB) error) error {
	return r.DB.Transaction(fn)
}
