# TradeMind AI — 构建与打包指南

## 前置条件

| 工具 | 版本 | 用途 |
|---|---|---|
| Go | 1.25+ | 后端编译（纯 Go，无 CGO 依赖） |
| Node.js | 20+ | 前端构建（Vite + React） |
| NSIS | 3.x | Windows 安装包打包（仅 Windows 上需要） |

> **关键**: 本项目使用 `glebarez/sqlite`（纯 Go SQLite 驱动），无需 CGO。
> 因此可以**从 macOS/Linux 直接交叉编译 Windows .exe**，不需要 MinGW 或 CGO 工具链。

## 快速构建

```bash
# 1. 设置 Go 代理（国内必须）
go env -w GOPROXY=https://goproxy.cn,direct

# 2. 构建当前平台（自动构建前端 → embed → 编译）
make build

# 3. 运行
./trademind
# → http://localhost:7789
```

## 交叉编译

### Windows .exe（从 macOS/Linux）

```bash
make build-win
# → build/windows/trademind.exe (约 40MB，单文件含前端)
```

技术原理：`GOOS=windows GOARCH=amd64 go build` — 因为 `glebarez/sqlite` 是纯 Go，
SQLite 引擎编译进二进制本身，不需要 Windows 上的 CGO/MinGW。

### macOS (Intel)

```bash
make build-mac
# → build/macos/trademind
```

### macOS (Apple Silicon)

```bash
make build-mac-arm
# → build/macos/trademind-arm64
```

### Linux

```bash
make build-linux
# → build/linux/trademind
```

## 打包安装包

### Windows NSIS 安装包

**前提**: 先在 macOS 上交叉编译 `make build-win`，然后把 `build/windows/` 目录拷贝到 Windows。

```powershell
# 安装 NSIS: https://nsis.sourceforge.io/Download
# 右键 installer.nsi → "Compile NSIS Script"
# 或命令行:
makensis build/windows/installer.nsi
# → build/windows/TradeMind-Setup-v1.0.0.exe
```

NSIS 安装包功能：
- 安装到 `C:\Program Files\TradeMind\`
- 开始菜单 + 桌面快捷方式
- 开机自启动（注册表 Run 键）
- 完整卸载程序（含数据清理选项）
- 中文 + 英文双语界面

### macOS .app

```bash
make package-mac
# → build/macos/TradeMind.app/
# 双击即可运行
```

## 测试

```bash
# 全部单元测试
make test

# 快速测试（跳过 bcrypt 等耗时项）
make test-short

# 覆盖率报告
make test-coverage
```

当前测试覆盖：
- `internal/pkg/crypto` — AES-256-GCM 加解密、bcrypt 密码
- `internal/service/document_parser` — 文档解析、文本分块、XML 标签剥离
- `internal/service/agent_service` — AI JSON 输出清洗
- `internal/service/daily_report_service` — 飞书 webhook 签名
- `internal/repository/knowledge_repo` — 余弦相似度、向量解析
- `internal/config` — 配置默认值与环境变量覆盖

## 版本管理

版本号通过 `make VERSION=x.y.z` 注入，默认 `1.0.0`：

```bash
make build VERSION=1.2.0
```

## 发布清单

发布前检查：
- [ ] `make check` 编译通过
- [ ] `make test` 全部通过
- [ ] `make build-win` Windows 交叉编译成功
- [ ] Windows 上 NSIS 打包成功
- [ ] 启动后 `curl http://localhost:7789/health` 返回正常
- [ ] 默认管理员 `admin/admin123` 可登录
- [ ] 首次启动向导 4 步骤完整走通
