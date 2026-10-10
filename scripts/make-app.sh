#!/bin/bash
# Build dist/OpenDiablo2.app: a double-clickable macOS app for this engine.
#
#   scripts/make-app.sh              native build (arm64 on Apple Silicon)
#   UNIVERSAL=1 scripts/make-app.sh  also try an x86_64 slice and lipo them
#                                    (needs the Xcode command line tools; the
#                                    engine needs CGO, so cross builds are best effort)
#   VERSION=1.2.3 scripts/make-app.sh  override the version in Info.plist
#                                    (default: the number from `git describe --tags`, e.g.
#                                    v0.1-pre-alpha-12-gabc -> 0.1.0; the full describe string
#                                    is kept in the OD2GitDescribe key)
#   SOURCE_DATE_EPOCH=<unix time>    stamp all bundle files with that time (scripts/release.sh
#                                    sets it from the commit) so the zip/dmg are repeatable
#   INSTALL=1 scripts/make-app.sh    also copy the app to /Applications
#   ZIP=1 scripts/make-app.sh        also write dist/OpenDiablo2-<version>-macos-arm64.zip
#                                    (ditto, keeps the signature and bundle metadata)
#
# Info.plist: arm64 only, macOS 14+, Retina (NSHighResolutionCapable); no document
# types (saves are imported from the Diablo II folder, not opened from Finder) and
# no LSEnvironment (real maps are the engine default, so the app needs no OD2_* variable).
# The app is ad-hoc signed (codesign -s -). No game files are included: on first
# launch it looks for your Diablo II folder or asks you to pick it.
set -euo pipefail

cd "$(dirname "$0")/.."
ROOT=$(pwd)
DIST="$ROOT/dist"
APP="$DIST/OpenDiablo2.app"
EXE=OpenDiablo2
BUNDLE_ID=io.github.sermare.OpenDiablo2

BRANCH=$(git rev-parse --abbrev-ref HEAD 2>/dev/null || echo local)
COMMIT=$(git rev-parse --short HEAD 2>/dev/null || echo build)
DESCRIBE=$(git describe --tags --always --dirty --match 'v[0-9]*' 2>/dev/null || echo "$COMMIT")
if [ -z "${VERSION:-}" ]; then
	# CFBundleShortVersionString must be 1-3 dot-separated integers: take them from the tag, pad to 3
	VERSION=$(printf '%s' "$DESCRIBE" | sed -nE 's/^v([0-9]+(\.[0-9]+){0,2}).*/\1/p')
	VERSION=${VERSION:-0.1.0}
	while [ "$(printf '%s' "$VERSION" | tr -cd . | wc -c | tr -d ' ')" -lt 2 ]; do VERSION=$VERSION.0; done
fi
case "$VERSION" in
	*[!0-9.]* | .* | *. | *..*) echo "bad VERSION '$VERSION': want digits and dots, e.g. 1.2.3" >&2; exit 1 ;;
esac
BUILD=$(git rev-list --count HEAD 2>/dev/null || echo 1)
LDFLAGS="-X main.GitBranch=$BRANCH -X main.GitCommit=$COMMIT"

for tool in go sips iconutil codesign plutil; do
	command -v "$tool" >/dev/null || { echo "missing tool: $tool" >&2; exit 1; }
done

rm -rf "$APP"
mkdir -p "$APP/Contents/MacOS" "$APP/Contents/Resources"
WORK=$(mktemp -d)
trap 'rm -rf "$WORK"' EXIT

echo "==> building $EXE (arm64 native)"
CGO_ENABLED=1 go build -trimpath -buildvcs=false -ldflags "$LDFLAGS" -o "$WORK/$EXE-arm64" . 2>&1 | grep -v "ignoring duplicate libraries" || true
[ -x "$WORK/$EXE-arm64" ] || { echo "build failed" >&2; exit 1; }
BIN="$WORK/$EXE-arm64"

if [ "${UNIVERSAL:-0}" = 1 ]; then
	echo "==> trying x86_64 slice"
	if CGO_ENABLED=1 GOARCH=amd64 CC="clang -arch x86_64" CXX="clang++ -arch x86_64" \
		go build -trimpath -buildvcs=false -ldflags "$LDFLAGS" -o "$WORK/$EXE-amd64" . 2>"$WORK/amd64.log"; then
		lipo -create "$WORK/$EXE-arm64" "$WORK/$EXE-amd64" -output "$WORK/$EXE-universal"
		BIN="$WORK/$EXE-universal"
	else
		echo "   x86_64 build failed (continuing with arm64 only); see:" >&2
		grep -v "ignoring duplicate" "$WORK/amd64.log" | head -5 >&2
	fi
