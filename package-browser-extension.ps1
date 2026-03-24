param(
  [string]$OutputDir = ".\browser-extension\ozon-shop-bridge\dist"
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

$repoRoot = (Resolve-Path $PSScriptRoot).Path
$extRoot = (Resolve-Path (Join-Path $repoRoot "browser-extension\ozon-shop-bridge")).Path
$manifestPath = Join-Path $extRoot "manifest.json"

if (!(Test-Path $manifestPath)) {
  throw "manifest.json not found at $manifestPath"
}

$manifest = Get-Content $manifestPath -Raw -Encoding UTF8 | ConvertFrom-Json
$version = $manifest.version
if ([string]::IsNullOrWhiteSpace($version)) {
  throw "manifest version is empty"
}

$outputPath = Join-Path $repoRoot $OutputDir
if (!(Test-Path $outputPath)) {
  New-Item -ItemType Directory -Path $outputPath -Force | Out-Null
}

$resolvedOutputDir = (Resolve-Path $outputPath).Path
$zipBaseName = "ozon-shop-bridge-v$version"
$zipPath = Join-Path $resolvedOutputDir ($zipBaseName + ".zip")

if (Test-Path $zipPath) {
  try {
    Remove-Item -Force -ErrorAction Stop $zipPath
  } catch {
    $zipPath = Join-Path $resolvedOutputDir ($zipBaseName + "-" + (Get-Date -Format "yyyyMMddHHmmss") + ".zip")
  }
}

$tempDir = Join-Path $env:TEMP ("ozon-shop-bridge-pack-" + [guid]::NewGuid().ToString("N"))
New-Item -ItemType Directory -Path $tempDir -Force | Out-Null

try {
  Copy-ExtensionRuntimeTree -SourceRoot $extRoot -DestinationRoot $tempDir
  Compress-Archive -Path (Join-Path $tempDir "*") -DestinationPath $zipPath
  Write-Host "Packaged browser extension: $zipPath"
  $zipPath
} finally {
  if (Test-Path $tempDir) {
    Remove-Item -Recurse -Force $tempDir
  }
}
