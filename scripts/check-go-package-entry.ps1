$ErrorActionPreference = "Stop"
Set-StrictMode -Version Latest

$checks = @(
  @{
    Path = "start-dev.bat"
    MustContain = @("go run ./cmd/server")
    MustNotContain = @("go run cmd/server/main.go")
  },
  @{
    Path = "AGENTS.md"
    MustContain = @("cd backend && go run ./cmd/server", "cd backend && go build -o server ./cmd/server")
    MustNotContain = @("cd backend && go run cmd/server/main.go", "cd backend && go build -o server cmd/server/main.go")
  },
  @{
    Path = "CLAUDE.md"
    MustContain = @("go run ./cmd/server", "go build -o server ./cmd/server")
    MustNotContain = @("go run cmd/server/main.go", "go build -o server cmd/server/main.go")
  }
)

foreach ($check in $checks) {
  $content = Get-Content -Path $check.Path -Raw -Encoding UTF8

  foreach ($required in $check.MustContain) {
    if (-not $content.Contains($required)) {
      throw "$($check.Path) is missing expected command: $required"
    }
  }

  foreach ($forbidden in $check.MustNotContain) {
    if ($content.Contains($forbidden)) {
      throw "$($check.Path) still contains single-file Go entry command: $forbidden"
    }
  }
}

Write-Host "Go package entry command checks passed."
