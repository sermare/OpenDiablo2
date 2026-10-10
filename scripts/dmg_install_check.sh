#!/bin/bash
# Rehearse "download the dmg, open it, drag the app to Applications" without a GUI and
# without starting the game. Static: nothing of the app runs.
#   scripts/dmg_install_check.sh [path/to/OpenDiablo2-<version>-macos-arm64.dmg]
# Steps: copy the dmg and mark it quarantined like a browser download, mount it read-only
# (hidden from Finder), ditto the app into a temporary "Applications" folder, mark the copy
# quarantined, and check that it still verifies, that the executable is arm64 and executable,
# and that the first-run strings are present. Your real /Applications and Desktop are never
# touched. A human still has to open the image in Finder once (docs/macos-quickstart.md).
set -u
cd "$(dirname "$0")/.." || exit 2
DMG=${1:-$(ls dist/*.dmg 2>/dev/null | head -1)}
[ -f "${DMG:-}" ] || { echo "no dmg: run scripts/release.sh or scripts/make-dmg.sh first" >&2; exit 2; }

fail=0
ok() { echo "ok:   $1"; }
bad() { echo "FAIL: $1"; fail=1; }

TMP=$(mktemp -d "${TMPDIR:-/tmp}/od2-dmg-install.XXXXXX")
MNT="$TMP/mnt"
mkdir -p "$MNT" "$TMP/Applications"
cleanup() {
	hdiutil detach -quiet "$MNT" >/dev/null 2>&1 || hdiutil detach -quiet -force "$MNT" >/dev/null 2>&1
	rm -rf "$TMP"
}
trap cleanup EXIT

COPY="$TMP/$(basename "$DMG")"
cp "$DMG" "$COPY"
xattr -w com.apple.quarantine "0081;00000000;Safari;" "$COPY" 2>/dev/null && ok "dmg copy marked quarantined (as a download)"

if ! hdiutil attach -quiet -readonly -nobrowse -mountpoint "$MNT" "$COPY" >/dev/null 2>&1; then
	bad "could not mount the quarantined copy"
	echo "DMG INSTALL CHECK FAILED"
	exit 1
fi
ok "mounted read-only at a private mount point"

[ -d "$MNT/OpenDiablo2.app" ] || { bad "no OpenDiablo2.app on the image"; echo "DMG INSTALL CHECK FAILED"; exit 1; }
ditto "$MNT/OpenDiablo2.app" "$TMP/Applications/OpenDiablo2.app" && ok "dragged the app to a temporary Applications folder" || bad "copy failed"
APP="$TMP/Applications/OpenDiablo2.app"
xattr -w com.apple.quarantine "0081;00000000;Safari;" "$APP" 2>/dev/null
xattr -r -w com.apple.quarantine "0081;00000000;Safari;" "$APP" 2>/dev/null

codesign --verify --deep --strict "$APP" 2>/dev/null && ok "installed copy verifies (codesign --deep --strict)" || bad "installed copy fails codesign"
BIN="$APP/Contents/MacOS/OpenDiablo2"
[ -x "$BIN" ] && ok "executable bit survived the copy" || bad "binary is not executable after the copy"
[ "$(lipo -archs "$BIN" 2>/dev/null)" = arm64 ] && ok "arm64 binary" || bad "binary is not arm64-only"
/usr/libexec/PlistBuddy -c "Print :CFBundleExecutable" "$APP/Contents/Info.plist" 2>/dev/null | grep -qx OpenDiablo2 && ok "Info.plist names the executable" || bad "Info.plist CFBundleExecutable"
minos=$(/usr/libexec/PlistBuddy -c "Print :LSMinimumSystemVersion" "$APP/Contents/Info.plist" 2>/dev/null)
[ -n "$minos" ] && ok "minimum macOS $minos" || bad "no LSMinimumSystemVersion"

# first-run wiring that must work with an empty profile (no game files, no env vars)
STR=$(strings -a "$BIN" 2>/dev/null)
grep -q "Application Support" <<<"$STR" && ok "config dir defaults to Application Support" || bad "no Application Support default"
grep -q "choose folder" <<<"$STR" && ok "first-run folder picker present" || bad "no first-run folder picker"
[ -f "$MNT/READ ME FIRST.txt" ] && grep -q "Open" "$MNT/READ ME FIRST.txt" && ok "READ ME FIRST explains right-click > Open" || bad "READ ME FIRST missing the Open hint"

[ "$fail" = 0 ] && echo "DMG INSTALL OK (a human still has to open it in Finder once)" || echo "DMG INSTALL CHECK FAILED"
exit $fail
