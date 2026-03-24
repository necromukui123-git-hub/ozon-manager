param(
  [string]$OutputDir = ".\release",
  [string]$PackageName = "ozon-manager-win-x64"
)

$ErrorActionPreference = "Stop"

function Copy-ExtensionRuntimeTree {
  param(
    [string]$SourceRoot,
    [string]$DestinationRoot
  )

  $sourcePath = (Resolve-Path $SourceRoot).Path
  $excludedTopLevel = @("dist", "scripts")

  Get-ChildItem -Path $sourcePath -Recurse -File | ForEach-Object {
    $relativePath = $_.FullName.Substring($sourcePath.Length).TrimStart('\', '/')
    $segments = $relativePath -split '[\\/]'
    if ($segments.Length -gt 0 -and $excludedTopLevel -contains $segments[0]) {
      return
    }

    $targetPath = Join-Path $DestinationRoot $relativePath
    $targetDir = Split-Path -Parent $targetPath
    if (!(Test-Path $targetDir)) {
      New-Item -ItemType Directory -Path $targetDir -Force | Out-Null
    }

    Copy-Item -Path $_.FullName -Destination $targetPath -Force
  }
}

function Assert-LastExitCode {
  param(
    [string]$StepName
  )

  if ($LASTEXITCODE -ne 0) {
    throw "$StepName failed with exit code $LASTEXITCODE"
  }
}

$repoRoot = (Resolve-Path $PSScriptRoot).Path
$backendDir = Join-Path $repoRoot "backend"
$frontendDir = Join-Path $repoRoot "frontend"
$extensionDir = Join-Path $repoRoot "browser-extension\ozon-shop-bridge"
$releaseRoot = Join-Path $repoRoot $OutputDir
$resolvedPackageName = $PackageName
$stageDir = Join-Path $releaseRoot $resolvedPackageName
$releaseZip = Join-Path $releaseRoot ($resolvedPackageName + ".zip")
$backendBuildDir = Join-Path $backendDir ".dist"
$serverBuildOutput = Join-Path $backendBuildDir ("server-" + [guid]::NewGuid().ToString("N") + ".exe")
$manifest = Get-Content (Join-Path $extensionDir "manifest.json") -Raw -Encoding UTF8 | ConvertFrom-Json
$extensionZipName = "ozon-shop-bridge-v$($manifest.version).zip"

if (!(Test-Path $releaseRoot)) {
  New-Item -ItemType Directory -Path $releaseRoot -Force | Out-Null
}

if (Test-Path $stageDir) {
  try {
    Remove-Item -Recurse -Force -ErrorAction Stop $stageDir
  } catch {
    $resolvedPackageName = $PackageName + "-" + (Get-Date -Format "yyyyMMddHHmmss")
    $stageDir = Join-Path $releaseRoot $resolvedPackageName
    $releaseZip = Join-Path $releaseRoot ($resolvedPackageName + ".zip")
  }
}

if (Test-Path $releaseZip) {
  try {
    Remove-Item -Force -ErrorAction Stop $releaseZip
  } catch {
    if ($resolvedPackageName -eq $PackageName) {
      $resolvedPackageName = $PackageName + "-" + (Get-Date -Format "yyyyMMddHHmmss")
      $stageDir = Join-Path $releaseRoot $resolvedPackageName
    }
    $releaseZip = Join-Path $releaseRoot ($resolvedPackageName + ".zip")
  }
}

Write-Host "[1/5] Building frontend..."
Push-Location $frontendDir
try {
  cmd /c npm run build
  Assert-LastExitCode -StepName "Frontend build"
} finally {
  Pop-Location
}

Write-Host "[2/5] Building backend..."
if (!(Test-Path $backendBuildDir)) {
  New-Item -ItemType Directory -Path $backendBuildDir -Force | Out-Null
}

Push-Location $backendDir
try {
  $env:GOCACHE = Join-Path $backendDir ".gocache-build"
  go build -o $serverBuildOutput ./cmd/server
  Assert-LastExitCode -StepName "Backend build"
} finally {
  Pop-Location
}

Write-Host "[3/5] Packaging browser extension..."
$extensionZip = (& (Join-Path $repoRoot "package-browser-extension.ps1") -OutputDir ".\browser-extension\ozon-shop-bridge\dist" | Select-Object -Last 1)
if (!(Test-Path $extensionZip)) {
  throw "Browser extension package not found at $extensionZip"
}

Write-Host "[4/5] Assembling release directory..."
$serverStageDir = Join-Path $stageDir "server"
$webStageDir = Join-Path $serverStageDir "web"
$configStageDir = Join-Path $serverStageDir "config"
$databaseStageDir = Join-Path $serverStageDir "database"
$extensionStageDir = Join-Path $stageDir "browser-extension\ozon-shop-bridge"

New-Item -ItemType Directory -Path $webStageDir -Force | Out-Null
New-Item -ItemType Directory -Path $configStageDir -Force | Out-Null
New-Item -ItemType Directory -Path $databaseStageDir -Force | Out-Null
New-Item -ItemType Directory -Path $extensionStageDir -Force | Out-Null

Copy-Item -Path $serverBuildOutput -Destination (Join-Path $serverStageDir "server.exe") -Force
Copy-Item -Path (Join-Path $frontendDir "dist\*") -Destination $webStageDir -Recurse -Force
Copy-Item -Path (Join-Path $backendDir "config\config.yaml.example") -Destination (Join-Path $configStageDir "config.yaml.example") -Force
Copy-Item -Path (Join-Path $backendDir "migrations\init_database.sql") -Destination (Join-Path $databaseStageDir "init_database.sql") -Force
Get-ChildItem -Path (Join-Path $backendDir "migrations") -Filter "upgrade_*.sql" | ForEach-Object {
  Copy-Item -Path $_.FullName -Destination (Join-Path $databaseStageDir $_.Name) -Force
}
Copy-Item -Path (Join-Path $repoRoot "start-release-server.bat") -Destination (Join-Path $serverStageDir "start-ozon-manager.bat") -Force
Copy-Item -Path (Join-Path $repoRoot "README-windows-deploy.md") -Destination (Join-Path $stageDir "README-windows-deploy.md") -Force
Copy-ExtensionRuntimeTree -SourceRoot $extensionDir -DestinationRoot $extensionStageDir
Copy-Item -Path $extensionZip -Destination (Join-Path $stageDir "browser-extension\$extensionZipName") -Force

Write-Host "[5/5] Compressing release archive..."
Compress-Archive -Path $stageDir -DestinationPath $releaseZip

if (Test-Path $serverBuildOutput) {
  Remove-Item -Force $serverBuildOutput
}

Write-Host "Release folder: $stageDir"
Write-Host "Release zip:    $releaseZip"
