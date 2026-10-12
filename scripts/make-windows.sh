#!/bin/bash
# Cross-compile OpenDiablo2 for Windows from a Mac or Linux machine.
#
#   scripts/make-windows.sh                 dist/windows-amd64/OpenDiablo2.exe (PE32+ x86-64)
#   ZIP=1 scripts/make-windows.sh           also write dist/OpenDiablo2-<describe>-windows-amd64.zip
#   WINGUI=1 scripts/make-windows.sh        build for the GUI subsystem (no console window; log lines are then
#                                           invisible, so the default keeps the console for the first test runs)
#   ARCH=arm64 scripts/make-windows.sh      Windows on ARM: only with Ebiten >= 2.10 (the script checks), see docs/WINDOWS.md
#
# No C compiler is needed: with CGO_ENABLED=0 the pinned Ebiten (v2.0.2) talks to Windows through x/sys/windows and an
# embedded glfw.dll, and the audio backend (oto) uses winmm. A zig or mingw toolchain is therefore NOT required. (Earlier
# zig builds only worked because the Unix-only code was split behind build tags, which is now done in the source.)
# If a future Ebiten upgrade needs cgo, use:  CGO_ENABLED=1 CC="zig cc -target x86_64-windows-gnu" scripts/make-windows.sh
# (install zig with `brew install zig`; this script never installs anything).
#
# The zip holds the .exe, a Windows README and the licence. It holds NO game files; you need your own Diablo II +
# Lord of Destruction (see docs/WINDOWS.md). The Windows build is cross-compiled only and has never been run on Windows.
set -euo pipefail

cd "$(dirname "$0")/.."
ARCH=${ARCH:-amd64}

EBITEN=$(sed -n 's|.*hajimehoshi/ebiten/v2 v\(.*\)|\1|p' go.mod | head -1)
if [ "$ARCH" != "amd64" ] && [ "${EBITEN#2.0.}" != "$EBITEN" ]; then
	echo "error: only ARCH=amd64 builds with Ebiten v$EBITEN. windows/arm64 fails in the pinned Ebiten v2.0.2 (internal/glfw has no embedded" >&2
	echo "       glfw.dll for arm64: 'undefined: glfwDLLCompressed'). Run the amd64 build on Windows-on-ARM (x64 emulation)," >&2
	echo "       or upgrade Ebiten first. See docs/WINDOWS.md." >&2
	exit 2
fi

BRANCH=$(git rev-parse --abbrev-ref HEAD 2>/dev/null || echo local)
COMMIT=$(git rev-parse --short HEAD 2>/dev/null || echo build)
DESCRIBE=$(git describe --tags --always --dirty --match 'v[0-9]*' 2>/dev/null || echo "$COMMIT")
LDFLAGS="-X main.GitBranch=$BRANCH -X main.GitCommit=$COMMIT"
[ -n "${WINGUI:-}" ] && LDFLAGS="$LDFLAGS -H=windowsgui"

OUT="dist/windows-$ARCH"
mkdir -p "$OUT"
rm -f "$OUT/OpenDiablo2.exe"

echo "building $OUT/OpenDiablo2.exe ($DESCRIBE)"
CGO_ENABLED=0 GOOS=windows GOARCH=$ARCH go build -trimpath -buildvcs=false -ldflags "$LDFLAGS" -o "$OUT/OpenDiablo2.exe" .

if command -v file >/dev/null; then
	INFO=$(file "$OUT/OpenDiablo2.exe")
	echo "$INFO"
	case "$INFO" in
	*PE32+*) ;;
	*)
		echo "error: not a PE32+ executable" >&2
		exit 1
		;;
	esac
fi

cp docs/WINDOWS.md "$OUT/README-WINDOWS.md"
cp LICENSE "$OUT/LICENSE.txt"

if [ -n "${ZIP:-}" ]; then
	ZIPFILE="$PWD/dist/OpenDiablo2-$DESCRIBE-windows-$ARCH.zip"
	rm -f "$ZIPFILE"
	(cd "$OUT" && env -u ZIP zip -q -X "$ZIPFILE" OpenDiablo2.exe README-WINDOWS.md LICENSE.txt)
	echo "wrote $ZIPFILE"
fi