fi
cp "$BIN" "$APP/Contents/MacOS/$EXE"

echo "==> icon (original artwork drawn by scripts/icon/main.go; no Blizzard art)"
go run scripts/icon/main.go "$WORK/square.png" 1024
ICONSET="$WORK/OpenDiablo2.iconset"
mkdir -p "$ICONSET"
for s in 16 32 128 256 512; do
	sips -z $s $s "$WORK/square.png" --out "$ICONSET/icon_${s}x${s}.png" >/dev/null
	sips -z $((s * 2)) $((s * 2)) "$WORK/square.png" --out "$ICONSET/icon_${s}x${s}@2x.png" >/dev/null
done
iconutil -c icns "$ICONSET" -o "$APP/Contents/Resources/OpenDiablo2.icns"
sips -z 256 256 "$WORK/square.png" --out "$APP/Contents/Resources/d2logo.png" >/dev/null # the engine loads it as the window icon

cat >"$APP/Contents/Info.plist" <<EOF
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>CFBundleDevelopmentRegion</key><string>en</string>
	<key>CFBundleExecutable</key><string>$EXE</string>
	<key>CFBundleIconFile</key><string>OpenDiablo2</string>
	<key>CFBundleIdentifier</key><string>$BUNDLE_ID</string>
	<key>CFBundleInfoDictionaryVersion</key><string>6.0</string>
	<key>CFBundleName</key><string>OpenDiablo2</string>
	<key>CFBundleDisplayName</key><string>OpenDiablo2</string>
	<key>CFBundlePackageType</key><string>APPL</string>
	<key>CFBundleShortVersionString</key><string>$VERSION</string>
	<key>CFBundleVersion</key><string>$BUILD</string>
	<key>OD2GitDescribe</key><string>$DESCRIBE</string>
	<key>LSApplicationCategoryType</key><string>public.app-category.games</string>
	<key>LSMinimumSystemVersion</key><string>14.0</string>
	<key>LSArchitecturePriority</key><array><string>arm64</string></array>
	<key>NSSupportsAutomaticGraphicsSwitching</key><true/>
	<key>NSHighResolutionCapable</key><true/>
	<key>NSPrincipalClass</key><string>NSApplication</string>
	<key>NSHumanReadableCopyright</key><string>OpenDiablo2 is free software (GPL). Diablo II is a trademark of Blizzard Entertainment; game files are not included.</string>
</dict>
</plist>
EOF
plutil -lint "$APP/Contents/Info.plist" >/dev/null

echo "==> ad-hoc signing"
codesign --force --deep -s - "$APP"
codesign --verify --deep --strict "$APP"

if [ -n "${SOURCE_DATE_EPOCH:-}" ]; then
	# file times are not part of the signature; fix them so archives of the bundle repeat
	STAMP=$(date -u -r "$SOURCE_DATE_EPOCH" +%Y%m%d%H%M.%S)
	find "$APP" -exec touch -h -t "$STAMP" {} +
fi

echo "built $APP ($(lipo -archs "$APP/Contents/MacOS/$EXE"))"

if [ "${ZIP:-0}" = 1 ]; then
	ARCH=$(lipo -archs "$APP/Contents/MacOS/$EXE" | tr ' ' '-')
	ZIPFILE="$DIST/OpenDiablo2-$VERSION-macos-$ARCH.zip"
	# the bundle must contain no game data: refuse to package anything that looks like it
	if find "$APP" \( -iname '*.mpq' -o -iname '*.d2s' -o -iname '*.dc6' -o -iname '*.dt1' -o -iname '*.ds1' \) | grep -q .; then
		echo "game files found inside the bundle; refusing to zip" >&2; exit 1
	fi
	rm -f "$ZIPFILE"
	ditto -c -k --keepParent "$APP" "$ZIPFILE"
	echo "zipped $ZIPFILE"
fi

if [ "${INSTALL:-0}" = 1 ]; then
	rm -rf /Applications/OpenDiablo2.app
	cp -R "$APP" /Applications/
	echo "installed /Applications/OpenDiablo2.app"
fi

echo "Double-click it, or: open \"$APP\""
echo "First launch: right-click > Open if macOS warns about an unidentified developer"
echo "(downloaded copy reported as damaged: xattr -dr com.apple.quarantine <app>)."
