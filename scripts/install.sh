#!/bin/sh
# Uploadcare CLI installer
# Usage: curl -fsSL https://raw.githubusercontent.com/uploadcare/uploadcare-cli/main/scripts/install.sh | sh
#
# Environment variables:
#   VERSION     - specific version to install (e.g., "v0.1.0"). Default: latest.
#   INSTALL_DIR - installation directory. Default: /usr/local/bin.

set -eu

REPO="uploadcare/uploadcare-cli"
BINARY="uploadcare"
INSTALL_DIR="${INSTALL_DIR:-/usr/local/bin}"

main() {
    detect_os
    detect_arch
    resolve_version
    download_and_verify
    install_binary
    print_success
}

detect_os() {
    OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
    case "$OS" in
        linux)  OS="linux" ;;
        darwin) OS="darwin" ;;
        *)
            echo "Error: unsupported operating system: $OS" >&2
            echo "Supported: linux, darwin (macOS)" >&2
            exit 1
            ;;
    esac
}

detect_arch() {
    ARCH="$(uname -m)"
    case "$ARCH" in
        x86_64|amd64)       ARCH="amd64" ;;
        aarch64|arm64)       ARCH="arm64" ;;
        armv7l|armv7)        ARCH="armv7" ;;
        *)
            echo "Error: unsupported architecture: $ARCH" >&2
            echo "Supported: x86_64/amd64, aarch64/arm64, armv7" >&2
            exit 1
            ;;
    esac
}

resolve_version() {
    if [ -n "${VERSION:-}" ]; then
        # Ensure version starts with 'v'
        case "$VERSION" in
            v*) ;;
            *)  VERSION="v$VERSION" ;;
        esac
        return
    fi

    echo "Fetching latest version..."
    # Use the GitHub releases redirect to avoid needing jq
    VERSION="$(curl -fsSI "https://github.com/$REPO/releases/latest" 2>/dev/null \
        | grep -i '^location:' \
        | sed 's|.*/tag/||' \
        | tr -d '[:space:]')"

    if [ -z "$VERSION" ]; then
        echo "Error: could not determine latest version." >&2
        echo "Check https://github.com/$REPO/releases or set VERSION manually." >&2
        exit 1
    fi

    echo "Latest version: $VERSION"
}

download_and_verify() {
    VERSION_NUM="${VERSION#v}"
    ARCHIVE="uploadcare-cli_${VERSION_NUM}_${OS}_${ARCH}.tar.gz"
    DOWNLOAD_URL="https://github.com/$REPO/releases/download/${VERSION}/${ARCHIVE}"
    CHECKSUMS_URL="https://github.com/$REPO/releases/download/${VERSION}/checksums.txt"

    TMPDIR="$(mktemp -d)"
    trap 'rm -rf "$TMPDIR"' EXIT

    echo "Downloading ${ARCHIVE}..."
    curl -fsSL "$DOWNLOAD_URL" -o "$TMPDIR/$ARCHIVE"
    curl -fsSL "$CHECKSUMS_URL" -o "$TMPDIR/checksums.txt"

    echo "Verifying checksum..."
    EXPECTED="$(grep "$ARCHIVE" "$TMPDIR/checksums.txt" | awk '{print $1}')"
    if [ -z "$EXPECTED" ]; then
        echo "Error: archive not found in checksums.txt" >&2
        exit 1
    fi

    if command -v sha256sum >/dev/null 2>&1; then
        ACTUAL="$(sha256sum "$TMPDIR/$ARCHIVE" | awk '{print $1}')"
    elif command -v shasum >/dev/null 2>&1; then
        ACTUAL="$(shasum -a 256 "$TMPDIR/$ARCHIVE" | awk '{print $1}')"
    else
        echo "Warning: neither sha256sum nor shasum found, skipping checksum verification" >&2
        ACTUAL="$EXPECTED"
    fi

    if [ "$EXPECTED" != "$ACTUAL" ]; then
        echo "Error: checksum mismatch" >&2
        echo "  Expected: $EXPECTED" >&2
        echo "  Actual:   $ACTUAL" >&2
        exit 1
    fi

    echo "Checksum verified."

    tar -xzf "$TMPDIR/$ARCHIVE" -C "$TMPDIR"
}

install_binary() {
    if [ ! -d "$INSTALL_DIR" ]; then
        echo "Creating $INSTALL_DIR..."
        if mkdir -p "$INSTALL_DIR" 2>/dev/null; then
            :
        else
            sudo mkdir -p "$INSTALL_DIR"
        fi
    fi

    if [ -w "$INSTALL_DIR" ]; then
        cp "$TMPDIR/$BINARY" "$INSTALL_DIR/$BINARY"
        chmod +x "$INSTALL_DIR/$BINARY"
    else
        echo "Installing to $INSTALL_DIR (requires sudo)..."
        sudo cp "$TMPDIR/$BINARY" "$INSTALL_DIR/$BINARY"
        sudo chmod +x "$INSTALL_DIR/$BINARY"
    fi
}

print_success() {
    echo ""
    echo "Uploadcare CLI installed to $INSTALL_DIR/$BINARY"
    "$INSTALL_DIR/$BINARY" version
}

main
