// Package service — 弱密码策略（gotcha #88：默认密码治理）。
//
// 背景：首启向导的 Complete() 虽然强制走 ChangePassword（flag 校验），
// 但不阻止「新密码 = 旧密码」或「新密码 = 常见弱密码」——管理员可以把
// 密码改回 admin123 后照样标记完成，之后系统再无任何检测（遗留观察
// 「管理员密码仍为默认 admin123」持续数周的根因）。
//
// 治理两层：
//  1. 写入口拦截 —— Register / ChangePassword 拒绝弱密码黑名单；
//  2. 登录检测 —— Login 成功后若 admin 仍用默认密码，响应带
//     must_change_password=true，前端顶部横幅持续提醒。

package service

import "strings"

// weakPasswords 常见弱密码黑名单（小写比较）。
// 私有化部署不做复杂度字典，只拦最常见的一档 + 本项目默认密码。
var weakPasswords = map[string]bool{
	"admin123":  true, // 本项目默认密码（最优先拦截）
	"123456":    true,
	"1234567":   true,
	"12345678":  true,
	"123456789": true,
	"password":  true,
	"abc123":    true,
	"qwerty":    true,
	"111111":    true,
	"888888":    true,
	"666666":    true,
	"000000":    true,
	"a123456":   true,
	"admin888":  true,
	"admin666":  true,
}

// isWeakPassword 密码是否命中弱密码黑名单（大小写不敏感）。
func isWeakPassword(pw string) bool {
	return weakPasswords[strings.ToLower(pw)]
}
