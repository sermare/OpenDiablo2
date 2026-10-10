#!/bin/bash
# One-command, repeatable Mac release build. Static only: nothing is launched.
#   scripts/release.sh
# Steps: go build (arm64, -trimpath) -> ad-hoc codesign -> zip -> dmg -> check_app_bundle.sh
# on the app and the dmg -> SHA-256 sums. Output in dist/. No game files are needed or included.
# Environment: VERSION=x.y.z overrides the tag-derived version; SOURCE_DATE_EPOCH overrides
# the commit time used to stamp file dates. ALLOW_DIRTY=1 skips the clean-tree warning.
set -euo pipefail

cd "$(dirname "$0")/.."

[ "$(uname -s)" = Darwin ] || { echo "release.sh needs macOS (cgo, codesign, hdiutil)" >&2; exit 1; }
[ "$(uname -m)" = arm64 ] || echo "warning: not an Apple Silicon host; the arm64 build may fail" >&2

if [ -n "$(git status --porcelain --untracked-files=no 2>/dev/null)" ] && [ "${ALLOW_DIRTY:-0}" != 1 ]; then
	echo "warning: tracked files have uncommitted changes; the version will end in -dirty (ALLOW_DIRTY=1 to silence)" >&2
fi

export SOURCE_DATE_EPOCH=${SOURCE_DATE_EPOCH:-$(git log -1 --format=%ct)}
export TZ=UTC LC_ALL=C

rm -rf dist
ZIP=1 scripts/make-app.sh
scripts/make-dmg.sh dist/OpenDiablo2.app

DMG=$(ls dist/*.dmg)
scripts/check_app_bundle.sh dist/OpenDiablo2.app "$DMG"
scripts/dmg_install_check.sh "$DMG"

(cd dist && shasum -a 256 -- *.zip *.dmg | tee SHA256SUMS)
echo "release artifacts in dist/ (not notarised: users right-click > Open, see docs/macos-quickstart.md)"
