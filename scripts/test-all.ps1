# MeteorX local test entry: backend checks + frontend lint/type/test/build,
# optionally browser E2E (requires backend on :8081, MySQL and Redis).
#
# Usage:
#   powershell -File scripts/test-all.ps1
#   powershell -File scripts/test-all.ps1 -WithE2E
#   powershell -File scripts/test-all.ps1 -SkipFrontend
#
# NOTE: keep this file ASCII-only so it parses correctly under
# Windows PowerShell 5.1 (which misreads UTF-8 without BOM).
param(
    [switch]$WithE2E,
    [switch]$SkipFrontend
)
$ErrorActionPreference = "Stop"

$root = Join-Path $PSScriptRoot ".."
Push-Location $root
try {
    Write-Host "==> [1/3] backend: vet + build + test (scripts/test.ps1)"
    & (Join-Path $PSScriptRoot "test.ps1")
    if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }

    if (-not $SkipFrontend) {
        Write-Host "==> [2/3] frontend: lint + type-check + coverage + build"
        Push-Location (Join-Path $root "web-admin")
        try {
            npm run lint
            if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
            npx vue-tsc --noEmit
            if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
            npm run test:coverage
            if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
            npm run build
            if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
        } finally {
            Pop-Location
        }
    }

    if ($WithE2E) {
        Write-Host "==> [3/3] e2e: Playwright (real Chrome + MySQL + Redis)"
        $health = & curl.exe -s -o NUL -w "%{http_code}" http://127.0.0.1:8081/health
        if ($health -ne "200") {
            Write-Warning "backend is not ready on http://127.0.0.1:8081 (health=$health)."
            Write-Warning "start it first: go run ./cmd/server"
            exit 1
        }
        if (-not (Test-Path (Join-Path $root "web-admin/dist"))) {
            Write-Warning "web-admin/dist is missing; run npm run build in web-admin first."
            exit 1
        }
        if (-not (Test-Path (Join-Path $root "e2e/node_modules"))) {
            Write-Warning "e2e/node_modules is missing; run npm ci in e2e first."
            exit 1
        }
        Push-Location (Join-Path $root "e2e")
        try {
            if (-not $env:E2E_CHANNEL) { $env:E2E_CHANNEL = "chrome" }
            npx playwright test
            if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
        } finally {
            Pop-Location
        }
    } else {
        Write-Host "==> [3/3] e2e skipped (use -WithE2E)"
    }

    Write-Host "==> all checks passed"
} finally {
    Pop-Location
}
