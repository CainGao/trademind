# TradeMind AI — 构建系统
#
# 用法:
#   make              → 构建 macOS 本地二进制（含前端 embed）
#   make test         → 运行全部单元测试
#   make frontend     → 构建前端到 web/dist/
#   make build-win    → 交叉编译 Windows .exe（纯 Go，无需 CGO）
#   make build-mac    → 构建 macOS 二进制
#   make package-win  → 构建 Windows 二进制（NSIS 打包需 Windows 上运行 makensis）
#   make clean        → 清理构建产物
#
# 技术要点:
#   - glebarez/sqlite 是纯 Go 驱动，GOOS=windows 交叉编译无需 CGO
#   - 前端通过 go:embed 嵌入二进制，最终交付单文件
#   - 版本号通过 ldflags 注入（可覆盖 config.go 默认值）

BINARY_NAME  = trademind
BINARY_WIN   = trademind.exe
VERSION     ?= 1.0.0
BUILD_DIR    = build
LDFLAGS      = -X 'github.com/CainGao/trademind/internal/config.AppVersion=$(VERSION)'

# Go 编译参数
GO           = go
GOFLAGS      = -ldflags "$(LDFLAGS)" -trimpath

# 默认目标
.DEFAULT_GOAL := build

# ===== 开发 =====

.PHONY: dev
dev: ## 启动开发模式（前端热更新 + Go 后端）
	cd web && npm run dev

# ===== 测试 =====

.PHONY: test
test: ## 运行全部单元测试
	$(GO) test ./internal/... -v -count=1 -timeout 60s

.PHONY: test-short
test-short: ## 运行快速测试（跳过耗时的 bcrypt）
	$(GO) test ./internal/... -short -count=1

.PHONY: test-coverage
test-coverage: ## 生成测试覆盖率报告
	$(GO) test ./internal/... -coverprofile=coverage.out -count=1
	$(GO) tool cover -func=coverage.out | tail -1

# ===== 前端 =====

.PHONY: frontend
frontend: ## 构建前端到 web/dist/
	cd web && npm run build

.PHONY: frontend-install
frontend-install: ## 安装前端依赖
	cd web && npm install

# ===== 构建 =====

.PHONY: build
build: frontend ## 构建 macOS 本地二进制（含前端 embed）
	$(GO) build $(GOFLAGS) -o $(BINARY_NAME) ./cmd/trademind/
	@echo "✅ 构建完成: ./$(BINARY_NAME)"
	@ls -lh $(BINARY_NAME)

.PHONY: build-mac
build-mac: frontend ## 构建 macOS 二进制到 build/macos/
	mkdir -p $(BUILD_DIR)/macos
	GOOS=darwin GOARCH=amd64 $(GO) build $(GOFLAGS) -o $(BUILD_DIR)/macos/$(BINARY_NAME) ./cmd/trademind/
	@echo "✅ macOS 构建完成: $(BUILD_DIR)/macos/$(BINARY_NAME)"

.PHONY: build-mac-arm
build-mac-arm: frontend ## 构建 macOS ARM64 (Apple Silicon) 二进制
	mkdir -p $(BUILD_DIR)/macos
	GOOS=darwin GOARCH=arm64 $(GO) build $(GOFLAGS) -o $(BUILD_DIR)/macos/$(BINARY_NAME)-arm64 ./cmd/trademind/
	@echo "✅ macOS ARM64 构建完成: $(BUILD_DIR)/macos/$(BINARY_NAME)-arm64"

.PHONY: build-win
build-win: frontend ## 交叉编译 Windows .exe（纯 Go SQLite，无需 CGO）
	mkdir -p $(BUILD_DIR)/windows
	GOOS=windows GOARCH=amd64 $(GO) build $(GOFLAGS) -o $(BUILD_DIR)/windows/$(BINARY_WIN) ./cmd/trademind/
	@echo "✅ Windows 构建完成: $(BUILD_DIR)/windows/$(BINARY_WIN)"
	@ls -lh $(BUILD_DIR)/windows/$(BINARY_WIN)

.PHONY: build-linux
build-linux: frontend ## 交叉编译 Linux 二进制
	mkdir -p $(BUILD_DIR)/linux
	GOOS=linux GOARCH=amd64 $(GO) build $(GOFLAGS) -o $(BUILD_DIR)/linux/$(BINARY_NAME) ./cmd/trademind/
	@echo "✅ Linux 构建完成: $(BUILD_DIR)/linux/$(BINARY_NAME)"

# ===== 打包 =====

.PHONY: package-win
package-win: build-win ## 准备 Windows NSIS 打包（需在 Windows 上运行 makensis）
	@echo "📦 Windows 二进制已就绪: $(BUILD_DIR)/windows/$(BINARY_WIN)"
	@echo "   在 Windows 上安装 NSIS 后运行:"
	@echo "   makensis $(BUILD_DIR)/windows/installer.nsi"
	@echo "   生成: $(BUILD_DIR)/windows/TradeMind-Setup-v$(VERSION).exe"

.PHONY: package-mac
package-mac: build-mac ## 打包 macOS .app 目录
	mkdir -p $(BUILD_DIR)/macos/TradeMind.app/Contents/MacOS
	mkdir -p $(BUILD_DIR)/macos/TradeMind.app/Contents/Resources
	cp $(BUILD_DIR)/macos/$(BINARY_NAME) $(BUILD_DIR)/macos/TradeMind.app/Contents/MacOS/
	cp build/macos/Info.plist $(BUILD_DIR)/macos/TradeMind.app/Contents/Info.plist 2>/dev/null || true
	@echo "✅ macOS .app 打包完成: $(BUILD_DIR)/macos/TradeMind.app"

# ===== 清理 =====

.PHONY: clean
clean: ## 清理构建产物
	rm -f $(BINARY_NAME)
	rm -rf $(BUILD_DIR)/windows/$(BINARY_WIN) $(BUILD_DIR)/macos/$(BINARY_NAME)
	rm -f coverage.out
	@echo "✅ 清理完成"

.PHONY: clean-all
clean-all: clean ## 清理所有产物（含前端 dist）
	rm -rf web/dist
	@echo "✅ 全部清理完成"

# ===== 辅助 =====

.PHONY: run
run: ## 直接运行（需先 make build）
	./$(BINARY_NAME)

.PHONY: deps
deps: ## 更新 Go 依赖
	$(GO) mod tidy
	$(GO) mod download

.PHONY: check
check: ## 检查编译（不产出二进制）
	$(GO) build ./... 2>&1
	@echo "✅ 编译检查通过"

.PHONY: help
help: ## 显示帮助
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-18s\033[0m %s\n", $$1, $$2}'
