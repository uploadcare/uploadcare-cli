#!/bin/sh
# Uploadcare CLI installer
# Usage: curl -fsSL https://raw.githubusercontent.com/uploadcare/uploadcare-cli/main/scripts/install.sh | sh
#
# Environment variables:
#   VERSION     - specific version to install (e.g., "0.1.0"). Default: latest.
#   INSTALL_DIR - installation directory. Default: /usr/local/bin.
#   UNINSTALL   - set to "1" to uninstall. Example: curl ... | UNINSTALL=1 sh

set -eu

IS_TTY=false
if [ -t 1 ]; then
    IS_TTY=true
fi

if [ "$IS_TTY" = true ]; then
    trap 'printf "\033[?25h"' EXIT
fi

REPO="uploadcare/uploadcare-cli"
BINARY="uploadcare"
INSTALL_DIR="${INSTALL_DIR:-/usr/local/bin}"

# ── Colors & UI ─────────────────────────────────────────────────────
if [ "$IS_TTY" = true ]; then
    ESC=$(printf '\033')
    BOLD="${ESC}[1m"
    DIM="${ESC}[2m"
    ORANGE="${ESC}[38;2;255;118;0m"
    BAR_COLOR="${ESC}[38;2;255;205;0m"
    GREEN="${ESC}[0;32m"
    YELLOW="${ESC}[0;33m"
    RED="${ESC}[0;31m"
    WHITE="${ESC}[38;2;255;255;255m"
    GRAY="${ESC}[38;2;128;128;128m"
    SEPARATOR="${ESC}[38;2;68;68;68m"
    NC="${ESC}[0m"
    HIDE_CURSOR="${ESC}[?25l"
    SHOW_CURSOR="${ESC}[?25h"

    BAR_WIDTH=52
    _lines_below_bar=0

    progress_bar() {
        _pct=$1
        _filled=$(( _pct * BAR_WIDTH / 100 ))
        _bar=""
        _i=0
        while [ "$_i" -lt "$_filled" ]; do
            _bar="${_bar}■"
            _i=$((_i + 1))
        done
        while [ "$_i" -lt "$BAR_WIDTH" ]; do
            _bar="${_bar}･"
            _i=$((_i + 1))
        done
        if [ "$_lines_below_bar" -gt 0 ]; then
            printf "\033[${_lines_below_bar}A"
        fi
        printf "\r${WHITE}%s${NC}   ${BAR_COLOR}%3d%%${NC}" "$_bar" "$_pct"
        if [ "$_lines_below_bar" -gt 0 ]; then
            printf "\033[${_lines_below_bar}B\r"
        fi
    }

    _stage_done_text=""

    stage_start() {
        if [ "$_lines_below_bar" -eq 0 ]; then
            printf "\n"
            _lines_below_bar=1
        fi
        printf "\r\033[K${_stage_done_text}${GRAY}%s${NC}" "$1"
    }

    stage_done() {
        _stage_done_text="${_stage_done_text}${BAR_COLOR}✓${NC} ${WHITE}$1${NC}  "
        printf "\r\033[K${_stage_done_text}"
    }

    stage_warn() {
        _stage_done_text="${_stage_done_text}${YELLOW}⚠${NC} ${WHITE}$1${NC}  "
        printf "\r\033[K${_stage_done_text}"
    }

    print_logo() {
        printf "\n"
        printf "%s\n" "${SEPARATOR}-----------------------------------------------------------${NC}"
        printf "\n"
        printf "${BAR_COLOR} ▄████▄ ${NC}\n"
        printf "${BAR_COLOR}███▀▀███${NC}  ${WHITE}█  █ ████ █    █▀▀█ █▀▀█ █▀▀▄ █▀▀▀ █▀▀█ █▀▀█ ████${NC}\n"
        printf "${BAR_COLOR}███▄▄███${NC}  ${WHITE}█▄▄█ █    █▄▄▄ █▄▄█ █▀▀█ █▄▄█ █▄▄▄ █▀▀█ █▀▀▄ █▄▄▄${NC}\n"
        printf "${BAR_COLOR} ▀████▀ ${NC}\n"
        printf "\n"
    }
else
    BOLD="" DIM="" ORANGE="" BAR_COLOR="" GREEN="" YELLOW="" RED=""
    WHITE="" GRAY="" SEPARATOR="" NC="" HIDE_CURSOR="" SHOW_CURSOR=""

    progress_bar() { :; }
    stage_start() { echo "$1"; }
    stage_done()  { echo "$1"; }
    stage_warn()  { echo "Warning: $1"; }
    print_logo()  { :; }
fi

