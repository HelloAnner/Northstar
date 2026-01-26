# Northstar Makefile
# 支持 Windows/macOS/Linux 三平台构建

# 版本信息
VERSION := 1.0.0
BUILD_TIME := $(shell date +%Y-%m-%d_%H:%M:%S)
GIT_COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")

# 目录
DIST_DIR := dist
WEB_DIR := web
STATIC_DIR := internal/server/dist
E2E_DIR := tests/e2e
REPORTS_DIR := tests/e2e-result

# Go 编译参数
LDFLAGS := -ldflags "-s -w -X main.Version=$(VERSION) -X main.BuildTime=$(BUILD_TIME) -X main.GitCommit=$(GIT_COMMIT)"

# 测试服务器端口
TEST_PORT := 18080

# 默认目标
.PHONY: all
all: build

# 帮助信息
.PHONY: help
help:
	@echo "Northstar 构建脚本"
	@echo ""
	@echo "用法:"
	@echo "  make build          - 构建当前平台可执行文件"
	@echo "  make build-all      - 构建全部三个平台可执行文件"
	@echo "  make test           - 运行全部测试（单元测试 + E2E测试）"
	@echo "  make test-unit      - 仅运行单元测试"
	@echo "  make test-e2e       - 仅运行E2E测试"
	@echo "  make start          - 启动可执行文件"
	@echo "  make dev            - 开发模式启动（热更新）"
	@echo "  make clean          - 清理构建产物"
	@echo "  make deps           - 安装依赖"
	@echo ""

# 安装依赖
.PHONY: deps
deps:
	@echo ">>> 安装 Go 依赖..."
	go mod tidy
	go mod download
	@echo ">>> 安装前端依赖..."
	cd $(WEB_DIR) && npm install

# 构建前端
.PHONY: build-web
build-web:
	@echo ">>> 构建前端..."
	cd $(WEB_DIR) && npm run build
	@echo ">>> 前端构建完成"

# 确保静态资源目录存在
.PHONY: ensure-static
ensure-static:
	@mkdir -p $(STATIC_DIR)

# 构建当前平台
.PHONY: build
build: build-web ensure-static
	@echo ">>> 构建当前平台可执行文件..."
	@mkdir -p $(DIST_DIR)
	go build $(LDFLAGS) -o $(DIST_DIR)/northstar ./cmd/northstar
	@cp config.toml.example $(DIST_DIR)/config.toml.example
	@echo ">>> 构建完成: $(DIST_DIR)/northstar"

# 构建 Windows (amd64)
.PHONY: build-windows
build-windows: build-web ensure-static
	@echo ">>> 构建 Windows (amd64)..."
	@mkdir -p $(DIST_DIR)
	GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o $(DIST_DIR)/northstar-windows-amd64.exe ./cmd/northstar
	@echo ">>> Windows 构建完成"

# 构建 macOS (amd64)
.PHONY: build-darwin-amd64
build-darwin-amd64: build-web ensure-static
	@echo ">>> 构建 macOS (amd64)..."
	@mkdir -p $(DIST_DIR)
	GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o $(DIST_DIR)/northstar-darwin-amd64 ./cmd/northstar
	@echo ">>> macOS amd64 构建完成"

# 构建 macOS (arm64)
.PHONY: build-darwin-arm64
build-darwin-arm64: build-web ensure-static
	@echo ">>> 构建 macOS (arm64)..."
	@mkdir -p $(DIST_DIR)
	GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o $(DIST_DIR)/northstar-darwin-arm64 ./cmd/northstar
	@echo ">>> macOS arm64 构建完成"

# 构建 Linux (amd64)
.PHONY: build-linux
build-linux: build-web ensure-static
	@echo ">>> 构建 Linux (amd64)..."
	@mkdir -p $(DIST_DIR)
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o $(DIST_DIR)/northstar-linux-amd64 ./cmd/northstar
	@echo ">>> Linux 构建完成"

