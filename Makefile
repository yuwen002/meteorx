.PHONY: help build build-all build-dev build-all-platforms test test-coverage lint vet security audit run
.PHONY: docker-build docker-up docker-down docker-dev-up docker-dev-down docker-prod-up docker-prod-down
.PHONY: frontend-install frontend-dev frontend-build frontend-lint frontend-test
.PHONY: migrate seed clean dev db-shell redis-shell

# 默认目标
.DEFAULT_GOAL := help

# 变量
APP_NAME := meteorx
BUILD_DIR := ./build
CMD_PATH := ./cmd/server
FRONTEND_DIR := ./web-admin

# 帮助信息
help:
	@echo "MeteorX - Makefile Commands"
	@echo ""
	@echo "Usage:  make <command>"
	@echo ""
	@echo "=== 构建 ==="
	@echo "  build               构建生产版本（Linux amd64, 带优化）"
	@echo "  build-all-platforms 构建多平台（linux/darwin/windows, amd64/arm64）"
	@echo "  build-dev           构建开发版本（带 race detector 和调试符号）"
	@echo ""
	@echo "=== 测试 ==="
	@echo "  test                运行所有测试"
	@echo "  test-coverage       运行测试并生成覆盖率报告"
	@echo "  vet                 运行 go vet 静态分析"
	@echo ""
	@echo "=== 代码质量 ==="
	@echo "  lint                运行 golangci-lint"
	@echo "  security            运行 govulncheck 安全扫描"
	@echo "  audit               运行 go mod tidy + vendor 审计"
	@echo ""
	@echo "=== 运行 ==="
	@echo "  run                 启动开发服务器（go run）"
	@echo "  dev                 启动开发环境（docker-compose.dev.yml）"
	@echo ""
	@echo "=== Docker ==="
	@echo "  docker-build        构建 Docker 镜像"
	@echo "  docker-up           启动 Docker Compose（生产）"
	@echo "  docker-down         停止 Docker Compose（生产）"
	@echo "  docker-prod-up      启动 Docker Compose（生产增强版）"
	@echo "  docker-prod-down    停止 Docker Compose（生产增强版）"
	@echo "  docker-dev-up       启动 Docker Compose（开发，含热重载）"
	@echo "  docker-dev-down     停止 Docker Compose（开发）"
	@echo ""
	@echo "=== 前端 ==="
	@echo "  frontend-install    安装前端依赖"
	@echo "  frontend-dev        启动前端开发服务器"
	@echo "  frontend-build      构建前端生产版本"
	@echo "  frontend-lint       运行前端代码检查"
	@echo "  frontend-test       运行前端测试"
	@echo ""
	@echo "=== 数据库 ==="
	@echo "  migrate             运行数据库迁移（应用启动时自动执行）"
	@echo "  seed                初始化种子数据（应用启动时自动执行）"
	@echo "  db-shell            连接到 MySQL 交互式 Shell"
	@echo "  redis-shell         连接到 Redis 交互式 CLI"
	@echo ""
	@echo "=== 工具 ==="
	@echo "  clean               清理构建产物"
	@echo "  tidy                运行 go mod tidy"

# ─── 构建 ───────────────────────────────────────────────────

build:
	@echo "Building production binary (linux amd64)..."
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-w -s" -o $(BUILD_DIR)/$(APP_NAME)-linux-amd64 $(CMD_PATH)
	@echo "Build complete: $(BUILD_DIR)/$(APP_NAME)-linux-amd64"

build-all-platforms:
	@echo "Building for all platforms..."
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-w -s" -o $(BUILD_DIR)/$(APP_NAME)-linux-amd64 $(CMD_PATH)
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -ldflags="-w -s" -o $(BUILD_DIR)/$(APP_NAME)-linux-arm64 $(CMD_PATH)
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -ldflags="-w -s" -o $(BUILD_DIR)/$(APP_NAME)-darwin-amd64 $(CMD_PATH)
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -ldflags="-w -s" -o $(BUILD_DIR)/$(APP_NAME)-windows-amd64.exe $(CMD_PATH)
	@echo "Build complete for linux/darwin/windows (amd64/arm64)"

build-dev:
	@echo "Building development binary..."
	@mkdir -p $(BUILD_DIR)
	go build -race -gcflags="all=-N -l" -o $(BUILD_DIR)/$(APP_NAME)-dev $(CMD_PATH)
	@echo "Development build complete: $(BUILD_DIR)/$(APP_NAME)-dev"

# ─── 测试 ───────────────────────────────────────────────────

test:
	@echo "Running tests..."
	go test -v -race ./...

test-coverage:
	@echo "Running tests with coverage..."
	go test -v -race -coverprofile=coverage.out -covermode=atomic ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

vet:
	@echo "Running go vet..."
	go vet ./...

# ─── 代码质量 ───────────────────────────────────────────────

lint:
	@echo "Running linter..."
	golangci-lint run --timeout=5m

security:
	@echo "Running govulncheck..."
	govulncheck ./...

audit:
	@echo "Running dependency audit..."
	go mod tidy -v
	go mod verify

# ─── 运行 ───────────────────────────────────────────────────

run:
	@echo "Starting development server..."
	go run $(CMD_PATH)/main.go

dev: docker-dev-up

# ─── Docker ─────────────────────────────────────────────────

docker-build:
	@echo "Building Docker image..."
	docker build -t $(APP_NAME):latest .

docker-up:
	@echo "Starting Docker Compose (production)..."
	docker-compose up -d

docker-down:
	@echo "Stopping Docker Compose (production)..."
	docker-compose down

docker-prod-up:
	@echo "Starting Docker Compose (production enhanced)..."
	docker-compose -f docker-compose.prod.yml up -d

docker-prod-down:
	@echo "Stopping Docker Compose (production enhanced)..."
	docker-compose -f docker-compose.prod.yml down

docker-dev-up:
	@echo "Starting Docker Compose (development)..."
	docker-compose -f docker-compose.dev.yml up -d

docker-dev-down:
	@echo "Stopping Docker Compose (development)..."
	docker-compose -f docker-compose.dev.yml down

# ─── 前端 ───────────────────────────────────────────────────

frontend-install:
	@echo "Installing frontend dependencies..."
	cd $(FRONTEND_DIR) && npm install

frontend-dev:
	@echo "Starting frontend dev server..."
	cd $(FRONTEND_DIR) && npm run dev

frontend-build:
	@echo "Building frontend..."
	cd $(FRONTEND_DIR) && npm run build

frontend-lint:
	@echo "Running frontend lint..."
	cd $(FRONTEND_DIR) && npm run lint

frontend-test:
	@echo "Running frontend tests..."
	cd $(FRONTEND_DIR) && npm run test:run

# ─── 数据库 ─────────────────────────────────────────────────

migrate:
	@echo "Migrations are auto-executed on app start"
	@echo "To run manually: go run $(CMD_PATH)/main.go --migrate-only"

seed:
	@echo "Seed data is auto-initialized on app start"
	@echo "To run manually: go run $(CMD_PATH)/main.go --seed-only"

db-shell:
	@echo "Connecting to MySQL..."
	docker exec -it meteorx-mysql mysql -uroot -p

redis-shell:
	@echo "Connecting to Redis..."
	docker exec -it meteorx-redis redis-cli -a

# ─── 工具 ───────────────────────────────────────────────────

clean:
	@echo "Cleaning build artifacts..."
	rm -rf $(BUILD_DIR)
	rm -f coverage.out coverage.html
	@echo "Clean complete"

tidy:
	@echo "Running go mod tidy..."
	go mod tidy -v