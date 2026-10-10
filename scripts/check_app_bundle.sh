#!/bin/bash
# Checks a built OpenDiablo2.app: structure, Info.plist keys, arm64, icon,
# ad-hoc signature, and that no game data or Blizzard files are inside.
#   scripts/check_app_bundle.sh [path/to/OpenDiablo2.app]   (default dist/OpenDiablo2.app)
set -u
cd "$(dirname "$0")/.." || exit 2
APP=${1:-dist/OpenDiablo2.app}
fail=0
bad() { echo "FAIL: $*"; fail=1; }
ok() { echo "ok:   $*"; }

[ -d "$APP" ] || { echo "no bundle at $APP" >&2; exit 2; }
C="$APP/Contents"
PLIST="$C/Info.plist"

for f in Info.plist MacOS/OpenDiablo2 Resources/OpenDiablo2.icns Resources/d2logo.png; do
	[ -f "$C/$f" ] && ok "has $f" || bad "missing Contents/$f"
done
[ -x "$C/MacOS/OpenDiablo2" ] || bad "executable bit missing"

if plutil -lint "$PLIST" >/dev/null 2>&1; then
	ok "Info.plist is valid"
	pb() { /usr/libexec/PlistBuddy -c "Print :$1" "$PLIST" 2>/dev/null; }
	[ "$(pb CFBundleExecutable)" = OpenDiablo2 ] && ok "CFBundleExecutable" || bad "CFBundleExecutable"
	[ "$(pb CFBundleIconFile)" = OpenDiablo2 ] && ok "CFBundleIconFile" || bad "CFBundleIconFile"
	[ "$(pb CFBundlePackageType)" = APPL ] && ok "package type APPL" || bad "package type"
	[ "$(pb NSHighResolutionCapable)" = true ] && ok "NSHighResolutionCapable (Retina)" || bad "NSHighResolutionCapable not true"
	[ -n "$(pb CFBundleIdentifier)" ] && ok "bundle id $(pb CFBundleIdentifier)" || bad "CFBundleIdentifier"
	[ -n "$(pb CFBundleShortVersionString)" ] || bad "CFBundleShortVersionString"
	min=$(pb LSMinimumSystemVersion)
	case "$min" in 14|14.*|15*|26*) ok "LSMinimumSystemVersion $min" ;; *) bad "LSMinimumSystemVersion is '$min', want 14.0 or newer" ;; esac
	[ -z "$(pb CFBundleDocumentTypes)" ] && ok "no document types declared" || bad "unexpected CFBundleDocumentTypes"
else
	bad "Info.plist is not valid"
fi

archs=$(lipo -archs "$C/MacOS/OpenDiablo2" 2>/dev/null)
case " $archs " in *" arm64 "*) ok "arm64 slice ($archs)" ;; *) bad "no arm64 slice (got '$archs')" ;; esac

if codesign --verify --deep --strict "$APP" 2>/dev/null; then ok "codesign --verify --deep --strict"; else bad "codesign verification failed"; fi
sig=$(codesign -dv "$APP" 2>&1)
case "$sig" in *"Signature=adhoc"*) ok "ad-hoc signature" ;; *) bad "not ad-hoc signed" ;; esac

# no game data, saves or Blizzard artwork, by extension and by size
forbidden=$(find "$APP" -type f \( -iname '*.mpq' -o -iname '*.d2s' -o -iname '*.d2x' -o -iname '*.dc6' -o -iname '*.dcc' \
	-o -iname '*.dt1' -o -iname '*.ds1' -o -iname '*.cof' -o -iname '*.wav' -o -iname '*.tbl' -o -iname '*.bik' \
	-o -iname '*.pl2' -o -iname '*.dat' -o -iname '*.txt' -o -iname '*.exe' -o -iname '*.dll' \))
if [ -n "$forbidden" ]; then echo "$forbidden"; bad "forbidden files inside the bundle"; else ok "no forbidden files"; fi
big=$(find "$APP/Contents/Resources" -type f -size +2M 2>/dev/null)
[ -z "$big" ] && ok "no Resources file over 2 MB" || { echo "$big"; bad "large resource file"; }

# nothing in the bundle may point at a user's home or a build machine
if strings "$C/MacOS/OpenDiablo2" | grep -q "/Users/[a-z]*/Library/Application Support/OpenDiablo2/Saves"; then
	bad "binary embeds a user save path"
fi

[ "$fail" = 0 ] && echo "BUNDLE OK" || echo "BUNDLE CHECK FAILED"
exit $fail
