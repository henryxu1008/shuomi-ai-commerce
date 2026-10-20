# Makefile for Task Processor / ListingKit
.PHONY: all build-all clean test test-fast test-all test-coverage help \
	build-current-application build-shein-login-worker \
	run-current-application lint fmt code-health-audit code-health-verify

# 版本信息
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "v1.0.0")
BUILD_TIME := $(shell date -u '+%Y-%m-%d_%H:%M:%S')
LDFLAGS := -X main.appVersion=$(VERSION) -X main.buildTime=$(BUILD_TIME)

# 输出目录
BIN_DIR := bin

# 当前维护的测试包
FAST_TEST_PACKAGES := ./cmd/current-application ./cmd/shein-login-worker ./internal/app/httpapi ./internal/listingadmin ./internal/promptmgmt ./internal/listingsubscription
BOUNDARY_TEST_PACKAGES := ./tests/...
ALL_TEST_PACKAGES := ./cmd/... ./internal/... ./tests/... ./tools/... ./hack/debug/...

# 帮助信息
help:
	@echo "Task Processor / ListingKit 构建工具"
	@echo ""
	@echo "当前正式入口以 docs/development/repository-structure.md 和 tests/repository_structure_test.go 为准。"
	@echo ""
	@echo "使用方法:"
	@echo "  make build-all                 - 构建当前受维护的服务入口"
	@echo "  make build-current-application - 构建Current Application"
	@echo "  make build-shein-login-worker   - 构建独立 SHEIN 登录 worker"
	@echo "  make clean                     - 清理构建文件"
	@echo "  make test                      - 运行常用快速测试"
	@echo "  make test-fast                 - 运行常用快速测试"
	@echo "  make test-all                  - 运行全部 Go 测试"
	@echo "  make code-health-audit        - 运行只读死代码/重复代码审计"
	@echo "  make code-health-verify       - 校验审计基线与前端未分类发现"
	@echo ""

all: build-all

# 构建所有当前受维护服务
build-all: build-current-application build-shein-login-worker
	@echo "✅ 所有当前受维护服务构建完成"

# Current Application
build-current-application:
	@echo "🔨 构建Current Application..."
	@mkdir -p $(BIN_DIR)
	CGO_ENABLED=0 GOOS=linux go build -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/current-application ./cmd/current-application
	@echo "✅ Current Application 构建完成: $(BIN_DIR)/current-application"

# Standalone SHEIN login worker
build-shein-login-worker:
	@mkdir -p $(BIN_DIR)
	CGO_ENABLED=0 GOOS=linux go build -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/shein-login-worker ./cmd/shein-login-worker

# 清理构建文件
clean:
	@echo "🧹 清理构建文件..."
	rm -rf $(BIN_DIR)
	@echo "✅ 清理完成"

# 运行测试
test: test-fast

# 运行常用快速测试
test-fast:
	@echo "🧪 运行常用快速测试..."
	go test -v $(FAST_TEST_PACKAGES)
	go test -v $(BOUNDARY_TEST_PACKAGES)

# 运行全部 Go 测试
test-all:
	@echo "🧪 运行全部 Go 测试..."
	go test -v $(ALL_TEST_PACKAGES)

# 运行测试（带覆盖率）
test-coverage:
	@echo "🧪 运行测试（带覆盖率）..."
	go test -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "✅ 覆盖率报告已生成: coverage.html"

# 代码检查
lint:
	@echo "🔍 运行代码检查..."
	golangci-lint run

# 格式化代码
fmt:
	@echo "✨ 格式化代码..."
	go fmt ./...

# Reproducible read-only code-health audit. Reports stay under .local/.
code-health-audit:
	pwsh -File scripts/code-health-audit.ps1 -Mode All

# CI/local ratchet: compile, run pinned analyzers, and reject new Knip findings.
code-health-verify:
	pwsh -File scripts/code-health-audit.ps1 -Mode Verify

# 本地运行Current Application
run-current-application:
	@test -n "$(CURRENT_APPLICATION_CONFIG)" || (echo "Set CURRENT_APPLICATION_CONFIG to an absolute private manifest path"; exit 1)
	@echo "🚀 启动Current Application..."
	go run ./cmd/current-application --config="$(CURRENT_APPLICATION_CONFIG)"
