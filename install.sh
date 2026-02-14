#!/bin/bash

set -euo pipefail

GITHUB_REPO="codag-megalith/codag-cli"
INSTALL_DIR="$HOME/.local/bin"
BINARY="codag"

# Colors (disabled in non-interactive mode)
if [[ -t 1 ]]; then
    RED='\033[0;31m'
    GREEN='\033[0;32m'
    YELLOW='\033[0;33m'
    BLUE='\033[0;34m'
    BOLD='\033[1m'
    NC='\033[0m'
else
    RED=''
    GREEN=''
    YELLOW=''
    BLUE=''
    BOLD=''
    NC=''
fi

info() {
    printf '%b%s%b\n' "${BLUE}==>${NC} ${BOLD}" "$1" "${NC}"
}

success() {
    printf '%b%s%b\n' "${GREEN}==>${NC} ${BOLD}" "$1" "${NC}"
}

warn() {
    printf '%b %s\n' "${YELLOW}Warning:${NC}" "$1"
}

error() {
    printf '%b %s\n' "${RED}Error:${NC}" "$1" >&2
    exit 1
}

detect_os() {
    local os
    os="$(uname -s | tr '[:upper:]' '[:lower:]')"
    case "$os" in
        darwin) echo "darwin" ;;
        linux)  echo "linux" ;;
        *)      error "Unsupported operating system: $os" ;;
    esac
}

detect_arch() {
    local arch
    arch="$(uname -m)"
    case "$arch" in
        x86_64|amd64)   echo "amd64" ;;
        arm64|aarch64)  echo "arm64" ;;
        *)              error "Unsupported architecture: $arch" ;;
    esac
}

get_latest_version() {
    local url="https://api.github.com/repos/${GITHUB_REPO}/releases/latest"
    local version
    local curl_opts=(-fsSL)

    # Support authenticated requests for rate limiting
    if [[ -n "${GITHUB_TOKEN:-}" ]]; then
        curl_opts+=(-H "Authorization: Bearer ${GITHUB_TOKEN}")
    fi

    version=$(curl "${curl_opts[@]}" "$url" 2>/dev/null | grep '"tag_name"' | sed -E 's/.*"tag_name": *"v?([^"]+)".*/\1/')

    if [[ -z "$version" ]]; then
        error "Failed to fetch latest version from GitHub. Please check your internet connection."
    fi

    echo "$version"
}

download_file() {
    local url="$1"
    local output="$2"

    if ! curl -fsSL "$url" -o "$output"; then
        error "Failed to download: ${url}"
    fi
}

verify_checksum() {
    local file="$1"
    local expected="$2"
    local actual

    if command -v sha256sum &> /dev/null; then
        actual=$(sha256sum "$file" | awk '{print $1}')
    elif command -v shasum &> /dev/null; then
        actual=$(shasum -a 256 "$file" | awk '{print $1}')
    else
        warn "No checksum tool found (sha256sum or shasum). Skipping verification."
        return 0
    fi

    if [[ "$actual" != "$expected" ]]; then
        error "Checksum verification failed!\n  Expected: ${expected}\n  Actual:   ${actual}"
    fi
}

