#!/bin/sh
set -e

REPO="jeroenjanssens/velocirepo"
INSTALL_DIR="${INSTALL_DIR:-./bin}"

detect_os() {
    case "$(uname -s)" in
        Linux*)  echo "linux" ;;
        Darwin*) echo "darwin" ;;
        *)       echo "unsupported"; return 1 ;;
    esac
}

detect_arch() {
    case "$(uname -m)" in
        x86_64|amd64)  echo "amd64" ;;
        arm64|aarch64) echo "arm64" ;;
        *)             echo "unsupported"; return 1 ;;
    esac
}

get_latest_version() {
    curl -sSfL -o /dev/null -w '%{url_effective}' "https://github.com/${REPO}/releases/latest" |
        grep -o '[^/]*$' |
        sed 's/^v//'
}

OS=$(detect_os)
ARCH=$(detect_arch)

if [ "$OS" = "unsupported" ] || [ "$ARCH" = "unsupported" ]; then
    echo "Error: unsupported platform $(uname -s)/$(uname -m)" >&2
    echo "See https://github.com/${REPO}/releases for available binaries." >&2
    exit 1
fi

if [ -n "$VERSION" ]; then
    VERSION="${VERSION#v}"
else
    echo "Fetching latest version..." >&2
    VERSION=$(get_latest_version)
fi

if [ -z "$VERSION" ]; then
    echo "Error: could not determine latest version" >&2
    exit 1
fi

URL="https://github.com/${REPO}/releases/download/v${VERSION}/velocirepo_${VERSION}_${OS}_${ARCH}.tar.gz"

echo "Installing velocirepo v${VERSION} (${OS}/${ARCH})..." >&2
echo "  from: ${URL}" >&2

mkdir -p "$INSTALL_DIR"

if ! curl -sSfL "$URL" | tar xz -C "$INSTALL_DIR" velocirepo; then
    echo "Error: download failed. Check that v${VERSION} exists for ${OS}/${ARCH}." >&2
    echo "  ${URL}" >&2
    exit 1
fi

chmod +x "${INSTALL_DIR}/velocirepo"

echo "Installed: ${INSTALL_DIR}/velocirepo" >&2
"${INSTALL_DIR}/velocirepo" version
