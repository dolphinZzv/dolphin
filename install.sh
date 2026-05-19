#!/bin/sh
set -eu

REPO="dolphinZzv/dolphin"
BIN="dolphin-ai"

# ---- dependency check ----
for cmd in curl tar uname; do
  command -v "$cmd" >/dev/null 2>&1 || { echo "Error: $cmd is required but not installed."; exit 1; }
done

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
VERSION="${LATEST#v}"  # strip v prefix — goreleaser archive names omit it
ARCHIVE="${BIN}_${VERSION}_${OS}_${ARCH}.tar.gz"
URL="https://github.com/$REPO/releases/download/${LATEST}/$ARCHIVE"
TMPDIR=$(mktemp -d)
trap 'rm -rf "$TMPDIR"' EXIT

echo "Downloading $URL ..."
curl -sSfL --retry 3 --retry-delay 2 "$URL" -o "$TMPDIR/$ARCHIVE"

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

if ! command -v "$BIN" >/dev/null 2>&1; then
  echo "  Warning: $DEST is not in your PATH."
  echo "  Add it: export PATH=\"\$PATH:$DEST\""
fi

echo "Run '$BIN setup' to get started."