# ── Uninstall ──────────────────────────────────────────────────────
uninstall() {
    printf "\n"

    _bin=""
    if [ -f "$INSTALL_DIR/$BINARY" ]; then
        _bin="$INSTALL_DIR/$BINARY"
    elif command -v "$BINARY" >/dev/null 2>&1; then
        _bin="$(command -v "$BINARY")"
    fi

    if [ -z "$_bin" ]; then
        printf "${YELLOW}uploadcare is not installed.${NC}\n\n"
        exit 0
    fi

    _version=$("$_bin" version 2>/dev/null | head -1 || echo "unknown")
    printf "${GRAY}Found: %s (%s)${NC}\n" "$_bin" "$_version"

    if [ -w "$(dirname "$_bin")" ]; then
        rm -f "$_bin"
    else
        sudo rm -f "$_bin"
    fi

    printf "${GREEN}✓${NC} ${WHITE}Uninstalled uploadcare from %s${NC}\n\n" "$_bin"
    exit 0
}

# ── Main ────────────────────────────────────────────────────────────
main() {
    printf "\n"

    if [ "${UNINSTALL:-}" = "1" ]; then
        uninstall
    fi

    # Check existing installation
    if command -v "$BINARY" >/dev/null 2>&1; then
        _existing=$("$BINARY" version 2>/dev/null | head -1 || echo "unknown")
        printf "${SEPARATOR}Currently installed version: %s${NC}\n" "$_existing"
    fi

    detect_os
    detect_arch
    resolve_version
    check_sudo

    printf "\n${GRAY}Installing ${WHITE}uploadcare ${GRAY}version: ${WHITE}%s${NC}\n" "${VERSION#v}"
    printf "$HIDE_CURSOR"
    progress_bar 0

    download_and_verify
    install_binary
    progress_bar 100
    printf "$SHOW_CURSOR"
    printf "\n"

    print_logo
    print_success
}

# ── Detect OS ───────────────────────────────────────────────────────
detect_os() {
    OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
    case "$OS" in
        linux)  OS="linux" ;;
        darwin) OS="darwin" ;;
        *)
            printf "\n${RED}Error: unsupported operating system: %s${NC}\n" "$OS" >&2
            printf "${RED}Supported: linux, darwin (macOS)${NC}\n" >&2
            exit 1
            ;;
    esac
}

# ── Detect Architecture ────────────────────────────────────────────
detect_arch() {
    ARCH="$(uname -m)"
    case "$ARCH" in
        x86_64|amd64)       ARCH="amd64" ;;
        aarch64|arm64)       ARCH="arm64" ;;
        armv7l|armv7)        ARCH="armv7" ;;
        *)
            printf "\n${RED}Error: unsupported architecture: %s${NC}\n" "$ARCH" >&2
            printf "${RED}Supported: x86_64/amd64, aarch64/arm64, armv7${NC}\n" >&2
            exit 1
            ;;
    esac
}

# ── Resolve Version ────────────────────────────────────────────────
resolve_version() {
    if [ -n "${VERSION:-}" ]; then
        case "$VERSION" in
            v*) ;;
            *)  VERSION="v$VERSION" ;;
        esac
        return
    fi

    VERSION="$(curl -fsSI "https://github.com/$REPO/releases/latest" 2>/dev/null \
        | grep -i '^location:' \
        | sed 's|.*/tag/||' \
        | tr -d '[:space:]')"

    if [ -z "$VERSION" ]; then
        printf "\n${RED}Error: could not determine latest version.${NC}\n" >&2
        printf "${RED}Check https://github.com/%s/releases or set VERSION manually.${NC}\n" "$REPO" >&2
        exit 1
    fi

}

