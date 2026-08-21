$ErrorActionPreference = 'Stop'
$root = Split-Path -Parent $PSScriptRoot
$tools = Join-Path $root '.tools'
$downloads = Join-Path $tools 'downloads'
$bin = Join-Path $tools 'bin'
New-Item -ItemType Directory -Force $downloads, $bin | Out-Null

function Get-CheckedArchive($Uri, $Destination, $Sha256) {
    if (-not (Test-Path $Destination)) { Invoke-WebRequest -Uri $Uri -OutFile $Destination }
    $stream = [System.IO.File]::OpenRead($Destination)
    try {
        $hasher = [System.Security.Cryptography.SHA256]::Create()
        $actual = ([System.BitConverter]::ToString($hasher.ComputeHash($stream))).Replace('-', '').ToLowerInvariant()
    } finally {
        $stream.Dispose()
    }
    if ($actual -ne $Sha256.ToLowerInvariant()) { throw "Checksum mismatch for $Destination (actual $actual)" }
}

$archive = Join-Path $downloads 'go1.26.6.windows-amd64.zip'
Get-CheckedArchive 'https://go.dev/dl/go1.26.6.windows-amd64.zip' $archive '5b6c5b556525810463b5c897b50dc7a82d6a3dc0bfaf55d990a7e9f31d6b2318'
if (-not (Test-Path (Join-Path $tools 'go/bin/go.exe'))) { Expand-Archive -LiteralPath $archive -DestinationPath $tools -Force }

$lintArchive = Join-Path $downloads 'golangci-lint-2.12.2-windows-amd64.zip'
Get-CheckedArchive 'https://github.com/golangci/golangci-lint/releases/download/v2.12.2/golangci-lint-2.12.2-windows-amd64.zip' $lintArchive 'bd42e3ebc8cb4ececb86941983baaf1dc221bbb04d838e94ce63b49cc91e02bb'
$lintExtract = Join-Path $tools 'extract-golangci'
if (-not (Test-Path (Join-Path $bin 'golangci-lint.exe'))) {
    New-Item -ItemType Directory -Force $lintExtract | Out-Null
    Expand-Archive -LiteralPath $lintArchive -DestinationPath $lintExtract -Force
    Copy-Item -LiteralPath (Join-Path $lintExtract 'golangci-lint-2.12.2-windows-amd64/golangci-lint.exe') -Destination (Join-Path $bin 'golangci-lint.exe')
}

Write-Host 'Installed project-local Go 1.26.6 and golangci-lint 2.12.2.'
Write-Host 'Global PATH and package-manager state were not changed.'
