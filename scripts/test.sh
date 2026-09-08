#!/usr/bin/env bash
# MeteorX 后端一键校验（vet + build + race 测试 + 覆盖率）。
# CI（.github/workflows/ci-cd.yml 的 backend job）与 Linux/macOS 本地共用同一入口，
# 保证命令范围与语义一致、不漂移。Windows 本地请使用 scripts/test.ps1。
set -euo pipefail
cd "$(dirname "$0")/.."

# 后端包范围。web-admin（前端）已通过其目录下的嵌套 go.mod 与后端 Go 模块隔离，
# 无需也不应被纳入后端校验。
PKGS="./cmd/... ./internal/... ./pkg/..."

echo "==> go vet $PKGS"
go vet $PKGS

echo "==> go build $PKGS"
go build $PKGS

echo "==> go test -race -count=1（生成 ./coverage.out）"
go test -race -count=1 -coverprofile=coverage.out $PKGS

echo "==> backend checks passed"
