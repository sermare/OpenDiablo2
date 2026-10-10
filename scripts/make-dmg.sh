#!/bin/bash
# Build a drag-to-install disk image from dist/OpenDiablo2.app:
#   scripts/make-dmg.sh [path/to/OpenDiablo2.app]     -> dist/OpenDiablo2-<version>-macos-arm64.dmg
# Build the app first with scripts/make-app.sh. The image holds the app, an
# Applications shortcut and a short README; no game files. The app is ad-hoc
# signed, so on first launch use right-click > Open (see docs/macos-quickstart.md).
set -euo pipefail

cd "$(dirname "$0")/.."
APP=${1:-dist/OpenDiablo2.app}
[ -d "$APP" ] || { echo "build the app first: scripts/make-app.sh" >&2; exit 1; }

scripts/check_app_bundle.sh "$APP" >/dev/null || { echo "bundle check failed; run scripts/check_app_bundle.sh" >&2; exit 1; }

VERSION=$(/usr/libexec/PlistBuddy -c "Print :CFBundleShortVersionString" "$APP/Contents/Info.plist")
ARCH=$(lipo -archs "$APP/Contents/MacOS/OpenDiablo2" | tr ' ' '-')
OUT="$(dirname "$APP")/OpenDiablo2-$VERSION-macos-$ARCH.dmg"
VOLNAME=${VOLNAME:-OpenDiablo2}

STAGE=$(mktemp -d)
trap 'rm -rf "$STAGE"' EXIT
ditto "$APP" "$STAGE/OpenDiablo2.app"
ln -s /Applications "$STAGE/Applications"
cat >"$STAGE/READ ME FIRST.txt" <<'EOF'
OpenDiablo2 for Mac (Apple Silicon, macOS 14 or newer)

1. Drag OpenDiablo2 onto the Applications folder.
2. The first time, right-click (or Control-click) OpenDiablo2 in Applications and
   choose Open, then Open again. The app is not notarised, so a plain double-click
   is blocked once. If macOS still refuses: System Settings > Privacy & Security >
   "Open Anyway". If it says the app is damaged (a downloaded copy), open Terminal
   and run:  xattr -dr com.apple.quarantine /Applications/OpenDiablo2.app
3. Point it at your own Diablo II 1.14b + Lord of Destruction folder when asked
   (the folder with d2data.mpq, d2exp.mpq, patch_d2.mpq ...). No game files are
   included in this app.

Settings: ~/Library/Application Support/OpenDiablo2
(your saves and config.json live there; originals of your own .d2s files are never changed)
Log:      ~/Library/Logs/OpenDiablo2/OpenDiablo2.log
Full screen: Cmd+Enter. Quit: Cmd+Q.
EOF

if [ -n "${SOURCE_DATE_EPOCH:-}" ]; then
	STAMP=$(date -u -r "$SOURCE_DATE_EPOCH" +%Y%m%d%H%M.%S)
	find "$STAGE" -exec touch -h -t "$STAMP" {} +
fi

rm -f "$OUT"
hdiutil create -quiet -volname "$VOLNAME" -srcfolder "$STAGE" -fs HFS+ -format UDZO -ov "$OUT"
hdiutil verify -quiet "$OUT"
echo "built $OUT"
