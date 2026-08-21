$ErrorActionPreference = 'Continue'
$versions = @{}
Get-Content '.tool-versions' | ForEach-Object {
    $parts = $_ -split '\s+', 2
    if ($parts.Count -eq 2) { $versions[$parts[0]] = $parts[1] }
}
$commands = [ordered]@{
    golang = @((if (Test-Path '.tools/go/bin/go.exe') { '.tools/go/bin/go.exe' } else { 'go' }), 'version')
    nodejs = @('node', '--version')
    pnpm = @('pnpm', '--version')
    postgres = @('docker', 'image', 'inspect', 'postgres:17.6-alpine', '--format', '{{index .RepoTags 0}}')
    redis = @('docker', 'image', 'inspect', 'redis:8.2.1-alpine', '--format', '{{index .RepoTags 0}}')
    kafka = @('docker', 'image', 'inspect', 'apache/kafka:4.1.0', '--format', '{{index .RepoTags 0}}')
    sqlc = @((if (Test-Path '.tools/bin/sqlc.exe') { '.tools/bin/sqlc.exe' } else { 'sqlc' }), 'version')
    golang-migrate = @((if (Test-Path '.tools/bin/migrate.exe') { '.tools/bin/migrate.exe' } else { 'migrate' }), '-version')
    buf = @((if (Test-Path '.tools/bin/buf.exe') { '.tools/bin/buf.exe' } else { 'buf' }), '--version')
    protoc = @('protoc', '--version')
    golangci-lint = @((if (Test-Path '.tools/bin/golangci-lint.exe') { '.tools/bin/golangci-lint.exe' } else { 'golangci-lint' }), '--version')
    k6 = @((if (Test-Path '.tools/bin/k6.exe') { '.tools/bin/k6.exe' } else { 'k6' }), 'version')
}
$failed = $false
foreach ($entry in $commands.GetEnumerator()) {
    try {
        $executable = $entry.Value[0]
        $arguments = @($entry.Value | Select-Object -Skip 1)
        $actual = (& $executable @arguments 2>&1 | Select-Object -First 1)
        if ($LASTEXITCODE -ne 0) { throw $actual }
        Write-Host "[ok] $($entry.Key) pinned=$($versions[$entry.Key]) actual=$actual"
    } catch {
        Write-Host "[missing] $($entry.Key) pinned=$($versions[$entry.Key])"
        $failed = $true
    }
}
if ($failed) { exit 1 }
