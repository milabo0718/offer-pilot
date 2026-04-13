$ErrorActionPreference = "Continue"

function Check-Cmd($name) {
  $cmd = Get-Command $name -ErrorAction SilentlyContinue
  if ($null -eq $cmd) {
    Write-Host "[MISSING] $name" -ForegroundColor Red
    return $false
  }
  Write-Host "[OK] $name -> $($cmd.Source)" -ForegroundColor Green
  return $true
}

Write-Host "=== OfferPilot Env Check (Windows) ===" -ForegroundColor Cyan

$ok = $true
$ok = (Check-Cmd git) -and $ok
$ok = (Check-Cmd node) -and $ok
$ok = (Check-Cmd npm) -and $ok
$ok = (Check-Cmd docker) -and $ok
$ok = (Check-Cmd go) -and $ok

Write-Host "\n--- Versions ---" -ForegroundColor Cyan
try { git --version } catch {}
try { node -v } catch {}
try { npm -v } catch {}
try { docker version } catch {}
try { docker compose version } catch {}
try { go version } catch {}

if (-not $ok) {
  Write-Host "\nEnvironment not ready: install missing tools (focus: Go)." -ForegroundColor Yellow
  exit 1
}

Write-Host "\nEnvironment ready." -ForegroundColor Green
