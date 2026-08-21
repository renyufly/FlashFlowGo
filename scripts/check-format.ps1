$ErrorActionPreference = 'Stop'
$gofmt = if (Test-Path '.tools/go/bin/gofmt.exe') { '.tools/go/bin/gofmt.exe' } else { 'gofmt' }
$files = Get-ChildItem -Path 'cmd', 'internal' -Recurse -Filter '*.go' -File | ForEach-Object { $_.FullName }
$unformatted = if ($files) { & $gofmt -l $files }
if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
if ($unformatted) {
    Write-Error "gofmt is required for: $($unformatted -join ', ')"
}
