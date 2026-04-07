param(
  [string]$EnvFile = ".env.local"
)

$ErrorActionPreference = "Stop"

Get-Content $EnvFile | ForEach-Object {
  $line = $_.Trim()
  if (-not $line -or $line.StartsWith("#")) { return }

  $idx = $line.IndexOf("=")
  if ($idx -lt 1) { return }

  $name = $line.Substring(0, $idx).Trim()
  $value = $line.Substring($idx + 1).Trim()

  [System.Environment]::SetEnvironmentVariable($name, $value, "Process")
}

Write-Host "Environment loaded. Starting backend..." -ForegroundColor Green
go run .