# ── Download & Verify ───────────────────────────────────────────────
download_and_verify() {
    VERSION_NUM="${VERSION#v}"
    ARCHIVE="uploadcare-cli_${VERSION_NUM}_${OS}_${ARCH}.tar.gz"
    DOWNLOAD_URL="https://github.com/$REPO/releases/download/${VERSION}/${ARCHIVE}"
    CHECKSUMS_URL="https://github.com/$REPO/releases/download/${VERSION}/checksums.txt"

    TMPDIR="$(mktemp -d)"
    if [ "$IS_TTY" = true ]; then
        trap 'rm -rf "$TMPDIR"; printf "\033[?25h"' EXIT
    else
        trap 'rm -rf "$TMPDIR"' EXIT
    fi

    stage_start "Downloading..."
    progress_bar 20

    # Download archive in background
    curl -fsSL "$DOWNLOAD_URL" -o "$TMPDIR/$ARCHIVE" &
    _dl_pid=$!

    # Download checksums in background
    curl -fsSL "$CHECKSUMS_URL" -o "$TMPDIR/checksums.txt" &
    _cs_pid=$!

    # Animate progress bar while downloading (TTY only)
    if [ "$IS_TTY" = true ]; then
        _pct=20
        while kill -0 "$_dl_pid" 2>/dev/null; do
            if [ "$_pct" -lt 65 ]; then
                _pct=$((_pct + 1))
            fi
            progress_bar "$_pct"
            sleep 0.1
        done
    fi

    # Wait for downloads to complete
    _dl_ok=0
    wait "$_dl_pid" || _dl_ok=1

    _cs_ok=0
    wait "$_cs_pid" || _cs_ok=1

    if [ "$_dl_ok" = "1" ]; then
        printf "$SHOW_CURSOR"
        printf "\n\n${RED}Error: failed to download %s${NC}\n" "$ARCHIVE" >&2
        exit 1
    fi

    if [ "$_cs_ok" = "1" ]; then
        printf "$SHOW_CURSOR"
        printf "\n\n${RED}Error: failed to download checksums${NC}\n" >&2
        exit 1
    fi

    progress_bar 70
    stage_done "Downloaded"

    stage_start "Verifying checksum..."

    # Verify checksum
    EXPECTED="$(grep "$ARCHIVE" "$TMPDIR/checksums.txt" | awk '{print $1}')"
    if [ -z "$EXPECTED" ]; then
        printf "$SHOW_CURSOR"
        printf "\n\n${RED}Error: archive not found in checksums.txt${NC}\n" >&2
        exit 1
    fi

    if command -v sha256sum >/dev/null 2>&1; then
        ACTUAL="$(sha256sum "$TMPDIR/$ARCHIVE" | awk '{print $1}')"
    elif command -v shasum >/dev/null 2>&1; then
        ACTUAL="$(shasum -a 256 "$TMPDIR/$ARCHIVE" | awk '{print $1}')"
    else
        progress_bar 80
        stage_warn "Checksum skipped ${YELLOW}(no sha256sum)${NC}"
        tar -xzf "$TMPDIR/$ARCHIVE" -C "$TMPDIR"
        progress_bar 85
        return
    fi

    if [ "$EXPECTED" != "$ACTUAL" ]; then
        progress_bar 80
        printf "$SHOW_CURSOR"
        printf "\n\n${RED}Error: checksum mismatch${NC}\n" >&2
        printf "${RED}  Expected: %s${NC}\n" "$EXPECTED" >&2
        printf "${RED}  Actual:   %s${NC}\n" "$ACTUAL" >&2
        exit 1
    fi

    progress_bar 80
    stage_done "Checksum verified"

    tar -xzf "$TMPDIR/$ARCHIVE" -C "$TMPDIR"
    progress_bar 85
}

# ── Install Binary ──────────────────────────────────────────────────
# ── Check Sudo ──────────────────────────────────────────────────────
check_sudo() {
    if [ -d "$INSTALL_DIR" ] && [ -w "$INSTALL_DIR" ]; then
        return
    fi
    if [ ! -d "$INSTALL_DIR" ] && mkdir -p "$INSTALL_DIR" 2>/dev/null; then
        return
    fi
    printf "${GRAY}Installing %s to %s requires elevated privileges.${NC}\n" "${VERSION#v}" "$INSTALL_DIR"
    sudo -v
}

# ── Install Binary ──────────────────────────────────────────────────
install_binary() {
    stage_start "Installing..."
    if [ ! -d "$INSTALL_DIR" ]; then
        if mkdir -p "$INSTALL_DIR" 2>/dev/null; then
            :
        else
            sudo mkdir -p "$INSTALL_DIR"
        fi
    fi

    progress_bar 90

    if [ -w "$INSTALL_DIR" ]; then
        cp "$TMPDIR/$BINARY" "$INSTALL_DIR/$BINARY"
        chmod +x "$INSTALL_DIR/$BINARY"
    else
        sudo cp "$TMPDIR/$BINARY" "$INSTALL_DIR/$BINARY"
        sudo chmod +x "$INSTALL_DIR/$BINARY"
    fi

    progress_bar 95
    stage_done "Installed"
}

# ── Success Message ─────────────────────────────────────────────────
print_success() {
    printf "${GRAY}Description:${NC}\n"
    printf "${WHITE}Official Uploadcare CLI for interacting with Uploadcare APIs${NC}\n"
    printf "\n"
    printf "${GRAY}Usage:${NC}\n"
    printf "${WHITE}uploadcare [command]${NC}\n"
    printf "\n"
    printf "${GRAY}Options:${NC}\n"
    printf "${WHITE}uploadcare --help                    ${GRAY} # Show all commands${NC}\n"
    printf "${WHITE}uploadcare file upload <${BAR_COLOR}photo.jpg${WHITE}>    ${GRAY}# Upload a file${NC}\n"
    printf "${WHITE}uploadcare file list                 ${GRAY} # Show files list${NC}\n"
    printf "${WHITE}uploadcare file info <${BAR_COLOR}uuid${WHITE}>           ${GRAY}# Get file details${NC}\n"
    printf "\n"
    printf "${GRAY}Set your credentials to start:${NC}\n"
    printf "${WHITE}export UPLOADCARE_PUBLIC_KEY=\"${BAR_COLOR}YOUR-PUBLIC-KEY${WHITE}\"${NC}\n"
    printf "${WHITE}export UPLOADCARE_SECRET_KEY=\"${BAR_COLOR}YOUR-SECRET-KEY${WHITE}\"${NC}\n"
    printf "\n"
    printf "%s\n" "${SEPARATOR}-----------------------------------------------------------${NC}"
    printf "\n"
    printf "${GRAY}Dashboard:  https://app.uploadcare.com${NC}\n"
    printf "${GRAY}Learn more: https://uploadcare.com/docs/cli${NC}\n"
    printf "\n"
}

main