main() {
    if ! command -v curl &> /dev/null; then
        error "curl is required but not installed. Please install curl and try again."
    fi

    info "Installing Codag CLI..."

    # Detect platform
    local os arch
    os=$(detect_os)
    arch=$(detect_arch)
    info "Detected platform: ${os}/${arch}"

    # Fetch latest version
    info "Fetching latest version..."
    local version
    version=$(get_latest_version)
    version="${version#v}"
    info "Installing version: ${version}"

    # Construct download URLs
    local archive_name="codag_${os}_${arch}.tar.gz"
    local download_url="https://github.com/${GITHUB_REPO}/releases/download/v${version}/${archive_name}"
    local checksums_url="https://github.com/${GITHUB_REPO}/releases/download/v${version}/checksums.txt"

    # Create temp directory
    local tmp_dir
    tmp_dir=$(mktemp -d)
    trap 'rm -rf "$tmp_dir"' EXIT

    # Download archive
    info "Downloading ${archive_name}..."
    local archive_path="${tmp_dir}/${archive_name}"
    download_file "$download_url" "$archive_path"

    # Download and verify checksums
    info "Verifying checksum..."
    local checksums_path="${tmp_dir}/checksums.txt"
    download_file "$checksums_url" "$checksums_path"

    local expected_checksum
    expected_checksum=$(grep -iE "${archive_name}\$" "$checksums_path" | awk '{print $1}' || true)
    if [[ -z "$expected_checksum" ]]; then
        error "Checksum for ${archive_name} not found in checksums.txt"
    fi
    verify_checksum "$archive_path" "$expected_checksum"
    success "Checksum verified"

    # Extract
    info "Extracting..."
    tar -xzf "$archive_path" -C "$tmp_dir"

    local binary_path="${tmp_dir}/${BINARY}"
    chmod +x "$binary_path"

    # Install
    info "Installing to ${INSTALL_DIR}..."
    mkdir -p "$INSTALL_DIR"

    if [[ ! -w "$INSTALL_DIR" ]]; then
        error "Cannot write to ${INSTALL_DIR}. Please check permissions."
    fi

    mv "$binary_path" "${INSTALL_DIR}/${BINARY}"

    # Verify the binary runs
    if "${INSTALL_DIR}/${BINARY}" version &> /dev/null; then
        success "Codag CLI v${version} installed to ${INSTALL_DIR}/${BINARY}"
    else
        error "Installation completed but the binary failed to execute."
    fi

    # Check PATH
    local path_binary
    path_binary=$(command -v "${BINARY}" 2>/dev/null || true)

    if [[ -n "$path_binary" && "$path_binary" != "${INSTALL_DIR}/${BINARY}" ]]; then
        # Another codag binary exists on PATH
        echo ""
        echo -e "  ${YELLOW}WARNING: PATH conflict detected${NC}"
        echo ""
        echo -e "  Installed to: ${INSTALL_DIR}/${BINARY}"
        echo -e "  But '${BINARY}' resolves to: ${path_binary}"
        echo ""
        echo -e "  Either remove the old binary or adjust your PATH to prioritize ${INSTALL_DIR}"
        echo ""

    elif [[ -z "$path_binary" ]]; then
        # Not on PATH yet — show shell-specific instructions
        local shell_name shell_config
        shell_name="$(basename "${SHELL:-}")"

        case "$shell_name" in
            zsh)
                shell_config="~/.zshrc" ;;
            bash)
                if [[ -f "$HOME/.bash_profile" ]]; then
                    shell_config="~/.bash_profile"
                else
                    shell_config="~/.bashrc"
                fi
                ;;
            fish)
                shell_config="~/.config/fish/config.fish" ;;
            *)
                shell_config="" ;;
        esac

        echo ""
        echo -e "  ${YELLOW}Almost there!${NC} Add Codag to your PATH:"
        echo ""

        if [[ "$shell_name" == "fish" ]]; then
            echo -e "    ${BOLD}fish_add_path ${INSTALL_DIR}${NC}"
            echo ""
            echo -e "  Add to ${shell_config} to make it permanent."
        elif [[ -n "$shell_config" ]]; then
            echo -e "  Run this, then restart your terminal:"
            echo ""
            echo -e "    ${BOLD}echo 'export PATH=\"${INSTALL_DIR}:\$PATH\"' >> ${shell_config}${NC}"
        else
            echo -e "  Add this to your shell config:"
            echo ""
            echo -e "    ${BOLD}export PATH=\"${INSTALL_DIR}:\$PATH\"${NC}"
        fi

        echo ""
    fi

    # Get started
    echo -e "  Get started:"
    echo ""
    echo -e "    ${BOLD}codag login${NC}"
    echo -e "    ${BOLD}codag init${NC}"
    echo ""
}

main "$@"
