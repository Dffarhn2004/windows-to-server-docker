$ErrorActionPreference = 'Stop'
if ($PSVersionTable.PSVersion.Major -lt 6) { $ErrorActionPreference = 'Continue' } # PS 5.1 treats docker stderr as terminating
$root = (Resolve-Path (Join-Path $PSScriptRoot '..')).Path
Set-Location $root
Write-Warning 'Menghentikan TUNNEL dan TRAEFIK menyebabkan semua aplikasi publik offline.'
if (-not (Test-Path '.env')) { throw '.env tidak ada.' }
docker compose --env-file .env -f infra/compose.yaml stop
if ($LASTEXITCODE -ne 0) { throw 'Perintah stop gagal.' }
Write-Host 'Container dihentikan tanpa menghapus volumes.'
