param(
    [Parameter(Mandatory=$true)]
    [ValidateSet('waha','9router','data')]
    [string]$Service
)
$ErrorActionPreference = 'Stop'
$root = (Resolve-Path (Join-Path $PSScriptRoot '..')).Path
Set-Location $root
if (-not (Test-Path '.env')) { throw 'Copy .env.example to .env first.' }
$values = @{}
foreach ($line in Get-Content '.env') {
    if ($line -match '^([A-Z_0-9]+)=(.*)$') { $values[$matches[1]] = $matches[2].Trim() }
}
$required = switch ($Service) {
    'waha'    { @('WAHA_API_KEY','WAHA_DASHBOARD_USERNAME','WAHA_DASHBOARD_PASSWORD') }
    '9router' { @('ROUTER_JWT_SECRET','ROUTER_API_KEY_SECRET','ROUTER_MACHINE_ID_SALT','ROUTER_INITIAL_PASSWORD') }
    'data'    { @('POSTGRES_PASSWORD','REDIS_PASSWORD') }
}
foreach ($name in $required) {
    $value = $values[$name]
    if (-not $value -or $value -match '^REPLACE_' -or $value -eq 'changeme' -or $value.Length -lt 12) {
        throw "Set a real strong value for $name in .env. Do NOT paste secret values into chat."
    }
}
docker network inspect homelab_services *> $null
if ($LASTEXITCODE -ne 0) { throw 'Network homelab_services is missing; run scripts/start-infra.ps1 first.' }
$composeFile = "projects/$Service/compose.yaml"
docker compose --env-file .env -f $composeFile config --quiet
if ($LASTEXITCODE -ne 0) { throw "Config error: $composeFile" }
docker compose --env-file .env -f $composeFile up -d
if ($LASTEXITCODE -ne 0) { throw "Deployment failed: $composeFile" }
Write-Host "Optional service $Service started. No host ports should be published."
docker compose --env-file .env -f $composeFile ps
