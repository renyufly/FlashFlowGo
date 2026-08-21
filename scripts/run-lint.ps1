$ErrorActionPreference = 'Stop'
$root = Split-Path -Parent $PSScriptRoot
$localGoBin = Join-Path $root '.tools/go/bin'
if (Test-Path (Join-Path $localGoBin 'go.exe')) {
    $env:Path = "$localGoBin;$env:Path"
}
$env:GOCACHE = Join-Path $root '.cache/go-build'
$env:GOMODCACHE = Join-Path $root '.cache/go-mod'
$env:GOLANGCI_LINT_CACHE = Join-Path $root '.cache/golangci-lint'
$lint = if (Test-Path (Join-Path $root '.tools/bin/golangci-lint.exe')) {
    Join-Path $root '.tools/bin/golangci-lint.exe'
} else {
    'golangci-lint'
}
& $lint run ./...
exit $LASTEXITCODE
