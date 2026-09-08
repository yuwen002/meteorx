# MeteorX backend one-shot checks (vet + build + tests + coverage).
# Windows PowerShell local entry; Linux/macOS and CI use scripts/test.sh
# (CI additionally runs tests with -race).
# NOTE: keep this file ASCII-only so it parses correctly under
# Windows PowerShell 5.1 (which misreads UTF-8 without BOM).
$ErrorActionPreference = "Stop"

Push-Location (Join-Path $PSScriptRoot "..")
try {
    # Backend package scope. web-admin (frontend) is isolated from the Go
    # module via its nested go.mod, so it is intentionally not included here.
    $pkgs = @("./cmd/...", "./internal/...", "./pkg/...")

    Write-Host "==> go vet $($pkgs -join ' ')"
    go vet @pkgs
    if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }

    Write-Host "==> go build $($pkgs -join ' ')"
    go build @pkgs
    if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }

    Write-Host "==> go test -count=1 (writes ./coverage.out)"
    # NOTE: quote the flag value - under Windows PowerShell an unquoted
    # "-coverprofile=coverage.out" is split at the dot into ".out".
    go test -count=1 "-coverprofile=coverage.out" @pkgs
    if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }

    Write-Host "==> backend checks passed"
}
finally {
    Pop-Location
}
