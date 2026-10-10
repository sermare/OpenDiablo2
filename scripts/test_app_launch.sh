#!/bin/bash
# Launch dist/OpenDiablo2.app like Finder does (open -n -W, no terminal) against
# a TEMPORARY config dir (never your real settings or saves), let it reach the
# main menu, take a screenshot, quit through OD2_AUTOEXIT and check the log.
# Needs a GUI session and your own Diablo II folder:
#   scripts/test_app_launch.sh [app] [diablo-ii-dir]
# One game window at a time; the app is killed by pid if it outlives the timeout.
set -u
cd "$(dirname "$0")/.." || exit 2
APP=${1:-dist/OpenDiablo2.app}
GAME=${2:-$HOME/.wine-d2classic/drive_c/Program Files (x86)/Diablo II}
TMP=$(mktemp -d "${TMPDIR:-/tmp}/od2-launch.XXXXXX")
LOG="$HOME/Library/Logs/OpenDiablo2/OpenDiablo2.log"
mkdir -p "$TMP/cfg"

# 1. missing files: a fake empty HOME (so nothing is discovered) must give a clear alert, no crash
echo "== launch with no game files (empty HOME, nothing to discover)"
mkdir -p "$TMP/empty" "$TMP/home"
HLOG="$TMP/home/Library/Logs/OpenDiablo2/OpenDiablo2.log"
launchctl asuser "$(id -u)" /usr/bin/open -n -W \
	--env HOME="$TMP/home" --env OD2_CONFIG_DIR="$TMP/cfg" --env OD2_SETUP_AUTOPICK="$TMP/empty" --env OD2_AUTOEXIT=1 "$APP"
grep -q "ALERT OpenDiablo2" "$HLOG" && echo "ok:   missing-files alert shown" || { echo "FAIL: no alert"; tail -5 "$HLOG"; exit 1; }
grep -q "panic:\|SIGSEGV" "$HLOG" && { echo "FAIL: crash with no game files"; exit 1; }

# 2. first run finds the game folder, reaches the menu, screenshot, clean exit
echo "== launch with game files (first-run setup picks $GAME)"
rm -rf "$TMP/cfg"; mkdir -p "$TMP/cfg"
launchctl asuser "$(id -u)" /usr/bin/open -n -W \
	--env OD2_CONFIG_DIR="$TMP/cfg" --env OD2_SETUP_AUTOPICK="$GAME" --env OD2_AUTOSHOT="$TMP/shot.png" \
	--env OD2_AUTOSHOT_SECONDS=8 --env OD2_AUTOEXIT=1 --env OD2_AUTOTEST_MUTE=1 "$APP" &
OPENPID=$!
for _ in $(seq 1 90); do kill -0 $OPENPID 2>/dev/null || break; sleep 1; done
if kill -0 $OPENPID 2>/dev/null; then
	echo "FAIL: app still running after 90 s; killing"
	APPPID=$(pgrep -f "$APP/Contents/MacOS/OpenDiablo2" | head -1)
	[ -n "$APPPID" ] && kill "$APPPID"; kill $OPENPID 2>/dev/null; exit 1
fi
rc=0
[ -f "$TMP/shot.png" ] && echo "ok:   screenshot $TMP/shot.png" || { echo "FAIL: no screenshot"; rc=1; }
grep -q "AUTOSHOT saved" "$LOG" && echo "ok:   AUTOSHOT saved" || rc=1
grep -q '"MpqPath"' "$TMP/cfg/config.json" && echo "ok:   config written under the temp dir" || { echo "FAIL: no config"; rc=1; }
grep -q "panic:\|SIGSEGV\|OpenDiablo2 stopped" "$LOG" && { echo "FAIL: crash in log"; rc=1; }
echo "temp dir: $TMP"
[ $rc = 0 ] && echo "LAUNCH OK" || echo "LAUNCH FAILED"
exit $rc
