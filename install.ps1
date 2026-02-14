# mpdl Installation Script for Windows
# Run with: powershell -ExecutionPolicy Bypass -File install.ps1

param(
    [switch]$Uninstall,
    [switch]$Help
)

$ErrorActionPreference = "Stop"

# Configuration
$Repo = "cumulus13/mpdl"
$InstallDir = "$env:ProgramFiles\mpdl"
$ConfigDir = "$env:APPDATA\mpdl"

# Colors
function Write-ColorOutput {
    param(
        [string]$Message,
        [string]$Color = "White"
    )
    Write-Host $Message -ForegroundColor $Color
}

# Detect architecture
function Get-Architecture {
    $arch = $env:PROCESSOR_ARCHITECTURE
    switch ($arch) {
        "AMD64" { return "amd64" }
        "ARM64" { return "arm64" }
        "x86" { return "386" }
        default {
            Write-ColorOutput "Unsupported architecture: $arch" "Red"
            exit 1
        }
    }
}

# Get latest release
function Get-LatestRelease {
    Write-ColorOutput "Fetching latest release..." "Blue"
    
    try {
        $response = Invoke-RestMethod -Uri "https://api.github.com/repos/$Repo/releases/latest"
        $version = $response.tag_name
        
        if (-not $version) {
            throw "Failed to get version"
        }
        
        Write-ColorOutput "Latest release: $version" "Green"
        return $version
    }
    catch {
        Write-ColorOutput "Failed to fetch latest release: $_" "Red"
        exit 1
    }
}

# Download binary
function Download-Binary {
    param(
        [string]$Version,
        [string]$Arch
    )
    
    $binaryName = "mpdl-windows-$Arch.exe"
    $archiveName = "$binaryName.zip"
    $downloadUrl = "https://github.com/$Repo/releases/download/$Version/$archiveName"
    $tempFile = "$env:TEMP\$archiveName"
    
    Write-ColorOutput "Downloading $archiveName..." "Blue"
    
    try {
        Invoke-WebRequest -Uri $downloadUrl -OutFile $tempFile
        Write-ColorOutput "Download complete" "Green"
        return $tempFile
    }
    catch {
        Write-ColorOutput "Failed to download binary: $_" "Red"
        exit 1
    }
}

# Extract and install
function Install-Binary {
    param(
        [string]$ArchivePath,
        [string]$Arch
    )
    
    Write-ColorOutput "Extracting archive..." "Blue"
    
    $tempDir = "$env:TEMP\mpdl-install"
    if (Test-Path $tempDir) {
        Remove-Item -Path $tempDir -Recurse -Force
    }
    New-Item -ItemType Directory -Path $tempDir | Out-Null
    
    try {
        Expand-Archive -Path $ArchivePath -DestinationPath $tempDir -Force
    }
    catch {
        Write-ColorOutput "Failed to extract archive: $_" "Red"
        exit 1
    }
    
    Write-ColorOutput "Installing to $InstallDir..." "Blue"
    
    # Create install directory
    if (-not (Test-Path $InstallDir)) {
        try {
            New-Item -ItemType Directory -Path $InstallDir -Force | Out-Null
        }
        catch {
            Write-ColorOutput "Failed to create install directory. Run as Administrator." "Red"
            exit 1
        }
    }
    
    # Copy binary
    $binaryName = "mpdl-windows-$Arch.exe"
    $sourcePath = Join-Path $tempDir $binaryName
    $destPath = Join-Path $InstallDir "mpdl.exe"
    
    try {
        Copy-Item -Path $sourcePath -Destination $destPath -Force
        Write-ColorOutput "Installation complete!" "Green"
    }
    catch {
        Write-ColorOutput "Failed to copy binary. Run as Administrator." "Red"
        exit 1
    }
    
    # Cleanup
    Remove-Item -Path $tempDir -Recurse -Force
    Remove-Item -Path $ArchivePath -Force
}

# Add to PATH
function Add-ToPath {
    Write-ColorOutput "Adding to PATH..." "Blue"
    
    $currentPath = [Environment]::GetEnvironmentVariable("Path", "User")
    
    if ($currentPath -notlike "*$InstallDir*") {
        try {
            $newPath = "$currentPath;$InstallDir"
            [Environment]::SetEnvironmentVariable("Path", $newPath, "User")
            Write-ColorOutput "Added to PATH (restart terminal to use)" "Green"
        }
        catch {
            Write-ColorOutput "Failed to add to PATH: $_" "Yellow"
            Write-ColorOutput "Please manually add '$InstallDir' to your PATH" "Yellow"
        }
    }
    else {
        Write-ColorOutput "Already in PATH" "Yellow"
    }
}

# Setup configuration
function Setup-Config {
    Write-ColorOutput "Setting up configuration directory..." "Blue"
    
    if (-not (Test-Path $ConfigDir)) {
        New-Item -ItemType Directory -Path $ConfigDir | Out-Null
    }
    
    $configFile = Join-Path $ConfigDir "config.toml"
    
    if (-not (Test-Path $configFile)) {
        Write-ColorOutput "Creating example configuration..." "Yellow"
        
        $configContent = @"
# mpdl Configuration File
# Edit this file to customize your settings

[mpd]
host = "localhost"
port = "6600"
password = ""
timeout = 10
music_root = "C:/Musics"
config_path = "C:/mpd/mpd.conf"

[gntp]
host = "localhost"
port = 23053
password = ""
icon_mode = "binary"
enabled = true

[display]
show_album_art = true
use_color = true
"@
        
        Set-Content -Path $configFile -Value $configContent
        Write-ColorOutput "Created config file: $configFile" "Green"
    }
    else {
        Write-ColorOutput "Config file already exists, skipping..." "Yellow"
    }
}

