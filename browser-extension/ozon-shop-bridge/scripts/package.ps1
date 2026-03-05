param(
  [string]$OutputDir = "..\dist"
)

$ErrorActionPreference = "Stop"

$scriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$extRoot = Resolve-Path (Join-Path $scriptDir "..")
$manifestPath = Join-Path $extRoot "manifest.json"

if (!(Test-Path $manifestPath)) {
  throw "manifest.json not found at $manifestPath"
}

$manifest = Get-Content $manifestPath -Raw -Encoding UTF8 | ConvertFrom-Json
$version = $manifest.version
if ([string]::IsNullOrWhiteSpace($version)) {
  throw "manifest version is empty"
}

$outputPath = Join-Path $scriptDir $OutputDir
$resolvedOutputDir = Resolve-Path $outputPath -ErrorAction SilentlyContinue
if (!$resolvedOutputDir) {
  New-Item -ItemType Directory -Path $outputPath | Out-Null
  $resolvedOutputDir = Resolve-Path $outputPath
}

$zipName = "ozon-shop-bridge-v$version.zip"
$zipPath = Join-Path $resolvedOutputDir $zipName

if (Test-Path $zipPath) {
  Remove-Item -Force $zipPath
}

$tempDir = Join-Path $env:TEMP ("ozon-shop-bridge-pack-" + [guid]::NewGuid().ToString("N"))
New-Item -ItemType Directory -Path $tempDir | Out-Null

try {
  Copy-Item -Path (Join-Path $extRoot "manifest.json") -Destination $tempDir
  Copy-Item -Path (Join-Path $extRoot "background.js") -Destination $tempDir
  Copy-Item -Path (Join-Path $extRoot "content-auth-sync.js") -Destination $tempDir
  Copy-Item -Path (Join-Path $extRoot "popup.html") -Destination $tempDir
  Copy-Item -Path (Join-Path $extRoot "popup.js") -Destination $tempDir
  Copy-Item -Path (Join-Path $extRoot "README.md") -Destination $tempDir

  Compress-Archive -Path (Join-Path $tempDir "*") -DestinationPath $zipPath
  Write-Host "Packaged: $zipPath"
} finally {
  if (Test-Path $tempDir) {
    Remove-Item -Recurse -Force $tempDir
  }
}
