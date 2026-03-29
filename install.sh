#!/bin/sh
# Grotto installer script
# Usage: curl -sSL https://raw.githubusercontent.com/nishchaysinha/grotto/main/install.sh | sh

set -e

REPO="nishchaysinha/grotto"
INSTALL_DIR="${INSTALL_DIR:-$HOME/.local/bin}"
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

    cp "$binary_path" "${install_dir}/${BINARY_NAME}"
    chmod +x "${install_dir}/${BINARY_NAME}"
}

detect_shell() {
    if [ -n "$SHELL" ]; then
        basename "$SHELL"
    else
        ps -p "$$" -o comm= 2>/dev/null | sed 's/^-//'
    fi
}

get_shell_rc_file() {
    local shell_name="$1"
    case "$shell_name" in
        bash) echo "${HOME}/.bashrc" ;;
        zsh) echo "${HOME}/.zshrc" ;;
        fish) echo "${HOME}/.config/fish/config.fish" ;;
        ksh) echo "${HOME}/.kshrc" ;;
        *) echo "" ;;
    esac
}

add_path_to_shell_config() {
    local install_dir="$1"
    local shell_name="$2"
    local rc_file="$3"
    local path_line=""

    if [ "$shell_name" = "fish" ]; then
        path_line="set -gx PATH \$PATH ${install_dir}"
    else
        path_line="export PATH=\"\$PATH:${install_dir}\""
    fi

    mkdir -p "$(dirname "$rc_file")"
    touch "$rc_file"

    if grep -F "$install_dir" "$rc_file" > /dev/null 2>&1; then
        log_info "${install_dir} already configured in ${rc_file}"
        return
    fi

    {
        printf "\n"
        printf "# Added by grotto installer\n"
        printf "%s\n" "$path_line"
    } >> "$rc_file"
    log_info "Added ${install_dir} to PATH in ${rc_file}"
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

    if [ ! -d "$INSTALL_DIR" ]; then
        log_info "Creating install directory: ${INSTALL_DIR}"
        mkdir -p "$INSTALL_DIR" || {
            log_error "Unable to create ${INSTALL_DIR}. Set INSTALL_DIR to a writable location and try again."
            exit 1
        }
    fi

    install_binary "$binary_path" "$INSTALL_DIR" || {
        log_error "Unable to install to ${INSTALL_DIR}. Set INSTALL_DIR to a writable location and try again."
        exit 1
    }

    log_info "Successfully installed grotto to ${INSTALL_DIR}/${BINARY_NAME}"
    log_info "Run 'grotto --version' to verify the installation"

    # Check if install dir is in PATH
    case ":$PATH:" in
        *":${INSTALL_DIR}:"*) ;;
        *)
            local shell_name
            local rc_file
            shell_name=$(detect_shell)
            rc_file=$(get_shell_rc_file "$shell_name")

            log_warn "${INSTALL_DIR} is not in your PATH"

            if [ -n "$rc_file" ]; then
                add_path_to_shell_config "$INSTALL_DIR" "$shell_name" "$rc_file"
                log_info "Run this to use grotto in your current shell session:"
                if [ "$shell_name" = "fish" ]; then
                    log_info "  set -gx PATH \$PATH ${INSTALL_DIR}"
                else
                    log_info "  export PATH=\"\$PATH:${INSTALL_DIR}\""
                fi
            else
                log_warn "Detected shell '${shell_name:-unknown}' is not automatically supported."
                log_warn "Please run one of the following based on your shell:"
                log_warn "  export PATH=\"\$PATH:${INSTALL_DIR}\""
                log_warn "or add that line to your shell profile (e.g. ~/.bashrc or ~/.zshrc)."
            fi
            ;;
    esac
}

main "$@"
