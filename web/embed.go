// Package web 内嵌前端构建产物。
//
// 生产模式：web/dist/ 通过 go:embed 嵌入二进制。
// 开发模式：前端跑 5173 独立服务，不走这里。
//
// 嵌入文件放在本包而非 main 包，是因为 go:embed 不支持 ../ 上溯。
// 本包位于 web/ 目录下，与 dist/ 同级。
package web

import "embed"

// DistFS 前端构建产物的 embed.FS。
//
//go:embed all:dist
var DistFS embed.FS
