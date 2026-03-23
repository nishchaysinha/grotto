#!/bin/sh
# Grotto installer script
# Usage: curl -sSL https://raw.githubusercontent.com/nishchaysinha/grotto/main/install.sh | sh

set -e

REPO="nishchaysinha/grotto"
INSTALL_DIR="${INSTALL_DIR:-/usr/local/bin}"
BINARY_NAME="grotto"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

log_info() {
    printf "${GREEN}[INFO]${NC} %s\n" "$1"
}

log_warn() {
    printf "${YELLOW}[WARN]${NC} %s\n" "$1"
}

log_error() {
    printf "${RED}[ERROR]${NC} %s\n" "$1"
}

# Detect OS
detect_os() {
    case "$(uname -s)" in
        Linux*)  echo "linux" ;;
        Darwin*) echo "darwin" ;;
        MINGW*|MSYS*|CYGWIN*) echo "windows" ;;
        *)
            log_error "Unsupported OS: $(uname -s)"
            exit 1
            ;;
    esac
}

# Detect architecture
detect_arch() {
    case "$(uname -m)" in
        x86_64|amd64) echo "amd64" ;;
        aarch64|arm64) echo "arm64" ;;
        *)
            log_error "Unsupported architecture: $(uname -m)"
            exit 1
            ;;
    esac
}

# Get latest release version
get_latest_version() {
    curl -sL "https://api.github.com/repos/${REPO}/releases/latest" | \
        grep '"tag_name":' | \
        sed -E 's/.*"([^"]+)".*/\1/'
}

# Download and verify binary
download_binary() {
    local os="$1"
    local arch="$2"
    local version="$3"
    local tmpdir="$4"

    local ext="tar.gz"
    if [ "$os" = "windows" ]; then
        ext="zip"
    fi

    local archive_name="grotto-${version#v}-${os}-${arch}.${ext}"
    local download_url="https://github.com/${REPO}/releases/download/${version}/${archive_name}"
    local checksum_url="https://github.com/${REPO}/releases/download/${version}/checksums.txt"

    log_info "Downloading ${archive_name}..."
    curl -sL -o "${tmpdir}/${archive_name}" "$download_url"

    log_info "Downloading checksums..."
    curl -sL -o "${tmpdir}/checksums.txt" "$checksum_url"

    log_info "Verifying checksum..."
    cd "$tmpdir"
    if command -v sha256sum > /dev/null 2>&1; then
        grep "${archive_name}" checksums.txt | sha256sum -c - > /dev/null 2>&1
    elif command -v shasum > /dev/null 2>&1; then
        grep "${archive_name}" checksums.txt | shasum -a 256 -c - > /dev/null 2>&1
    else
        log_warn "Neither sha256sum nor shasum found, skipping checksum verification"
    fi

    log_info "Extracting archive..."
    if [ "$ext" = "tar.gz" ]; then
        tar -xzf "${archive_name}"
    else
        unzip -q "${archive_name}"
    fi

    echo "${tmpdir}/grotto"
}

# Install binary
install_binary() {
    local binary_path="$1"
    local install_dir="$2"

    # Check if we can write to install dir
    if [ -w "$install_dir" ]; then
        cp "$binary_path" "${install_dir}/${BINARY_NAME}"
        chmod +x "${install_dir}/${BINARY_NAME}"
    else
        log_info "Requesting sudo to install to ${install_dir}..."
        sudo cp "$binary_path" "${install_dir}/${BINARY_NAME}"
        sudo chmod +x "${install_dir}/${BINARY_NAME}"
    fi
}

main() {
    local os
    local arch
    local version
    local tmpdir

    os=$(detect_os)
    arch=$(detect_arch)

    log_info "Detected OS: ${os}, Architecture: ${arch}"

    version=$(get_latest_version)
    if [ -z "$version" ]; then
        log_error "Failed to get latest version"
        exit 1
    fi
    log_info "Latest version: ${version}"

    tmpdir=$(mktemp -d)
    trap 'rm -rf "$tmpdir"' EXIT

    binary_path=$(download_binary "$os" "$arch" "$version" "$tmpdir")

    # Check if install dir exists, create if needed
    if [ ! -d "$INSTALL_DIR" ]; then
        log_info "Creating install directory: ${INSTALL_DIR}"
        if [ -w "$(dirname "$INSTALL_DIR")" ]; then
            mkdir -p "$INSTALL_DIR"
        else
            sudo mkdir -p "$INSTALL_DIR"
        fi
    fi

    install_binary "$binary_path" "$INSTALL_DIR"

    log_info "Successfully installed grotto to ${INSTALL_DIR}/${BINARY_NAME}"
    log_info "Run 'grotto --version' to verify the installation"

    # Check if install dir is in PATH
    case ":$PATH:" in
        *":${INSTALL_DIR}:"*) ;;
        *)
            log_warn "${INSTALL_DIR} is not in your PATH"
            log_warn "Add it to your PATH by running:"
            log_warn "  export PATH=\"\$PATH:${INSTALL_DIR}\""
            ;;
    esac
}

main "$@"
