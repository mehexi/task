#!/usr/bin/env bash
set -euo pipefail

REPO="mehexi/task"
BIN="oc-tasks"
INSTALL_DIR="${INSTALL_DIR:-/usr/local/bin}"

detect_arch() {
  arch=$(uname -m)
  case "$arch" in
    x86_64)  echo "amd64" ;;
    aarch64) echo "arm64" ;;
    arm64)   echo "arm64" ;;
    *)       echo "unsupported: $arch" >&2; exit 1 ;;
  esac
}

detect_os() {
  os=$(uname -s)
  case "$os" in
    Linux)  echo "linux" ;;
    Darwin) echo "darwin" ;;
    *)      echo "unsupported: $os" >&2; exit 1 ;;
  esac
}

OS=$(detect_os)
ARCH=$(detect_arch)
VERSION=$(curl -sL "https://api.github.com/repos/$REPO/releases/latest" | grep '"tag_name"' | cut -d'"' -f4)

if [ -z "$VERSION" ]; then
  echo "Failed to fetch latest version" >&2
  exit 1
fi

URL="https://github.com/$REPO/releases/download/$VERSION/${BIN}-${OS}-${ARCH}"
echo "Downloading $BIN $VERSION for $OS/$ARCH..."

TMPDIR=$(mktemp -d)
trap 'rm -rf "$TMPDIR"' EXIT

if command -v curl &>/dev/null; then
  curl -fsSL "$URL" -o "$TMPDIR/$BIN"
elif command -v wget &>/dev/null; then
  wget -q "$URL" -O "$TMPDIR/$BIN"
else
  echo "neither curl nor wget found" >&2
  exit 1
fi

chmod +x "$TMPDIR/$BIN"

if [ ! -d "$INSTALL_DIR" ]; then
  mkdir -p "$INSTALL_DIR"
fi

if [ -w "$INSTALL_DIR" ]; then
  mv "$TMPDIR/$BIN" "$INSTALL_DIR/$BIN"
else
  sudo mv "$TMPDIR/$BIN" "$INSTALL_DIR/$BIN"
fi

echo "Installed $BIN $VERSION to $INSTALL_DIR/$BIN"
