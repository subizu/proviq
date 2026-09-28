$ErrorActionPreference = "Stop"

$Repo = "subizu/proviq"
$Binary = "agent-proof"
$InstallDir = "$env:LOCALAPPDATA\Programs\agent-proof"

$Arch = if ([Environment]::Is64BitOperatingSystem) { "amd64" } else { "386" }
if ($env:PROCESSOR_ARCHITECTURE -eq "ARM64") {
    $Arch = "arm64"
}

Write-Host "Checking latest release of $Binary from GitHub..." -ForegroundColor Cyan

try {
    $Release = Invoke-RestMethod -Uri "https://api.github.com/repos/$Repo/releases/latest" -Headers @{"User-Agent"="agent-proof-installer"}
    $Tag = $Release.tag_name
} catch {
    $Tag = "v0.1.0"
}

$Version = $Tag.TrimStart("v")
$ArchiveName = "${Binary}_${Version}_windows_${Arch}.zip"
$DownloadUrl = "https://github.com/$Repo/releases/download/$Tag/$ArchiveName"

$TempZip = Join-Path $env:TEMP $ArchiveName
$TempExtract = Join-Path $env:TEMP "agent-proof-extract"

Write-Host "Downloading $Binary $Tag..." -ForegroundColor Cyan

try {
    Invoke-WebRequest -Uri $DownloadUrl -OutFile $TempZip -UseBasicParsing
    Expand-Archive -Path $TempZip -DestinationPath $TempExtract -Force

    if (-not (Test-Path $InstallDir)) {
        New-Item -ItemType Directory -Path $InstallDir -Force | Out-Null
    }

    Copy-Item -Path (Join-Path $TempExtract "$Binary.exe") -Destination (Join-Path $InstallDir "$Binary.exe") -Force

    # Add to User PATH if not already present
    $UserPath = [Environment]::GetEnvironmentVariable("PATH", "User")
    if ($UserPath -notlike "*$InstallDir*") {
        [Environment]::SetEnvironmentVariable("PATH", "$UserPath;$InstallDir", "User")
        $env:PATH = "$env:PATH;$InstallDir"
        Write-Host "Added $InstallDir to user PATH." -ForegroundColor Green
    }

    Write-Host "`n✓ Successfully installed $Binary to $InstallDir\$Binary.exe" -ForegroundColor Green
    Write-Host "Open a new terminal and run '$Binary --help' to get started!`n" -ForegroundColor Green
} catch {
    Write-Host "Pre-built binary download failed. Trying 'go install'..." -ForegroundColor Yellow
    if (Get-Command go -ErrorAction SilentlyContinue) {
        & go install "github.com/$Repo/cmd/$Binary@latest"
        Write-Host "`n✓ Successfully installed via 'go install'!" -ForegroundColor Green
    } else {
        Write-Error "Failed to install: $_"
    }
} finally {
    Remove-Item -Path $TempZip -Force -ErrorAction SilentlyContinue
    Remove-Item -Path $TempExtract -Recurse -Force -ErrorAction SilentlyContinue
}
