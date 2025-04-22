#!/bin/bash
set -euo pipefail

BASEREPONAME=krang
REPO="dougbtv/$BASEREPONAME"
BINARY="krangctl"
INSTALL_DIR="/usr/local/bin"
VERSION="v0.0.2"

ARCH=$(uname -m)
OS=$(uname | tr '[:upper:]' '[:lower:]')

echo "🍕 Ordering pizza..."

# Normalize arch
case "$ARCH" in
  x86_64) ARCH="amd64" ;;
  aarch64 | arm64) ARCH="arm64" ;;
  *) echo "❌ Sorry, we don't have a build for this architecture yet (PRs welcome!): $ARCH" && exit 1 ;;
esac

# Check write access to INSTALL_DIR
if [ ! -w "$INSTALL_DIR" ]; then
  echo "🚫 Cannot write to ${INSTALL_DIR}."
  echo "👉 Try running this script with sudo, if you roll like that:"
  echo ""
  echo "   curl -sSfL https://raw.githubusercontent.com/$REPO/main/getkrang.sh | sudo bash"
  echo ""
  exit 1
fi

# Grab latest release
if [[ -z "$VERSION" ]]; then
  echo "🔍 Fetching latest version from GitHub..."
  # LATEST=$(curl -s $AUTH_HEADER "https://api.github.com/repos/${REPO}/releases/latest" | grep tag_name | cut -d '"' -f 4)
  LATEST=$(curl -s https://api.github.com/repos/$REPO/releases/latest | grep tag_name | cut -d '"' -f 4)
  if [[ -z "$LATEST" ]]; then
    echo "❌ Failed to fetch latest release"
    exit 1
  fi
else
  LATEST="$VERSION"
fi

if [[ -z "$LATEST" ]]; then
  echo "❌ Failed to fetch latest release"
  exit 1
fi

# LATEST comes as v0.0.2, but the archive name is 0.0.2
if [[ "$LATEST" =~ ^v ]]; then
  BASEVERSION="${LATEST:1}"
else
  BASEVERSION="$LATEST"
fi

# Construct URL for .tar.gz asset
ARCHIVE_NAME="${BASEREPONAME}_${BASEVERSION}_${OS}_${ARCH}.tar.gz"
URL="https://github.com/${REPO}/releases/download/${LATEST}/${ARCHIVE_NAME}"

TMPDIR=$(mktemp -d)
TARBALL="${TMPDIR}/${ARCHIVE_NAME}"

echo "📦 Downloading ${BINARY} ${LATEST} for ${OS}/${ARCH}..."
if ! curl -sSfL "$URL" -o "$TARBALL"; then
  echo "❌ Download failed! Tried to get:"
  echo "   $URL"
  exit 1
fi

echo "📂 Extracting archive..."
tar -xzf "$TARBALL" -C "$TMPDIR"

if [ ! -f "${TMPDIR}/${BINARY}" ]; then
  echo "❌ Binary not found in extracted archive!"
  exit 1
fi

chmod +x "${TMPDIR}/${BINARY}"
mv "${TMPDIR}/${BINARY}" "${INSTALL_DIR}/${BINARY}"

echo "✅ Installed ${BINARY} to ${INSTALL_DIR}/${BINARY}"

"${INSTALL_DIR}/${BINARY}" --help || echo "🚨 Installed, but failed to run --help. Something might be off."

echo "🧠 krangctl installed!"
