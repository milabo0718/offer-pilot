$ErrorActionPreference = "Stop"

$root = Split-Path -Parent $MyInvocation.MyCommand.Path
$repo = Split-Path -Parent $root
Set-Location $repo

Write-Host "Stopping middleware services ..." -ForegroundColor Yellow

docker compose down
