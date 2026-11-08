$ErrorActionPreference = 'Stop'
$root = (Resolve-Path (Join-Path $PSScriptRoot '..')).Path
Set-Location $root
if (-not (Test-Path '.env')) { throw 'Create .env first.' }
foreach ($file in @('infra/compose.yaml','projects/hello/compose.yaml','projects/status/compose.yaml')) {
    Write-Host "Validating $file"
    docker compose --env-file .env -f $file config --quiet
    if ($LASTEXITCODE -ne 0) { throw "Compose validation failed: $file" }
}
Write-Host 'Core Compose config validation passed (runtime/network not tested).'
