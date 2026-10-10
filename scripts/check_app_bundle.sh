#!/bin/bash
# Static checks of a built OpenDiablo2.app (and, optionally, its disk image):
# structure, Info.plist keys, arm64 only, minimum macOS, icon, ad-hoc signature,
# no game data or Blizzard files, no OD2_* environment baked into the plist.
# Nothing is launched.
#   scripts/check_app_bundle.sh [path/to/OpenDiablo2.app] [path/to/image.dmg]
#   (default app: dist/OpenDiablo2.app; with a dmg also checks its layout)
# ALLOW_UNIVERSAL=1 accepts an arm64+x86_64 binary (make-app.sh UNIVERSAL=1).
set -u
cd "$(dirname "$0")/.." || exit 2
APP=${1:-dist/OpenDiablo2.app}
DMG=${2:-}
fail=0
bad() { echo "FAIL: $*"; fail=1; }
ok() { echo "ok:   $*"; }

[ -d "$APP" ] || { echo "no bundle at $APP" >&2; exit 2; }
C="$APP/Contents"
PLIST="$C/Info.plist"
EXPECT_ID=io.github.sermare.OpenDiablo2
BIN="$C/MacOS/OpenDiablo2"

for f in Info.plist MacOS/OpenDiablo2 Resources/OpenDiablo2.icns Resources/d2logo.png; do
	[ -f "$C/$f" ] && ok "has $f" || bad "missing Contents/$f"
done
[ -x "$BIN" ] || bad "executable bit missing"
n=$(find "$C/MacOS" -type f | wc -l | tr -d ' ')
[ "$n" = 1 ] && ok "one executable in Contents/MacOS" || bad "Contents/MacOS has $n files, want 1"

# only these top-level entries may exist in Contents (_CodeSignature appears after signing)
extra=$(ls -A "$C" | grep -vxE 'Info.plist|MacOS|Resources|_CodeSignature' || true)
[ -z "$extra" ] && ok "Contents holds only Info.plist, MacOS, Resources, _CodeSignature" || { echo "$extra"; bad "unexpected entries in Contents"; }

# symlinks could point outside the bundle (or at a user's game folder)
links=$(find "$APP" -type l)
[ -z "$links" ] && ok "no symlinks in the bundle" || { echo "$links"; bad "symlinks inside the bundle"; }

# a quarantine flag on a freshly built bundle means it was copied from a download
if xattr -rl "$APP" 2>/dev/null | grep -q "com.apple.quarantine"; then
	bad "bundle carries com.apple.quarantine (build it locally; users clear it with xattr -dr)"
else
	ok "no quarantine attribute on the bundle"
fi

if plutil -lint "$PLIST" >/dev/null 2>&1; then
	ok "Info.plist is valid"
	pb() { /usr/libexec/PlistBuddy -c "Print :$1" "$PLIST" 2>/dev/null; }
	[ "$(pb CFBundleExecutable)" = OpenDiablo2 ] && ok "CFBundleExecutable" || bad "CFBundleExecutable"
	[ "$(pb CFBundleIconFile)" = OpenDiablo2 ] && ok "CFBundleIconFile" || bad "CFBundleIconFile"
	[ "$(pb CFBundlePackageType)" = APPL ] && ok "package type APPL" || bad "package type"
	[ "$(pb CFBundleName)" = OpenDiablo2 ] && ok "CFBundleName" || bad "CFBundleName"
	[ "$(pb NSHighResolutionCapable)" = true ] && ok "NSHighResolutionCapable (Retina)" || bad "NSHighResolutionCapable not true"
	[ "$(pb CFBundleIdentifier)" = "$EXPECT_ID" ] && ok "bundle id $EXPECT_ID" || bad "CFBundleIdentifier is '$(pb CFBundleIdentifier)', want $EXPECT_ID"

	ver=$(pb CFBundleShortVersionString)
	if printf '%s' "$ver" | grep -Eq '^[0-9]+(\.[0-9]+){0,2}$'; then ok "CFBundleShortVersionString $ver"; else bad "CFBundleShortVersionString '$ver' is not 1-3 integers"; fi
	build=$(pb CFBundleVersion)
	if printf '%s' "$build" | grep -Eq '^[0-9]+(\.[0-9]+){0,2}$'; then ok "CFBundleVersion $build"; else bad "CFBundleVersion '$build' is not numeric"; fi
	[ -n "$(pb OD2GitDescribe)" ] && ok "OD2GitDescribe $(pb OD2GitDescribe)" || bad "OD2GitDescribe missing (git describe string)"

	min=$(pb LSMinimumSystemVersion)
	case "$min" in 14|14.*|15*|26*) ok "LSMinimumSystemVersion $min" ;; *) bad "LSMinimumSystemVersion is '$min', want 14.0 or newer" ;; esac
	[ "$(pb LSApplicationCategoryType)" = public.app-category.games ] && ok "category games" || bad "LSApplicationCategoryType is not public.app-category.games"
	prio=$(pb LSArchitecturePriority | tr -d ' \n')
	case "$prio" in "Array{arm64}") ok "LSArchitecturePriority arm64" ;; *) bad "LSArchitecturePriority is '$prio', want just arm64" ;; esac

	[ -z "$(pb CFBundleDocumentTypes)" ] && ok "no document types declared" || bad "unexpected CFBundleDocumentTypes"
	[ -z "$(pb CFBundleURLTypes)" ] && ok "no URL schemes declared" || bad "unexpected CFBundleURLTypes"
	# real maps are the engine default; a plist that sets OD2_* would silently change behaviour for every player
	if /usr/libexec/PlistBuddy -c "Print :LSEnvironment" "$PLIST" >/dev/null 2>&1; then bad "LSEnvironment present (the app must not depend on environment variables)"; else ok "no LSEnvironment"; fi
	for k in LSUIElement LSBackgroundOnly NSAppTransportSecurity; do
		/usr/libexec/PlistBuddy -c "Print :$k" "$PLIST" >/dev/null 2>&1 && bad "unexpected key $k"
	done
	if grep -q 'OD2_' "$PLIST"; then bad "Info.plist mentions OD2_*"; else ok "Info.plist mentions no OD2_* variable"; fi
	if grep -q '/Users/' "$PLIST"; then bad "Info.plist contains a /Users path"; else ok "Info.plist has no user paths"; fi
