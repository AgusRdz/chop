#!/bin/sh
set -e

REPO="AgusRdz/chop"
INSTALL_DIR="${CHOP_INSTALL_DIR:-$HOME/bin}"

# Detect OS
OS="$(uname -s)"
case "$OS" in
  Linux*)  OS="linux" ;;
  Darwin*) OS="darwin" ;;
  MINGW*|MSYS*|CYGWIN*) OS="windows" ;;
  *) echo "unsupported OS: $OS" >&2; exit 1 ;;
esac

# Detect architecture
ARCH="$(uname -m)"
case "$ARCH" in
  x86_64|amd64) ARCH="amd64" ;;
  aarch64|arm64) ARCH="arm64" ;;
  *) echo "unsupported architecture: $ARCH" >&2; exit 1 ;;
esac

EXT=""
if [ "$OS" = "windows" ]; then
  EXT=".exe"
fi

BINARY="chop-${OS}-${ARCH}${EXT}"

# Get latest version
if [ -z "$CHOP_VERSION" ]; then
  CHOP_VERSION=$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" | grep '"tag_name"' | sed 's/.*"tag_name": *"//;s/".*//')
fi

if [ -z "$CHOP_VERSION" ]; then
  echo "failed to determine latest version" >&2
  exit 1
fi

URL="https://github.com/${REPO}/releases/download/${CHOP_VERSION}/${BINARY}"

echo "installing chop ${CHOP_VERSION} (${OS}/${ARCH})..."

mkdir -p "$INSTALL_DIR"
curl -fsSL "$URL" -o "${INSTALL_DIR}/chop${EXT}"
chmod +x "${INSTALL_DIR}/chop${EXT}"

echo "installed chop to ${INSTALL_DIR}/chop${EXT}"
echo ""

# Check if install dir is in PATH
case ":$PATH:" in
  *":${INSTALL_DIR}:"*) ;;
  *)
    echo "NOTE: ${INSTALL_DIR} is not in your PATH."
    echo "Add it with:"
    echo "  export PATH=\"${INSTALL_DIR}:\$PATH\""
    echo ""
    ;;
esac

echo "Next steps:"
echo ""
echo "  # Use directly with any command:"
echo "  chop git status"
echo "  chop docker ps"
echo ""
echo "  # Shell integration (auto-wrap all supported commands):"
echo "  echo 'eval \"\$(chop init bash)\"' >> ~/.bashrc"
echo "  echo 'eval \"\$(chop init zsh)\"'  >> ~/.zshrc"
echo ""
echo "  # Claude Code hook (auto-rewrite Bash tool calls):"
echo "  chop init --global"