# Verify installation
function Test-Installation {
    Write-ColorOutput "Verifying installation..." "Blue"
    
    $exePath = Join-Path $InstallDir "mpdl.exe"
    
    if (Test-Path $exePath) {
        Write-ColorOutput "✓ mpdl installed successfully" "Green"
        
        # Try to get version (may fail if not in PATH yet)
        try {
            $version = & $exePath --version 2>&1 | Select-Object -First 1
            Write-ColorOutput "  $version" "Green"
        }
        catch {
            Write-ColorOutput "  Installed at: $exePath" "Green"
        }
        
        return $true
    }
    else {
        Write-ColorOutput "✗ Installation verification failed" "Red"
        return $false
    }
}

# Show usage
function Show-Usage {
    Write-Host ""
    Write-ColorOutput "mpdl has been installed successfully!" "Green"
    Write-Host ""
    Write-Host "Quick start:"
    Write-ColorOutput "  mpdl --help" "Blue"
    Write-Host "           Show help"
    Write-ColorOutput "  mpdl status" "Blue"
    Write-Host "           Show MPD status"
    Write-ColorOutput "  mpdl monitor" "Blue"
    Write-Host "          Start monitor mode"
    Write-ColorOutput "  mpdl play" "Blue"
    Write-Host "             Start playback"
    Write-Host ""
    Write-Host "Configuration:"
    Write-ColorOutput "  $ConfigDir\config.toml" "Blue"
    Write-Host ""
    Write-Host "Documentation:"
    Write-ColorOutput "  https://github.com/$Repo" "Blue"
    Write-Host ""
    Write-Host "Examples:"
    Write-ColorOutput "  mpdl add C:\Music\song.mp3" "Blue"
    Write-ColorOutput "  mpdl volume 75" "Blue"
    Write-ColorOutput "  mpdl random on" "Blue"
    Write-Host ""
    Write-ColorOutput "Note: You may need to restart your terminal for PATH changes to take effect" "Yellow"
    Write-Host ""
}

# Uninstall
function Uninstall-Mpdl {
    Write-ColorOutput "Uninstalling mpdl..." "Yellow"
    
    if (Test-Path $InstallDir) {
        try {
            Remove-Item -Path $InstallDir -Recurse -Force
            Write-ColorOutput "✓ Binary removed" "Green"
        }
        catch {
            Write-ColorOutput "Failed to remove binary. Run as Administrator." "Red"
        }
    }
    else {
        Write-ColorOutput "Installation directory not found" "Yellow"
    }
    
    $response = Read-Host "Remove configuration directory? (y/N)"
    if ($response -eq 'y' -or $response -eq 'Y') {
        if (Test-Path $ConfigDir) {
            Remove-Item -Path $ConfigDir -Recurse -Force
            Write-ColorOutput "✓ Configuration removed" "Green"
        }
    }
    
    # Remove from PATH
    $currentPath = [Environment]::GetEnvironmentVariable("Path", "User")
    if ($currentPath -like "*$InstallDir*") {
        $newPath = $currentPath -replace [regex]::Escape(";$InstallDir"), ""
        $newPath = $newPath -replace [regex]::Escape("$InstallDir;"), ""
        [Environment]::SetEnvironmentVariable("Path", $newPath, "User")
        Write-ColorOutput "✓ Removed from PATH" "Green"
    }
    
    Write-ColorOutput "Uninstallation complete" "Green"
}

# Show help
function Show-Help {
    Write-Host @"
mpdl Installation Script for Windows

Usage: powershell -ExecutionPolicy Bypass -File install.ps1 [OPTIONS]

Options:
  -Uninstall    Uninstall mpdl
  -Help         Show this help message

This script will:
  1. Detect your platform
  2. Download the latest release
  3. Install to $InstallDir
  4. Add to PATH
  5. Create configuration directory

Note: You may need to run as Administrator for installation.
"@
}

# Main installation
function Main {
    Write-Host ""
    Write-ColorOutput "═══════════════════════════════════════" "Blue"
    Write-ColorOutput "  mpdl Installation Script for Windows" "Blue"
    Write-ColorOutput "═══════════════════════════════════════" "Blue"
    Write-Host ""
    
    if ($Help) {
        Show-Help
        exit 0
    }
    
    if ($Uninstall) {
        Uninstall-Mpdl
        exit 0
    }
    
    # Check if running as admin (recommended but not required)
    $isAdmin = ([Security.Principal.WindowsPrincipal] [Security.Principal.WindowsIdentity]::GetCurrent()).IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)
    
    if (-not $isAdmin) {
        Write-ColorOutput "Warning: Not running as Administrator" "Yellow"
        Write-ColorOutput "Installation may fail. Consider running as Administrator." "Yellow"
        Write-Host ""
    }
    
    # Run installation steps
    $arch = Get-Architecture
    Write-ColorOutput "Detected architecture: windows-$arch" "Blue"
    
    $version = Get-LatestRelease
    $archivePath = Download-Binary -Version $version -Arch $arch
    Install-Binary -ArchivePath $archivePath -Arch $arch
    Add-ToPath
    Setup-Config
    
    Write-Host ""
    
    if (Test-Installation) {
        Show-Usage
        exit 0
    }
    else {
        Write-ColorOutput "Installation failed. Please check the errors above." "Red"
        exit 1
    }
}

# Run main
Main
