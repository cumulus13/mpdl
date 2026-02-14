#!/bin/bash
# mpdl Installation Script
# Supports Linux and macOS

set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
REPO="cumulus13/mpdl"
INSTALL_DIR="/usr/local/bin"
CONFIG_DIR="${HOME}/.config/mpdl"

# Detect OS and Architecture
detect_platform() {
    local os=$(uname -s | tr '[:upper:]' '[:lower:]')
    local arch=$(uname -m)
    
    case "$os" in
        linux)
            OS="linux"
            ;;
        darwin)
            OS="darwin"
            ;;
        *)
            echo -e "${RED}Unsupported operating system: $os${NC}"
            exit 1
            ;;
    esac
    
    case "$arch" in
        x86_64|amd64)
            ARCH="amd64"
            ;;
        aarch64|arm64)
            ARCH="arm64"
            ;;
        armv7l|armv7)
            ARCH="armv7"
            ;;
        i386|i686)
            ARCH="386"
            ;;
        *)
            echo -e "${RED}Unsupported architecture: $arch${NC}"
            exit 1
            ;;
    esac
    
    PLATFORM="${OS}-${ARCH}"
    echo -e "${BLUE}Detected platform: ${PLATFORM}${NC}"
}

# Get latest release
get_latest_release() {
    echo -e "${BLUE}Fetching latest release...${NC}"
    
    LATEST_RELEASE=$(curl -s "https://api.github.com/repos/${REPO}/releases/latest" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')
    
    if [ -z "$LATEST_RELEASE" ]; then
        echo -e "${RED}Failed to fetch latest release${NC}"
        exit 1
    fi
    
    echo -e "${GREEN}Latest release: ${LATEST_RELEASE}${NC}"
}

# Download binary
download_binary() {
    BINARY_NAME="mpdl-${PLATFORM}"
    ARCHIVE_NAME="${BINARY_NAME}.tar.gz"
    DOWNLOAD_URL="https://github.com/${REPO}/releases/download/${LATEST_RELEASE}/${ARCHIVE_NAME}"
    
    echo -e "${BLUE}Downloading ${ARCHIVE_NAME}...${NC}"
    
    if ! curl -L -o "/tmp/${ARCHIVE_NAME}" "$DOWNLOAD_URL"; then
        echo -e "${RED}Failed to download binary${NC}"
        exit 1
    fi
    
    echo -e "${GREEN}Download complete${NC}"
}

# Extract and install
install_binary() {
    echo -e "${BLUE}Extracting archive...${NC}"
    
    cd /tmp
    tar -xzf "${ARCHIVE_NAME}"
    
    echo -e "${BLUE}Installing to ${INSTALL_DIR}...${NC}"
    
    if [ ! -w "$INSTALL_DIR" ]; then
        echo -e "${YELLOW}Installing with sudo...${NC}"
        sudo mv "${BINARY_NAME}" "${INSTALL_DIR}/mpdl"
        sudo chmod +x "${INSTALL_DIR}/mpdl"
    else
        mv "${BINARY_NAME}" "${INSTALL_DIR}/mpdl"
        chmod +x "${INSTALL_DIR}/mpdl"
    fi
    
    # Cleanup
    rm -f "/tmp/${ARCHIVE_NAME}"
    
    echo -e "${GREEN}Installation complete!${NC}"
}

# Create config directory
setup_config() {
    echo -e "${BLUE}Setting up configuration directory...${NC}"
    
    mkdir -p "$CONFIG_DIR"
    
    if [ ! -f "${CONFIG_DIR}/config.toml" ]; then
        echo -e "${YELLOW}Creating example configuration...${NC}"
        cat > "${CONFIG_DIR}/config.toml" <<EOF
# mpdl Configuration File
# Edit this file to customize your settings

[mpd]
host = "localhost"
port = "6600"
password = ""
timeout = 10
music_root = "${HOME}/Music"
config_path = "${HOME}/.config/mpd/mpd.conf"

[gntp]
host = "localhost"
port = 23053
password = ""
icon_mode = "binary"
enabled = true

[display]
show_album_art = true
use_color = true
EOF
        echo -e "${GREEN}Created config file: ${CONFIG_DIR}/config.toml${NC}"
    else
        echo -e "${YELLOW}Config file already exists, skipping...${NC}"
    fi
}

# Verify installation
verify_installation() {
    echo -e "${BLUE}Verifying installation...${NC}"
    
    if command -v mpdl >/dev/null 2>&1; then
        VERSION=$(mpdl --version 2>&1 | head -n1)
        echo -e "${GREEN}✓ mpdl installed successfully${NC}"
        echo -e "${GREEN}  $VERSION${NC}"
        return 0
    else
        echo -e "${RED}✗ Installation verification failed${NC}"
        return 1
    fi
}

# Show usage information
show_usage() {
    cat <<EOF

${GREEN}mpdl has been installed successfully!${NC}

Quick start:
  ${BLUE}mpdl --help${NC}           Show help
  ${BLUE}mpdl status${NC}           Show MPD status
  ${BLUE}mpdl monitor${NC}          Start monitor mode
  ${BLUE}mpdl play${NC}             Start playback

Configuration:
  ${BLUE}${CONFIG_DIR}/config.toml${NC}

Documentation:
  ${BLUE}https://github.com/${REPO}${NC}

Examples:
  ${BLUE}mpdl add ~/Music/song.mp3${NC}
  ${BLUE}mpdl volume 75${NC}
  ${BLUE}mpdl random on${NC}

EOF
}

# Uninstall function
uninstall() {
    echo -e "${YELLOW}Uninstalling mpdl...${NC}"
    
    if [ -f "${INSTALL_DIR}/mpdl" ]; then
        if [ ! -w "$INSTALL_DIR" ]; then
            sudo rm -f "${INSTALL_DIR}/mpdl"
        else
            rm -f "${INSTALL_DIR}/mpdl"
        fi
        echo -e "${GREEN}✓ Binary removed${NC}"
    else
        echo -e "${YELLOW}Binary not found${NC}"
    fi
    
    read -p "Remove configuration directory? (y/N) " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        rm -rf "$CONFIG_DIR"
        echo -e "${GREEN}✓ Configuration removed${NC}"
    fi
    
    echo -e "${GREEN}Uninstallation complete${NC}"
}

# Main installation flow
main() {
    echo -e "${BLUE}═══════════════════════════════════════${NC}"
    echo -e "${BLUE}  mpdl Installation Script${NC}"
    echo -e "${BLUE}═══════════════════════════════════════${NC}"
    echo
    
    # Check for uninstall flag
    if [ "$1" = "--uninstall" ] || [ "$1" = "-u" ]; then
        uninstall
        exit 0
    fi
    
    # Check for help flag
    if [ "$1" = "--help" ] || [ "$1" = "-h" ]; then
        cat <<EOF
Usage: $0 [OPTIONS]

Options:
  -h, --help       Show this help message
  -u, --uninstall  Uninstall mpdl

This script will:
  1. Detect your platform
  2. Download the latest release
  3. Install to ${INSTALL_DIR}
  4. Create configuration directory

EOF
        exit 0
    fi
    
    # Check for curl
    if ! command -v curl >/dev/null 2>&1; then
        echo -e "${RED}Error: curl is required but not installed${NC}"
        exit 1
    fi
    
    # Check for tar
    if ! command -v tar >/dev/null 2>&1; then
        echo -e "${RED}Error: tar is required but not installed${NC}"
        exit 1
    fi
    
    # Run installation steps
    detect_platform
    get_latest_release
    download_binary
    install_binary
    setup_config
    
    echo
    
    if verify_installation; then
        show_usage
        exit 0
    else
        echo -e "${RED}Installation failed. Please check the errors above.${NC}"
        exit 1
    fi
}

# Run main function
main "$@"
