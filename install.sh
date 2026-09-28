#!/bin/sh
set -e

REPO="subizu/proviq"
BINARY="agent-proof"
INSTALL_DIR="${INSTALL_DIR:-/usr/local/bin}"

OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
ARCH="$(uname -m)"

case "$ARCH" in
  x86_64) ARCH="amd64" ;;
  aarch64|arm64) ARCH="arm64" ;;
  *) echo "Unsupported architecture: $ARCH"; exit 1 ;;
esac

case "$OS" in
  linux|darwin) ;;
  *) echo "Unsupported OS: $OS"; exit 1 ;;
esac

LATEST_TAG=$(curl -sSL "https://api.github.com/repos/${REPO}/releases/latest" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')
if [ -z "$LATEST_TAG" ]; then
  LATEST_TAG="v0.1.0"
fi
VERSION="${LATEST_TAG#v}"

ARCHIVE_NAME="${BINARY}_${VERSION}_${OS}_${ARCH}.tar.gz"
DOWNLOAD_URL="https://github.com/${REPO}/releases/download/${LATEST_TAG}/${ARCHIVE_NAME}"

echo "Downloading ${BINARY} ${LATEST_TAG} for ${OS}/${ARCH}..."
TMP_DIR=$(mktemp -d)
trap 'rm -rf "$TMP_DIR"' EXIT

if curl -sSL -f "$DOWNLOAD_URL" -o "$TMP_DIR/$ARCHIVE_NAME"; then
  tar -xzf "$TMP_DIR/$ARCHIVE_NAME" -C "$TMP_DIR"
  if [ -w "$INSTALL_DIR" ]; then
    mv "$TMP_DIR/$BINARY" "$INSTALL_DIR/$BINARY"
  else
    echo "Installing to $INSTALL_DIR requires sudo privileges:"
    sudo mv "$TMP_DIR/$BINARY" "$INSTALL_DIR/$BINARY"
  fi
  chmod +x "$INSTALL_DIR/$BINARY"
  echo "✓ Successfully installed $BINARY to $INSTALL_DIR/$BINARY"
  echo "Run '$BINARY --help' to get started!"
else
  echo "Pre-built binary not found. Falling back to 'go install'..."
  if command -v go >/dev/null 2>&1; then
    go install "github.com/${REPO}/cmd/${BINARY}@latest"
    echo "✓ Successfully installed via Go!"
  else
    echo "Error: Failed to download binary and 'go' is not installed."
    exit 1
  fi
fi
