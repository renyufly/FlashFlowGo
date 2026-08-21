$ErrorActionPreference = 'Stop'
$root = Split-Path -Parent $PSScriptRoot
$go = if (Test-Path (Join-Path $root '.tools/go/bin/go.exe')) { Join-Path $root '.tools/go/bin/go.exe' } else { 'go' }

# The Go race detector needs a working 64-bit C toolchain on Windows. Prefer it
# when available; otherwise use the pinned Linux Go image without global setup.
& $go env CGO_ENABLED | Out-Null
$gcc = Get-Command gcc -ErrorAction SilentlyContinue
if ($gcc) {
    $machine = (& gcc -dumpmachine 2>$null)
    if ($machine -match 'x86_64|amd64') {
        & $go test -race ./...
        exit $LASTEXITCODE
    }
}

Write-Host 'No compatible local 64-bit C compiler; running race tests in Docker.'
docker run --rm --volume "${root}:/workspace" --workdir /workspace `
    --env GOCACHE=/tmp/go-build --env GOMODCACHE=/tmp/go-mod `
    golang:1.26.6-bookworm go test -race ./...
exit $LASTEXITCODE
