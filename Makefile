.PHONY: help build build-dev test test-coverage lint run docker-build docker-up docker-down docker-dev-up docker-dev-down migrate seed clean

# 默认目标
.DEFAULT_GOAL := help

# 变量
APP_NAME := meteorx
BUILD_DIR := ./build
CMD_PATH := ./cmd/server

# 帮助信息
help:
	@echo "MeteorX - Makefile Commands"
	@echo ""
	@echo "Usage:"
	@echo "  make <command>"
	@echo ""
	@echo "Commands:"
	@echo "  build              构建生产版本（Linux）"
	@echo "  build-all          构建所有平台版本"
	@echo "  build-dev          构建开发版本"
	@echo "  test               运行所有测试"
	@echo "  test-coverage      运行测试并生成覆盖率报告"
	@echo "  lint               运行代码质量检查"
	@echo "  run                运行开发服务器"
	@echo "  docker-build       构建 Docker 镜像"
	@echo "  docker-up          启动 Docker Compose（生产）"
	@echo "  docker-down        停止 Docker Compose（生产）"
	@echo "  docker-dev-up      启动 Docker Compose（开发）"
	@echo "  docker-dev-down    停止 Docker Compose（开发）"
	@echo "  migrate            运行数据库迁移"
	@echo "  seed               初始化种子数据"
	@echo "  clean              清理构建产物"

# 构建
build:
	@echo "Building production binary..."
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o $(BUILD_DIR)/$(APP_NAME) $(CMD_PATH)
	@echo "Build complete: $(BUILD_DIR)/$(APP_NAME)"

build-all:
	@echo "Building for all platforms..."
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o $(BUILD_DIR)/$(APP_NAME) $(CMD_PATH)
	CGO_ENABLED=0 GOOS=darwin go build -a -installsuffix cgo -o $(BUILD_DIR)/$(APP_NAME)-macos $(CMD_PATH)
	CGO_ENABLED=0 GOOS=windows go build -a -installsuffix cgo -o $(BUILD_DIR)/$(APP_NAME).exe $(CMD_PATH)
	@echo "Build complete for linux, darwin, windows"

build-dev:
	@echo "Building development binary..."
	@mkdir -p $(BUILD_DIR)
	go build -race -gcflags="all=-N -l" -o $(BUILD_DIR)/$(APP_NAME)-dev $(CMD_PATH)
	@echo "Development build complete: $(BUILD_DIR)/$(APP_NAME)-dev"

# 测试
test:
	@echo "Running tests..."
	go test -v -race ./...

test-coverage:
	@echo "Running tests with coverage..."
	go test -v -race -coverprofile=coverage.out -covermode=atomic ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# 代码质量
lint:
	@echo "Running linter..."
	golangci-lint run --timeout=5m

# 运行
run:
	@echo "Starting development server..."
	go run $(CMD_PATH)/main.go

# Docker
docker-build:
	@echo "Building Docker image..."
	docker build -t $(APP_NAME):latest .

docker-up:
	@echo "Starting Docker Compose (production)..."
	docker-compose up -d

docker-down:
	@echo "Stopping Docker Compose (production)..."
	docker-compose down

docker-dev-up:
	@echo "Starting Docker Compose (development)..."
	docker-compose -f docker-compose.dev.yml up -d

docker-dev-down:
	@echo "Stopping Docker Compose (development)..."
	docker-compose -f docker-compose.dev.yml down

# 数据库
migrate:
	@echo "Running database migrations..."
	@echo "Migrations are auto-executed on app start"

seed:
	@echo "Seeding database..."
	@echo "Seed data is auto-initialized on app start"

# 清理
clean:
	@echo "Cleaning build artifacts..."
	rm -rf $(BUILD_DIR)
	rm -f coverage.out coverage.html
	@echo "Clean complete"