# 构建全部平台
.PHONY: build-all
build-all: build-web ensure-static
	@echo ">>> 构建全部平台..."
	@mkdir -p $(DIST_DIR)

	@echo ">>> [1/4] Windows (amd64)..."
	GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o $(DIST_DIR)/northstar-windows-amd64.exe ./cmd/northstar

	@echo ">>> [2/4] macOS (amd64)..."
	GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o $(DIST_DIR)/northstar-darwin-amd64 ./cmd/northstar

	@echo ">>> [3/4] macOS (arm64)..."
	GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o $(DIST_DIR)/northstar-darwin-arm64 ./cmd/northstar

	@echo ">>> [4/4] Linux (amd64)..."
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o $(DIST_DIR)/northstar-linux-amd64 ./cmd/northstar

	@cp config.toml.example $(DIST_DIR)/
	@echo ""
	@echo ">>> 全部平台构建完成!"
	@ls -la $(DIST_DIR)/

# 开发模式启动（模拟可执行文件）
.PHONY: start
start: build
	@echo ">>> 启动 Northstar..."
	$(DIST_DIR)/northstar

# 开发模式启动（代码热更新）
.PHONY: dev
dev:
	@echo ">>> 开发模式启动..."
	@echo ">>> 启动前端开发服务器..."
	cd $(WEB_DIR) && npm run dev &
	@sleep 2
	@echo ">>> 启动后端服务器..."
	go run ./cmd/northstar -dev

# 仅启动后端（开发模式）
.PHONY: start-backend
start-backend:
	@echo ">>> 启动后端服务器 (开发模式)..."
	go run ./cmd/northstar -dev

# 仅启动前端开发服务器
.PHONY: start-web
start-web:
	@echo ">>> 启动前端开发服务器..."
	cd $(WEB_DIR) && npm run dev

# ==================== 测试相关 ====================

# 清理测试报告目录
.PHONY: clean-test-reports
clean-test-reports:
	@echo ">>> 清理旧的测试结果..."
	@rm -rf $(REPORTS_DIR)

# 准备测试报告目录（仅创建，不删除）
.PHONY: prepare-test-reports
prepare-test-reports:
	@mkdir -p $(REPORTS_DIR)

# 安装 E2E 测试依赖
.PHONY: setup-e2e
setup-e2e:
	@echo ">>> 设置 E2E 测试环境..."
	@if [ ! -d "$(E2E_DIR)/venv" ]; then \
		echo ">>> 创建 Python 虚拟环境..."; \
		python3 -m venv $(E2E_DIR)/venv; \
	fi
	@echo ">>> 安装 Python 依赖..."
	@. $(E2E_DIR)/venv/bin/activate && pip install -q -r $(E2E_DIR)/requirements.txt
	@echo ">>> E2E 测试环境准备完成"

# 生成测试数据
.PHONY: generate-test-data
generate-test-data: setup-e2e
	@echo ">>> 生成测试数据 (300条企业数据)..."
	@rm -rf $(E2E_DIR)/fixtures
	@. $(E2E_DIR)/venv/bin/activate && python $(E2E_DIR)/test_data_generator.py
	@echo ">>> 测试数据生成完成"

# 仅运行单元测试（独立运行，包含清理）
.PHONY: test-unit
test-unit: clean-test-reports prepare-test-reports test-unit-only
	@echo ">>> 覆盖率报告: $(REPORTS_DIR)/coverage.html"

# 仅运行 E2E 测试（独立运行，包含清理和数据生成）
.PHONY: test-e2e
test-e2e: build clean-test-reports prepare-test-reports generate-test-data test-e2e-only
	@echo ">>> 测试报告: $(REPORTS_DIR)/report.html"

