// Package service — 审计日志业务逻辑。
//
// 审计日志从 Week 1 起持续写入（登录/登录失败等敏感操作，规范 §6.7），
// 本服务提供管理员查看能力（列表 + 筛选 + 用户名映射）。
package service

import (
	"github.com/CainGao/trademind/internal/models"
	"github.com/CainGao/trademind/internal/repository"
)

// AuditLogItem 审计日志视图（附带用户名，前端无需二次映射）。
type AuditLogItem struct {
	models.AuditLog
	Username string `json:"username"`
}

// AuditListResult 审计日志列表结果。
type AuditListResult struct {
	Items []AuditLogItem `json:"items"`
	Total int64          `json:"total"`
	Page  int            `json:"page"`
	Size  int            `json:"size"`
}

// AuditService 审计日志服务。
type AuditService struct {
	auditRepo *repository.AuditLogRepo
	userRepo  *repository.UserRepo
}

func NewAuditService(auditRepo *repository.AuditLogRepo, userRepo *repository.UserRepo) *AuditService {
	return &AuditService{auditRepo: auditRepo, userRepo: userRepo}
}

// List 查询审计日志（筛选 + 分页），并把 user_id 映射为用户名。
// 用户已软删时用户名仍可查出（审计记录不因用户删除而失去可读性）。
func (s *AuditService) List(f repository.AuditLogFilter, page, size int) (*AuditListResult, error) {
	logs, total, err := s.auditRepo.ListAll(f, page, size)
	if err != nil {
		return nil, err
	}

	// 收集本页出现的 user_id，一次性建映射（避免 N+1 查询）
	idSet := map[uint]bool{}
	for _, l := range logs {
		idSet[l.UserID] = true
	}
	names := map[uint]string{}
	for uid := range idSet {
		if u, err := s.userRepo.GetByID(uid); err == nil {
			names[uid] = u.Username
		}
	}

	items := make([]AuditLogItem, 0, len(logs))
	for _, l := range logs {
		name := names[l.UserID]
		if name == "" {
			name = "（已删除用户）"
		}
		items = append(items, AuditLogItem{AuditLog: l, Username: name})
	}

	outPage, outSize := page, size
	if outPage < 1 {
		outPage = 1
	}
	if outSize < 1 {
		outSize = 20
	}
	return &AuditListResult{Items: items, Total: total, Page: outPage, Size: outSize}, nil
}
