# TradeMind AI

> 企业级私有化 AI 外贸智能操作系统 — 装在老板电脑里的 AI 外贸大脑

## 📋 项目概述

- **部署模式：** 企业私有化（单机安装）
- **技术路线：** Go 单体应用 + SQLite + Chrome 插件 + AI Agent
- **覆盖场景：** 传统外贸（B2B）+ 跨境电商（B2C），共享底座 + 两套场景模块包
- **数据私有：** 100% 本地存储，AI 调用直连厂商（用户自带 Key），零云端依赖

## 🏗️ 技术栈

| 层 | 技术 |
|---|---|
| 后端 | Go 1.22+ / Gin / GORM |
| 数据库 | SQLite（modernc.org/sqlite，纯 Go 无 CGO） |
| 向量库 | sqlite-vec（RAG 知识库） |
| 前端 | React 18 + Ant Design 5（go:embed 进 Go 二进制） |
| 桌面 | systray（系统托盘常驻） |
| 打包 | NSIS（Windows）/ .dmg（Mac） |

## 📁 目录结构

```
trademind/
├── cmd/trademind/          # 程序入口
├── internal/
│   ├── config/             # 配置加载
│   ├── database/           # DB 初始化 + Migration + Seed
│   ├── models/             # GORM Model（纯数据结构）
│   ├── repository/         # 数据访问层
│   ├── service/            # 业务逻辑层
│   ├── handler/            # HTTP 处理器（Gin）
│   ├── middleware/         # 中间件（JWT/CORS/日志/限流）
│   ├── router/             # 路由注册
│   ├── modules/            # 场景模块包
│   │   ├── common/         # 共享底座（两个场景都跑）
│   │   ├── b2b/            # 外贸 Pack
│   │   └── b2c/            # 跨境 Pack
│   ├── agent/              # Agent 体系
│   ├── ai/                 # AI 网关（多模型路由）
│   ├── tools/              # Agent 工具
│   └── pkg/                # 工具函数
│       ├── response/       # 统一响应
│       ├── logger/         # zap 日志
│       ├── crypto/         # AES 加密
│       └── errors/         # 自定义错误
├── web/                    # React 前端源码
├── extension/              # Chrome 插件 v3.0
├── build/                  # 打包脚本
└── docs/                   # 用户手册
```

## 📚 核心文档

开发前必读（飞书知识库）：
1. **开发方案 V2** — Go 单体私有化全量方案
2. **开发规范 V1.0** — 强制规范（违反即 bug）
3. **架构设计** — 双场景模块化 + 统一数据模型 + Agent 调度

## 🚀 开发

```bash
# 开发模式（前后端分离）
go run cmd/trademind/main.go       # 后端 :7789
cd web && npm run dev              # 前端 :5173

# 生产构建（前端嵌入二进制）
cd web && npm run build            # 产物在 web/dist/
go build -o trademind cmd/trademind/main.go
```

## 📦 贡献规范

- 严格遵守 [开发规范 V1.0](docs/STANDARDS.md)
- 四层架构：Handler → Service → Repository → Model（禁止跨层）
- 所有业务表必须软删除（`gorm.DeletedAt`）
- Commit 格式：`<type>(<scope>): <subject>`
- 一个功能一个分支，一个分支一个 PR

---

**License:** 商业产品，不开源
