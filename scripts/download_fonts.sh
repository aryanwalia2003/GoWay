#!/usr/bin/env bash
# scripts/download_fonts.sh
#
# Downloads Roboto Regular and Bold from Google Fonts and places them in the
# correct embed directory. Run once before `go build` or `make build`.
#
# Usage:
#   chmod +x scripts/download_fonts.sh
#   ./scripts/download_fonts.sh

set -euo pipefail

FONT_DIR="internal/assets/fonts"
REGULAR="${FONT_DIR}/Roboto-Regular.ttf"
BOLD="${FONT_DIR}/Roboto-Bold.ttf"

mkdir -p "${FONT_DIR}"

# Check if already present
if [[ -f "${REGULAR}" && -f "${BOLD}" ]]; then
  echo "✓ Fonts already present in ${FONT_DIR}/"
  exit 0
fi

echo "Downloading Roboto font family from Google Fonts ..."

TMP=$(mktemp -d)
trap 'rm -rf "${TMP}"' EXIT

ZIP="${TMP}/roboto.zip"

# Google Fonts direct download (stable URL, Apache 2.0 / OFL 1.1 licensed)
curl -fsSL \
  "https://fonts.google.com/download?family=Roboto" \
  -o "${ZIP}"

echo "Extracting fonts ..."
unzip -j "${ZIP}" \
  "*/Roboto-Regular.ttf" \
  "*/Roboto-Bold.ttf" \
  -d "${FONT_DIR}"

# Verify extraction
if [[ ! -f "${REGULAR}" ]]; then
  echo "ERROR: Roboto-Regular.ttf not found after extraction" >&2
  exit 1
fi
if [[ ! -f "${BOLD}" ]]; then
  echo "ERROR: Roboto-Bold.ttf not found after extraction" >&2
  exit 1
fi

REGULAR_KB=$(du -k "${REGULAR}" | cut -f1)
BOLD_KB=$(du -k "${BOLD}" | cut -f1)

echo "✓ Roboto-Regular.ttf (${REGULAR_KB} KB)"
echo "✓ Roboto-Bold.ttf    (${BOLD_KB} KB)"
echo ""
echo "Fonts ready. Run: make build"
