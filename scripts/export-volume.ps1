param(
    [Parameter(Mandatory=$true)][string]$Volume,
    [string]$OutputDir = 'backups'
)
$ErrorActionPreference = 'Stop'
if ($Volume -notmatch '^[a-zA-Z0-9][a-zA-Z0-9_.-]*$') { throw 'Nama volume tidak valid.' }
$root = (Resolve-Path (Join-Path $PSScriptRoot '..')).Path
$dest = Join-Path $root $OutputDir
New-Item -ItemType Directory -Force -Path $dest | Out-Null
$dest = (Resolve-Path $dest).Path
$filename = "$Volume-$(Get-Date -Format 'yyyyMMdd-HHmmss').tar.gz"
Write-Warning 'STOP container penulis data sebelum backup volume. Untuk PostgreSQL gunakan pg_dump dan metode konsisten khusus database.'
docker volume inspect $Volume *> $null
if ($LASTEXITCODE -ne 0) { throw "Docker volume tidak ada: $Volume" }
docker run --rm --mount "type=volume,source=$Volume,target=/source,readonly" --mount "type=bind,source=$dest,target=/backup" alpine:3.22 sh -c "cd /source && tar czf /backup/$filename ."
if ($LASTEXITCODE -ne 0) { throw 'Backup gagal.' }
Write-Host "Backup dibuat: $(Join-Path $dest $filename)"
Write-Host 'Uji restore di mesin terpisah sebelum mengandalkan backup.'