else
	bad "Info.plist is not valid"
fi

archs=$(lipo -archs "$BIN" 2>/dev/null)
case " $archs " in *" arm64 "*) ok "arm64 slice ($archs)" ;; *) bad "no arm64 slice (got '$archs')" ;; esac
if [ "$archs" = arm64 ] || [ "${ALLOW_UNIVERSAL:-0}" = 1 ]; then ok "architectures: $archs"; else bad "binary is '$archs', want arm64 only (ALLOW_UNIVERSAL=1 to accept)"; fi

# Mach-O deployment target must not exceed what Info.plist promises
minos=$(otool -l "$BIN" 2>/dev/null | awk '/LC_BUILD_VERSION/{f=1} f&&/minos/{print $2; exit}')
if [ -n "$minos" ]; then
	if [ "${minos%%.*}" -le 14 ] 2>/dev/null; then ok "Mach-O minos $minos"; else bad "Mach-O minos $minos is newer than macOS 14"; fi
else
	bad "no LC_BUILD_VERSION in the binary"
fi

if codesign --verify --deep --strict "$APP" 2>/dev/null; then ok "codesign --verify --deep --strict"; else bad "codesign verification failed"; fi
sig=$(codesign -dv "$APP" 2>&1)
case "$sig" in *"Signature=adhoc"*) ok "ad-hoc signature" ;; *) bad "not ad-hoc signed" ;; esac
case "$sig" in *"Identifier=$EXPECT_ID"*) ok "signature identifier is the bundle id" ;; *) bad "signature identifier is not $EXPECT_ID" ;; esac
case "$sig" in *"Info.plist=not bound"*) bad "Info.plist is not bound by the signature" ;; *) ok "Info.plist bound by the signature" ;; esac

# no game data, saves or Blizzard artwork, by extension and by size
forbidden=$(find "$APP" -type f \( -iname '*.mpq' -o -iname '*.d2s' -o -iname '*.d2x' -o -iname '*.dc6' -o -iname '*.dcc' \
	-o -iname '*.dt1' -o -iname '*.ds1' -o -iname '*.cof' -o -iname '*.wav' -o -iname '*.tbl' -o -iname '*.bik' \
	-o -iname '*.pl2' -o -iname '*.dat' -o -iname '*.txt' -o -iname '*.exe' -o -iname '*.dll' -o -iname '*.key' \
	-o -iname '*.json' -o -iname '*.log' -o -iname '*.mp3' -o -iname '*.ogg' \))