# 运行全部测试（单元测试 + E2E 测试）
.PHONY: test
test: build clean-test-reports prepare-test-reports generate-test-data test-unit-only test-e2e-only
	@echo ""
	@echo "=========================================="
	@echo "  全部测试完成"
	@echo "=========================================="
	@echo ""
	@echo "测试报告目录: $(REPORTS_DIR)/"
	@echo "  - 测试报告:       $(REPORTS_DIR)/report.html"
	@echo "  - 单元测试覆盖率: $(REPORTS_DIR)/coverage.html"
	@echo "  - 测试数据文件:   $(REPORTS_DIR)/test_companies_300.xlsx"
	@echo ""

# 内部目标：仅运行单元测试（不含依赖）
.PHONY: test-unit-only
test-unit-only:
	@echo ""
	@echo "=========================================="
	@echo "  运行 Go 单元测试"
	@echo "=========================================="
	@echo ""
	go test -v -cover -coverprofile=$(REPORTS_DIR)/coverage.out ./...
	@echo ""
	@echo ">>> 生成覆盖率报告..."
	go tool cover -func=$(REPORTS_DIR)/coverage.out
	go tool cover -html=$(REPORTS_DIR)/coverage.out -o $(REPORTS_DIR)/coverage.html
	@echo ""
	@echo ">>> 单元测试完成"

# 内部目标：仅运行 E2E 测试（不含依赖）
.PHONY: test-e2e-only
test-e2e-only:
	@echo ""
	@echo "=========================================="
	@echo "  运行 E2E 端到端测试"
	@echo "=========================================="
	@echo ""
	@echo ">>> 清理可能残留的测试服务器..."
	@pkill -f "northstar -port $(TEST_PORT)" 2>/dev/null || true
	@sleep 1
	@echo ">>> 启动测试服务器 (端口: $(TEST_PORT))..."
	@$(DIST_DIR)/northstar -port $(TEST_PORT) > $(REPORTS_DIR)/server.log 2>&1 &
	@echo ">>> 等待服务器启动..."
	@sleep 3
	@echo ">>> 执行 E2E 测试用例..."
	@. $(E2E_DIR)/venv/bin/activate && python -m pytest $(E2E_DIR)/ \
		-v \
		--html=$(REPORTS_DIR)/report.html \
		--self-contained-html; \
	TEST_RESULT=$$?; \
	echo ">>> 停止测试服务器..."; \
	pkill -f "northstar -port $(TEST_PORT)" 2>/dev/null || true; \
	echo ">>> 复制测试数据文件..."; \
	cp -r $(E2E_DIR)/fixtures/* $(REPORTS_DIR)/ 2>/dev/null || true; \
	exit $$TEST_RESULT
	@echo ""
	@echo ">>> E2E 测试完成"

# 快速测试（仅单元测试，不启动服务器）
.PHONY: test-quick
test-quick:
	@echo ">>> 快速单元测试..."
	go test -v ./...

# 代码检查
.PHONY: lint
lint:
	@echo ">>> 运行代码检查..."
	go vet ./...
	@echo ">>> 代码检查完成"

# 格式化代码
.PHONY: fmt
fmt:
	@echo ">>> 格式化代码..."
	go fmt ./...
	cd $(WEB_DIR) && npm run format 2>/dev/null || true
	@echo ">>> 格式化完成"

# 清理
.PHONY: clean
clean:
	@echo ">>> 清理构建产物..."
	rm -rf $(DIST_DIR)
	rm -rf $(STATIC_DIR)
	rm -rf $(REPORTS_DIR)
	rm -f coverage.out coverage.html report.html
	rm -rf $(E2E_DIR)/fixtures $(E2E_DIR)/test_output
	cd $(WEB_DIR) && rm -rf node_modules dist
	@echo ">>> 清理完成"

# 清理（保留 node_modules 和 venv）
.PHONY: clean-build
clean-build:
	@echo ">>> 清理构建产物..."
	rm -rf $(DIST_DIR)
	rm -rf $(STATIC_DIR)
	rm -rf $(REPORTS_DIR)
	rm -f coverage.out coverage.html report.html
	@echo ">>> 清理完成"
