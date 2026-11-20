#!/usr/bin/env bash
# Builds KanjiBar.app (a self-contained unsigned/ad-hoc menu bar app).
#
#   ./build.sh             build dist/KanjiBar.app
#   ./build.sh --install   build, then copy to ~/Applications and launch it
#
set -euo pipefail
cd "$(dirname "$0")"

APP_NAME="KanjiBar"
BUNDLE="dist/${APP_NAME}.app"

echo "==> go vet"
go vet ./...

echo "==> go test"
go test ./...

echo "==> go build"
rm -rf "$BUNDLE"
mkdir -p "$BUNDLE/Contents/MacOS" "$BUNDLE/Contents/Resources"
go build -trimpath -ldflags="-s -w" -o "$BUNDLE/Contents/MacOS/${APP_NAME}" .
cp Info.plist "$BUNDLE/Contents/Info.plist"

echo "==> ad-hoc sign"
codesign --force --sign - "$BUNDLE" >/dev/null 2>&1 || {
  echo "warning: ad-hoc codesign failed; the app may still run locally" >&2
}

echo "==> built $BUNDLE"
if [[ "${1:-}" == "--install" ]]; then
  mkdir -p "$HOME/Applications"
  rm -rf "$HOME/Applications/${APP_NAME}.app"
  cp -R "$BUNDLE" "$HOME/Applications/"
  echo "==> installed $HOME/Applications/${APP_NAME}.app"
  open "$HOME/Applications/${APP_NAME}.app"
fi
