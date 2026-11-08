# TEST MODE ONLY: Cloudflare Quick Tunnel (random *.trycloudflare.com URL, no account/domain/token).
# Exposes ONE demo app directly (bypasses Traefik host routing). URL changes on every start.
# Do NOT use for WAHA/9Router/databases. Stop it when finished: -Stop
param(
    [ValidateSet('hello','status')]
    [string]$Target = 'hello',
    [switch]$Stop
)
$ErrorActionPreference = 'Stop'
$name = 'homelab-quick-tunnel'
$image = 'cloudflare/cloudflared:2026.10.7'

if ($Stop) {
    docker rm -f $name *> $null
    Write-Host 'Quick tunnel dihentikan.'
    return
}

docker info *> $null
if ($LASTEXITCODE -ne 0) { throw 'Docker Engine belum aktif.' }
docker network inspect homelab_edge *> $null
if ($LASTEXITCODE -ne 0) { throw 'Network homelab_edge belum ada. Jalankan scripts/start-infra.ps1 -WithExamples dahulu.' }
$running = docker ps --filter "name=homelab-$Target-$Target" --format '{{.Names}}'
if (-not $running) { throw "Container demo '$Target' belum berjalan. Jalankan scripts/deploy-examples.ps1." }

docker rm -f $name *> $null
docker run -d --name $name --network homelab_edge $image tunnel --no-autoupdate --url "http://${Target}:80" | Out-Null
if ($LASTEXITCODE -ne 0) { throw 'Gagal memulai quick tunnel.' }

Write-Host 'Menunggu URL dari Cloudflare...'
$url = $null
for ($i = 0; $i -lt 30 -and -not $url; $i++) {
    Start-Sleep -Seconds 2
    $log = docker logs $name 2>&1 | Out-String
    if ($log -match 'https://[a-z0-9-]+\.trycloudflare\.com') { $url = $matches[0] }
}
if (-not $url) { throw "URL tidak ditemukan. Cek: docker logs $name" }
Write-Host "URL sementara untuk '$Target': $url" -ForegroundColor Green
Write-Host 'Hentikan dengan: scripts\quick-tunnel.ps1 -Stop'
