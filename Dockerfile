# ============================================================
# Stage 1: Build
# ============================================================
FROM golang:1.25-alpine AS builder

WORKDIR /app

# 安装构建依赖
RUN apk add --no-cache git ca-certificates tzdata

# 利用 Docker 缓存层，先复制依赖文件
COPY go.mod go.sum ./
RUN go mod download

# 复制源代码
COPY . .

# 构建静态链接二进制（去除调试信息、压缩体积）
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -ldflags="-w -s" -a -installsuffix cgo -o meteorx ./cmd/server

# ============================================================
# Stage 2: Runtime
# ============================================================
FROM alpine:3.20

# 元数据标签
LABEL org.opencontainers.image.title="MeteorX"
LABEL org.opencontainers.image.description="MeteorX 多租户 SaaS 管理系统后端"
LABEL org.opencontainers.image.version="1.0.0"
LABEL org.opencontainers.image.source="https://github.com/meteorx/meteorx"
LABEL org.opencontainers.image.licenses="MIT"

# 安装运行时依赖
RUN apk add --no-cache ca-certificates tzdata curl

# 设置时区
ENV TZ=Asia/Shanghai

# 创建非 root 用户
RUN addgroup -g 1001 appgroup && \
    adduser -S -u 1001 -G appgroup appuser

# 创建工作目录
WORKDIR /app

# 从构建阶段复制产物
COPY --from=builder /app/meteorx .
COPY --from=builder /app/internal/config/config.yaml ./config/

# 创建必要目录并设置权限
RUN mkdir -p logs uploads data && \
    chown -R appuser:appgroup /app

# 切换到非 root 用户
USER appuser

# 暴露端口
EXPOSE 8080

# 健康检查
HEALTHCHECK --interval=30s --timeout=3s --start-period=10s --retries=3 \
    CMD curl -sf http://localhost:8080/health || exit 1

# 启动应用
ENTRYPOINT ["./meteorx"]