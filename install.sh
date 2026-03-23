#!/bin/bash
# caldav-cli installer
# Usage: curl -fsSL https://raw.githubusercontent.com/ksinistr/caldav-cli/main/install.sh | bash
# Or with custom options:
#   REPO_SLUG=owner/repo INSTALL_DIR=/path VERSION=vX.Y.Z bash install.sh

set -e

# Configuration with defaults
REPO_SLUG="${REPO_SLUG:-ksinistr/caldav-cli}"
INSTALL_DIR="${INSTALL_DIR:-$HOME/.local/bin}"
VERSION="${VERSION:-latest}"
BINARY_NAME="caldav-cli"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Functions
info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

error() {
    echo -e "${RED}[ERROR]${NC} $1"
    exit 1
}

# Detect platform
detect_platform() {
    OS="$(uname -s)"
    ARCH="$(uname -m)"

    case "$OS" in
        Linux)
            OS="linux"
            ;;
        Darwin)
            OS="darwin"
            ;;
        MINGW*|MSYS*|CYGWIN*)
            OS="windows"
            ;;
        *)
            error "Unsupported OS: $OS"
            ;;
    esac

    case "$ARCH" in
        x86_64|amd64)
            ARCH="amd64"
            ;;
        aarch64|arm64)
            ARCH="arm64"
            ;;
        i386|i686)
            ARCH="386"
            ;;
        *)
            error "Unsupported architecture: $ARCH"
            ;;
    esac

    echo "${OS}_${ARCH}"
}

# Get latest version from GitHub API
get_latest_version() {
    if command -v curl >/dev/null 2>&1; then
        curl -s "https://api.github.com/repos/${REPO_SLUG}/releases/latest" | grep '"tag_name"' | sed -E 's/.*"([^"]+)".*/\1/'
    elif command -v wget >/dev/null 2>&1; then
        wget -qO- "https://api.github.com/repos/${REPO_SLUG}/releases/latest" | grep '"tag_name"' | sed -E 's/.*"([^"]+)".*/\1/'
    else
        error "Neither curl nor wget is available"
    fi
}

# Main installation
main() {
    info "Installing ${BINARY_NAME}..."

    # Detect platform
    PLATFORM=$(detect_platform)
    info "Detected platform: ${PLATFORM}"

    # Determine version
    if [ "$VERSION" = "latest" ]; then
        VERSION=$(get_latest_version)
        if [ -z "$VERSION" ]; then
            error "Failed to determine latest version"
        fi
        info "Latest version: ${VERSION}"
    else
        info "Version: ${VERSION}"
    fi

    # Create install directory
    if [ ! -d "$INSTALL_DIR" ]; then
        info "Creating install directory: ${INSTALL_DIR}"
        mkdir -p "$INSTALL_DIR"
    fi

    # Determine binary name for platform
    if [ "$OS" = "windows" ]; then
        BINARY_NAME="${BINARY_NAME}.exe"
    fi

    # Download URL
    DOWNLOAD_URL="https://github.com/${REPO_SLUG}/releases/download/${VERSION}/${BINARY_NAME}_${PLATFORM}"

    info "Downloading from: ${DOWNLOAD_URL}"

    # Download to temporary file
    TMP_FILE=$(mktemp)
    trap "rm -f ${TMP_FILE}" EXIT

    if command -v curl >/dev/null 2>&1; then
        curl -fsSL "$DOWNLOAD_URL" -o "$TMP_FILE"
    elif command -v wget >/dev/null 2>&1; then
        wget -q "$DOWNLOAD_URL" -O "$TMP_FILE"
    else
        error "Neither curl nor wget is available"
    fi

    # Make executable
    chmod +x "$TMP_FILE"

    # Install
    info "Installing to: ${INSTALL_DIR}/${BINARY_NAME}"
    mv "$TMP_FILE" "${INSTALL_DIR}/${BINARY_NAME}"

    # Verify installation
    if [ -f "${INSTALL_DIR}/${BINARY_NAME}" ]; then
        info "Installation successful!"
        info ""
        info "Add to PATH (if not already):"
        info "  export PATH=\"${INSTALL_DIR}:\$PATH\""
        info ""
        info "Run '${BINARY_NAME} --help' to verify"
    else
        error "Installation failed - binary not found"
    fi
}

main
