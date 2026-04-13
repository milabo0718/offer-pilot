param(
  [string]$EnvFile = ".env"
)

$ErrorActionPreference = "Stop"

$root = Split-Path -Parent $MyInvocation.MyCommand.Path
$repo = Split-Path -Parent $root
Set-Location $repo

if (Test-Path $EnvFile) {
  Write-Host "Using compose env file: $EnvFile" -ForegroundColor Cyan
}

Write-Host "Starting middleware services (mysql/redis/rabbitmq) ..." -ForegroundColor Green
# docker compose 会自动读取当前目录下的 .env；这里允许用户传入不同文件名
if ($EnvFile -ne ".env" -and (Test-Path $EnvFile)) {
  Copy-Item $EnvFile ".env" -Force
}

docker compose up -d

docker compose ps