if [ -n "$forbidden" ]; then echo "$forbidden"; bad "forbidden files inside the bundle"; else ok "no forbidden files"; fi
big=$(find "$APP/Contents/Resources" -type f -size +2M 2>/dev/null)
[ -z "$big" ] && ok "no Resources file over 2 MB" || { echo "$big"; bad "large resource file"; }
# game archives start with the MPQ magic; catch a renamed one
mpqmagic=$(find "$APP" -type f ! -path "$BIN" -exec sh -c 'for f; do [ "$(head -c 3 "$f")" = "MPQ" ] && echo "$f"; done' sh {} + 2>/dev/null)
[ -z "$mpqmagic" ] && ok "no file with an MPQ header" || { echo "$mpqmagic"; bad "MPQ archive hidden in the bundle"; }

# strings in the binary: first-run, log and config isolation wiring must be present,
# and nothing may point at a user's home
STR=$(strings -a "$BIN" 2>/dev/null)
echo "$STR" | grep -q "OD2_CONFIG_DIR" && ok "binary honours OD2_CONFIG_DIR" || bad "binary lacks OD2_CONFIG_DIR (config isolation)"
echo "$STR" | grep -q "OpenDiablo2.previous.log" && ok "binary rotates ~/Library/Logs/OpenDiablo2 logs" || bad "binary lacks the log rotation (OpenDiablo2.previous.log)"
echo "$STR" | grep -q "choose folder" && ok "binary has the Diablo II folder picker" || bad "binary lacks the folder picker (first run without game files)"
USERPATHS=$(echo "$STR" | grep -oE "/Users/[A-Za-z0-9._-]+/[^ ]*" | grep -v "^/Users/Shared/" | sort -u)
if [ -n "$USERPATHS" ]; then
	echo "$USERPATHS" | head -5
	bad "binary embeds a /Users path (build with -trimpath)"
else
	ok "binary embeds no /Users path"
fi

# disk image layout (optional second argument)
if [ -n "$DMG" ]; then
	[ -f "$DMG" ] || { bad "no disk image at $DMG"; DMG=; }
fi
if [ -n "$DMG" ]; then
	if hdiutil verify -quiet "$DMG" >/dev/null 2>&1; then ok "hdiutil verify $DMG"; else bad "hdiutil verify failed"; fi
	MNT=$(mktemp -d)
	if hdiutil attach -quiet -readonly -nobrowse -noverify -mountpoint "$MNT" "$DMG" >/dev/null 2>&1; then
		[ -d "$MNT/OpenDiablo2.app" ] && ok "dmg holds OpenDiablo2.app" || bad "dmg lacks OpenDiablo2.app"
		if [ -L "$MNT/Applications" ] && [ "$(readlink "$MNT/Applications")" = /Applications ]; then ok "dmg has the Applications symlink"; else bad "dmg lacks the Applications -> /Applications symlink"; fi
		[ -f "$MNT/READ ME FIRST.txt" ] && ok "dmg has READ ME FIRST.txt" || bad "dmg lacks READ ME FIRST.txt"
		stray=$(ls -A "$MNT" | grep -vxE 'OpenDiablo2.app|Applications|READ ME FIRST.txt|\.fseventsd|\.Trashes|\.VolumeIcon.icns|\.DS_Store' || true)
		[ -z "$stray" ] && ok "dmg has nothing else (no background picture)" || { echo "$stray"; bad "unexpected items in the dmg"; }
		[ ! -e "$MNT/.background" ] || bad "dmg has a .background folder"
		vol=$(diskutil info "$MNT" 2>/dev/null | sed -n 's/^ *Volume Name: *//p')
		case "$vol" in OpenDiablo2*) ok "volume name '$vol'" ;; *) bad "volume name '$vol' should start with OpenDiablo2" ;; esac
		if codesign --verify --deep --strict "$MNT/OpenDiablo2.app" 2>/dev/null; then ok "app inside the dmg verifies"; else bad "app inside the dmg fails codesign"; fi
		dforbidden=$(find "$MNT" -type f \( -iname '*.mpq' -o -iname '*.d2s' -o -iname '*.dc6' -o -iname '*.dt1' -o -iname '*.ds1' \))
		[ -z "$dforbidden" ] && ok "no game files in the dmg" || bad "game files in the dmg"
		hdiutil detach -quiet "$MNT" >/dev/null 2>&1 || hdiutil detach -quiet -force "$MNT" >/dev/null 2>&1
	else
		bad "could not mount $DMG"
	fi
	rmdir "$MNT" 2>/dev/null
fi

[ "$fail" = 0 ] && echo "BUNDLE OK" || echo "BUNDLE CHECK FAILED"
exit $fail
