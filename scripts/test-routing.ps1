$ErrorActionPreference = 'Stop'
$root = (Resolve-Path (Join-Path $PSScriptRoot '..')).Path
Set-Location $root
if (-not (Test-Path '.env')) { throw 'File .env belum dibuat.' }
$envMap = @{}
foreach ($line in Get-Content '.env') {
    if ($line -match '^([A-Z_0-9]+)=(.*)$') { $envMap[$matches[1]] = $matches[2].Trim() }
}
foreach ($name in @('HELLO_HOST','STATUS_HOST')) {
    $hostname = $envMap[$name]
    if (-not $hostname) { throw "Missing $name in .env" }
    Write-Host "Testing Traefik Host: $hostname"
    $status = docker run --rm --network homelab_edge curlimages/curl:8.23.0 -sS --connect-timeout 10 --max-time 30 -o /dev/null -w '%{http_code}' -H "Host: $hostname" http://traefik:80/
    if ($LASTEXITCODE -ne 0 -or "$status".Trim() -ne '200') { throw "Expected 200 for $hostname, received $status" }
    Write-Host "PASS - HTTP $status"
}
Write-Host 'Testing unknown Host (must NOT serve any container)'
$unknown = docker run --rm --network homelab_edge curlimages/curl:8.23.0 -sS --connect-timeout 10 --max-time 30 -o /dev/null -w '%{http_code}' -H 'Host: unknown.invalid' http://traefik:80/
if ($LASTEXITCODE -ne 0 -or "$unknown".Trim() -ne '404') { throw "Expected HTTP 404 for unknown.invalid, received $unknown" }
Write-Host 'PASS - HTTP 404 for unknown host.'
Write-Host 'NOTE: Cloudflare DNS and public HTTPS require separate browser verification.'
