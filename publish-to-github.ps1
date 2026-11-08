# Push this starter project to the user's GitHub repository.
# Requires Git for Windows and a GitHub account with write access.
# Does not use or store any GitHub token in this script.
[CmdletBinding()]
param(
    [string]$RepositoryUrl = 'https://github.com/Dffarhn2004/windows-to-server-docker.git'
)

$ErrorActionPreference = 'Stop'

function Invoke-Git {
    param([string[]]$GitArguments)
    & git @GitArguments
    if ($LASTEXITCODE -ne 0) {
        throw "Git command failed: git $($GitArguments -join ' ') (exit $LASTEXITCODE)"
    }
}

if (-not (Get-Command git -ErrorAction SilentlyContinue)) {
    throw 'Git belum terinstal. Install Git for Windows dari https://git-scm.com/downloads/win dan buka PowerShell baru.'
}

$destination = Join-Path $env:TEMP ("windows-to-server-docker-publish-" + $PID)
if (Test-Path -LiteralPath $destination) {
    throw "Folder sementara sudah ada: $destination"
}

Write-Host '1/5 Clone repository GitHub...' -ForegroundColor Cyan
Invoke-Git -GitArguments @('clone', $RepositoryUrl, $destination)

$existingFiles = @( & git -C $destination ls-files )
if ($LASTEXITCODE -ne 0) { throw 'Gagal memeriksa file yang ada di repository.' }
if ($existingFiles.Count -gt 0) {
    throw "Repository tujuan tidak kosong. Untuk mencegah overwrite, script berhenti. Periksa: $destination"
}

Write-Host '2/5 Salin starter files ke repository...' -ForegroundColor Cyan
Get-ChildItem -LiteralPath $PSScriptRoot -Force | ForEach-Object {
    if ($_.Name -ne '.git') {
        Copy-Item -LiteralPath $_.FullName -Destination $destination -Recurse -Force
    }
}

Write-Host '3/5 Siapkan commit Git...' -ForegroundColor Cyan
Invoke-Git -GitArguments @('-C', $destination, 'checkout', '-B', 'main')
Invoke-Git -GitArguments @('-C', $destination, 'add', '-A')

$staged = @( & git -C $destination diff --cached --name-only )
if ($LASTEXITCODE -ne 0) { throw 'Gagal memeriksa Git staging.' }
if ($staged.Count -eq 0) { throw 'Tidak ditemukan file baru untuk di-commit.' }
$bad = @($staged | Where-Object { $_ -match '(^|/)(\.env|id_rsa|id_ed25519|auth\.json)$' -or $_ -match '\.(pem|p12|pfx|key)$' })
if ($bad.Count -gt 0) {
    throw "File rahasia terdeteksi dalam staging; upload dibatalkan: $($bad -join ', ')"
}
Write-Host ("Menyiapkan {0} file." -f $staged.Count)

Write-Host '4/5 Commit...' -ForegroundColor Cyan
Invoke-Git -GitArguments @('-C', $destination, 'commit', '-m', 'feat: initial Windows Docker Cloudflare Traefik homelab')

Write-Host '5/5 Push ke GitHub...' -ForegroundColor Cyan
Write-Host 'Jika Git meminta login, autentikasi melalui Git Credential Manager/browser.'
Invoke-Git -GitArguments @('-C', $destination, 'push', '-u', 'origin', 'HEAD:main')

Write-Host 'SUCCESS: Project sudah dipush ke GitHub.' -ForegroundColor Green
Write-Host 'https://github.com/Dffarhn2004/windows-to-server-docker'
