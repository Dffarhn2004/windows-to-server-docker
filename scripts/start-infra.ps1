param([switch]$WithExamples)
$ErrorActionPreference = 'Stop'
if ($PSVersionTable.PSVersion.Major -lt 6) { $ErrorActionPreference = 'Continue' } # PS 5.1 treats docker stderr as terminating
$root = (Resolve-Path (Join-Path $PSScriptRoot '..')).Path
Set-Location $root
if (-not (Test-Path '.env')) {
    Copy-Item '.env.example' '.env'
    throw 'File .env dibuat. Edit domain dan TUNNEL_TOKEN terlebih dahulu, lalu jalankan kembali.'
}
$lines = Get-Content '.env'
$token = ($lines | Where-Object { $_ -match '^TUNNEL_TOKEN=' } | Select-Object -First 1)
if (-not $token -or $token -match 'REPLACE_WITH' -or $token -eq 'TUNNEL_TOKEN=') {
    throw 'TUNNEL_TOKEN belum disetel di file .env.'
}
if (-not (Get-Command docker -ErrorAction SilentlyContinue)) { throw 'Docker CLI tidak ditemukan. Install Docker Desktop terlebih dahulu.' }
docker info *> $null
if ($LASTEXITCODE -ne 0) { throw 'Docker Engine belum aktif. Buka Docker Desktop dan ulangi.' }
docker compose version
if ($LASTEXITCODE -ne 0) { throw 'Docker Compose plugin tidak tersedia.' }
docker network inspect homelab_services *> $null
if ($LASTEXITCODE -ne 0) {
    docker network create homelab_services | Out-Null
    if ($LASTEXITCODE -ne 0) { throw 'Gagal membuat homelab_services network.' }
}
docker compose --env-file .env -f infra/compose.yaml config --quiet
if ($LASTEXITCODE -ne 0) { throw 'Konfigurasi infra tidak valid.' }
docker compose --env-file .env -f infra/compose.yaml up -d
if ($LASTEXITCODE -ne 0) { throw 'Gagal memulai infrastruktur.' }
if ($WithExamples) {
    & (Join-Path $PSScriptRoot 'deploy-examples.ps1')
    if ($LASTEXITCODE -ne 0) { throw 'Deploy contoh gagal.' }
}
Write-Host 'Infrastruktur aktif. Cek Cloudflare Published Application wildcard ke http://traefik:80.'
docker compose --env-file .env -f infra/compose.yaml ps
