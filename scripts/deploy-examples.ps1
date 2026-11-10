$ErrorActionPreference = 'Stop'
if ($PSVersionTable.PSVersion.Major -lt 6) { $ErrorActionPreference = 'Continue' } # PS 5.1 treats docker stderr as terminating
$root = (Resolve-Path (Join-Path $PSScriptRoot '..')).Path
Set-Location $root
if (-not (Test-Path '.env')) { throw 'File .env tidak ditemukan.' }
foreach ($path in @('projects/hello/compose.yaml', 'projects/status/compose.yaml')) {
    docker compose --env-file .env -f $path config --quiet
    if ($LASTEXITCODE -ne 0) { throw "Config invalid: $path" }
    docker compose --env-file .env -f $path up -d
    if ($LASTEXITCODE -ne 0) { throw "Deploy gagal: $path" }
}
Write-Host 'Dua contoh aktif. Cek routing lokal melalui scripts/test-routing.ps1.'
