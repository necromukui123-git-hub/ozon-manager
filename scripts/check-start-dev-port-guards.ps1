$ErrorActionPreference = "Stop"
Set-StrictMode -Version Latest

$content = Get-Content -Path "start-dev.bat" -Raw -Encoding UTF8

$requiredSnippets = @(
  "call :find_listening_pid 8080 BACKEND_PID",
  "call :find_listening_pid 5173 FRONTEND_PID",
  "Backend already listening on port 8080",
  "Frontend already listening on port 5173",
  ":find_listening_pid"
)

foreach ($snippet in $requiredSnippets) {
  if (-not $content.Contains($snippet)) {
    throw "start-dev.bat is missing expected port-guard snippet: $snippet"
  }
}

Write-Host "start-dev.bat port guard checks passed."
