param(
  [string]$EnvFile = ".env.local",
  [ValidateSet("backend", "rag-ingest")]
  [string]$Mode = "backend",
  [string]$IngestDir = "./examples/rag_data_structured_strict",
  [switch]$Mock
)

$ErrorActionPreference = "Stop"

if (-not (Test-Path -LiteralPath $EnvFile)) {
  throw "Environment file not found: $EnvFile"
}

Get-Content -LiteralPath $EnvFile | ForEach-Object {
  $line = $_.Trim()
  if (-not $line -or $line.StartsWith("#")) { return }

  $idx = $line.IndexOf("=")
  if ($idx -lt 1) { return }

  $name = $line.Substring(0, $idx).Trim()
  $value = $line.Substring($idx + 1).Trim().Trim("'`"")

  [System.Environment]::SetEnvironmentVariable($name, $value, "Process")
}

if ($Mode -eq "backend") {
  Write-Host "Environment loaded. Starting backend..." -ForegroundColor Green
  go run .
  exit $LASTEXITCODE
}

$mockFlag = if ($Mock.IsPresent) { "true" } else { "false" }
Write-Host "Environment loaded. Starting rag_ingest..." -ForegroundColor Green
Write-Host "dir=$IngestDir mock=$mockFlag" -ForegroundColor DarkGray
go run ./cmd/rag_ingest -dir $IngestDir -mock=$mockFlag
exit $LASTEXITCODE