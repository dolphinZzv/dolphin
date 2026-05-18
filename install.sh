#!/bin/sh
set -eu

REPO="dolphinZzv/dolphin"
BIN="dolphin"

# ---- detect platform ----
case "$(uname -s)" in
  Linux)  OS="linux" ;;
  Darwin) OS="macOS" ;;
  MINGW*|MSYS*|CYGWIN*)
    echo "Windows detected — please download the .zip from https://github.com/$REPO/releases/latest"
    exit 1
    ;;
  *) echo "unsupported OS: $(uname -s)"; exit 1 ;;
esac

case "$(uname -m)" in
  x86_64|amd64) ARCH="x86_64" ;;
  aarch64|arm64) ARCH="arm64" ;;
  *) echo "unsupported arch: $(uname -m)"; exit 1 ;;
esac

# ---- fetch latest release ----
echo "Fetching latest release info..."
LATEST=$(curl -sSfL "https://api.github.com/repos/$REPO/releases/latest" 2>/dev/null \
  | sed -n 's/.*"tag_name": *"\(v[^"]*\)".*/\1/p') \
  || { echo "Failed to fetch latest version"; exit 1; }
[ -n "$LATEST" ] || { echo "Could not determine latest version"; exit 1; }
echo "Latest release: $LATEST"

# ---- download & install ----
ARCHIVE="$BIN_${LATEST}_${OS}_${ARCH}.tar.gz"
URL="https://github.com/$REPO/releases/download/${LATEST}/$ARCHIVE"
TMPDIR=$(mktemp -d)
trap 'rm -rf "$TMPDIR"' EXIT

echo "Downloading $URL ..."
curl -sSfL "$URL" -o "$TMPDIR/$ARCHIVE"

echo "Extracting..."
tar xzf "$TMPDIR/$ARCHIVE" -C "$TMPDIR"

# install
if [ "$(id -u)" -eq 0 ]; then
  DEST="/usr/local/bin"
else
  DEST="${HOME}/.local/bin"
  mkdir -p "$DEST"
fi
cp "$TMPDIR/$BIN" "$DEST/$BIN"
chmod +x "$DEST/$BIN"

echo "Installed $BIN to $DEST/$BIN"
echo "Run '$BIN setup' to get started."
