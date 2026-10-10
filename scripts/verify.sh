#!/bin/zsh
# One-command verification for the macOS fork. Needs a Diablo II 1.14b + LoD
# install (see docs/macos-quickstart.md). Environment variables (all optional):
#   D2_TABLES        folder with extracted game tables (itemstatcost.bin, armor.txt, ...)
#   D2S_SAMPLE_BODY  a real .d2s save with a body; D2S_SAMPLE_BODY_JSON its expected parse
#   OD2_VERIFY_SAVE  a .d2s file to start in the game (default: $D2S_SAMPLE_BODY copied to a .d2s)
set -u
# multi-process scenarios (96, 9d) put this in their own launch lines: muted unless OD2_VERIFY_SOUND=1
OD2_VERIFY_MUTE_ENV="OD2_AUTOTEST_MUTE=1"
[ -n "${OD2_VERIFY_SOUND:-}" ] && OD2_VERIFY_MUTE_ENV=""
export OD2_VERIFY_MUTE_ENV
cd "${0:A:h}/.."

fail=0


# Start a .command file in the user's GUI session WITHOUT opening a Terminal window (every `open x.command`
# leaves a window behind; hundreds of them stop Terminal from working). Order of preference:
#   $OD2_VERIFY_LAUNCH (e.g. "launchctl asuser 501 /bin/zsh"), launchctl asuser, then `open` as a last resort.
launch_game() {
  if [ -n "${OD2_VERIFY_LAUNCH:-}" ]; then ${=OD2_VERIFY_LAUNCH} $1 >/dev/null 2>&1 &
  elif launchctl asuser $(id -u) /usr/bin/true >/dev/null 2>&1; then launchctl asuser $(id -u) /bin/zsh $1 >/dev/null 2>&1 &
  else open $1
  fi
}

# the launcher returns before the game starts (slowly, on a busy machine): wait until the
# game process appears, then until it exits
wait_run() {
  local i
  for i in {1..120}; do
    pgrep -f "$tmp/od2" >/dev/null && break
    # a game that crashed or finished instantly is already gone but left its log: do not wait the full start timeout
    [ $i -gt 5 ] && [ -s "${log:-/nonexistent}" ] && return
    sleep 1
  done
  for i in {1..${scenario_timeout:-420}}; do pgrep -f "$tmp/od2" >/dev/null || break; sleep 1; done
  # a game still alive now is stuck (e.g. on the title screen): kill it, never leave windows behind.
  # The pattern is this run's private scratch dir, so other runs/agents/the user's own games are untouched.
  if pgrep -f "$tmp/od2" >/dev/null; then echo "REAPED: game of this scenario did not finish (stuck?), killing it"; pkill -f "$tmp/od2"; sleep 1; pkill -9 -f "$tmp/od2" 2>/dev/null; fi
}
# every run gets its own scratch folder and server port, so parallel runs (e.g. several agents) do not collide
tmp=$(mktemp -d /tmp/od2-verify.XXXXXX)
cleanup_games() { pkill -f "$tmp/od2" 2>/dev/null; pkill -f "$tmp/[0-9a-z-]*\.command" 2>/dev/null; }
trap cleanup_games EXIT
trap 'cleanup_games; exit 130' INT
trap 'cleanup_games; exit 143' TERM
step() { printf '\n== %s\n' "$1"; }

# every run uses its own server port so parallel runs (e.g. several agents) do not collide
export OD2_PORT=$(( 20000 + RANDOM % 20000 ))
while lsof -nP -iTCP:$OD2_PORT -sTCP:LISTEN >/dev/null 2>&1; do export OD2_PORT=$(( 20000 + RANDOM % 20000 )); done

# a verify without the real data silently skipped the scenario loop (false green): refuse to run
for v in D2S_SAMPLE_BODY D2S_SAMPLE_BODY_JSON D2_TABLES; do
  [ -n "${(P)v:-}" ] || { echo "VERIFY ABORTED: $v is unset; without it the real-data checks and the in-game scenarios would be skipped (false green)"; exit 1; }
done
scenarios_run=0

step "repo hygiene (no game files / decompiled code)"
scripts/check_repo_hygiene.sh || { echo "REPO HYGIENE FAILED"; exit 1; }

step "build"
go build -o $tmp/od2 . 2>&1 | grep -v "ld: warning" ; [ ${pipestatus[1]} -eq 0 ] || { echo "BUILD FAILED"; exit 1; }

if [ -z "${SKIP_UNIT:-}" ]; then
step "unit tests"
go test ./... 2>&1 | grep -v "ld: warning\|no test files\|^# " | grep -v "^ok" ; [ ${pipestatus[1]} -eq 0 ] || fail=1
echo "(only failures are printed above)"
fi

# Level generation on the real game data: time budget per level kind (cold caches, then warm) and the MPQ
# decoder checks against the old decoder. The install folder is D2_GAME_DIR, else MpqPath of the game config.
game_dir="${D2_GAME_DIR:-}"
if [ -z "$game_dir" ]; then
  game_dir=$(sed -n 's/.*"MpqPath": *"\(.*\)".*/\1/p' "$HOME/Library/Application Support/OpenDiablo2/config.json" 2>/dev/null | head -1)
fi
if [ -n "$game_dir" ] && [ -f "$game_dir/d2data.mpq" ]; then
  step "level generation budget (real data: every act, outdoor / preset / maze levels)"
  D2_GAME_DIR="$game_dir" go test -count=1 -v -run 'LevelBudget' ./d2core/d2map/d2mapgen/ 2>&1 | grep -E "PERF level|^(--- |FAIL|ok)" | cut -c1-160 | tee $tmp/budget.txt
  grep -q "^ok" $tmp/budget.txt || { echo "FAIL: level generation budget"; fail=1; }
  step "MPQ decoder against the old decoder (every imploded sector of d2data.mpq)"
  D2_GAME_DIR="$game_dir" go test -count=1 -v -run 'Explode' ./d2common/d2fileformats/d2mpq/ 2>&1 | grep -E "^(--- |FAIL|ok)|identical" | tee $tmp/explode.txt
  grep -q "^ok" $tmp/explode.txt || { echo "FAIL: MPQ explode decoder"; fail=1; }
fi

if [ -n "${D2S_SAMPLE_BODY:-}" ]; then
  step "real .d2s oracle tests"
  go test -v ./d2common/d2fileformats/d2s/ 2>&1 | grep -E "^(--- |FAIL|ok)" || fail=1
  step "save-back roundtrip (import -> export unchanged is byte-identical; edits survive a re-parse)"
  go test -v -run Export ./d2core/d2hero/ 2>&1 | grep -E "^(--- |FAIL|ok)" | tee $tmp/export.txt
  grep -q "^--- PASS: TestExportUnchangedIsByteIdentical" $tmp/export.txt || { echo "FAIL: .d2s export roundtrip"; fail=1; }
  grep -q "^--- FAIL\|^FAIL" $tmp/export.txt && fail=1
fi


# ---------------------------------------------------------------------------
# In-game scenarios: every file in scripts/verify.d/*.sh describes one. A file defines
#   scenario_name="human readable title"
#   scenario_env()    echo shell lines (exports) for the game; may use $save, $tmp, $OD2_PORT
#   scenario_check()  inspect $log.txt (ANSI-stripped log) and set fail=1 on problems
#   scenario_realtime=1      (optional) keep the game clock at real time (default: OD2_AUTOSPEED=4 for every scenario)
#   scenario_timeout=<seconds> (optional) wall-clock limit before the game is reaped (default 420; long playthroughs)
#   scenario_unmuted=1       (optional) play real audio (no OD2_AUTOTEST_MUTE); OD2_VERIFY_SOUND=1 does it for all
#   scenario_warnings_ok=1   (optional) do not fail on [ERROR]/[WARNING] lines
# Adding a scenario = adding one small file; no edits to this runner are needed.
# A GUI session is required (the game is started with `open`).
# ---------------------------------------------------------------------------
if [ -n "${D2S_SAMPLE_BODY:-}" ]; then
  save="${OD2_VERIFY_SAVE:-$tmp/save.d2s}"
  [ -f "$save" ] || cp "$D2S_SAMPLE_BODY" "$save"
  # make_hero <out.d2s> for the level 94 playthroughs; OD2_HERO=barb picks the generated Barbarian (default: the Sorceress)
  source scripts/verify.d/lib/hero.sh

  for f in scripts/verify.d/*.sh(N); do
    unset -f scenario_env scenario_check 2>/dev/null; scenario_name="${f:t}"; scenario_warnings_ok=""; scenario_unmuted=""; scenario_realtime=""; scenario_timeout=""
    source "$f"
    # OD2_VERIFY_ONLY=<glob> (e.g. "83-*") runs only the scenarios whose file name matches
    [ -n "${OD2_VERIFY_ONLY:-}" ] && [[ ${f:t} != ${~OD2_VERIFY_ONLY} && ${f:t:r} != ${~OD2_VERIFY_ONLY} ]] && continue
    fail_before=$fail
    scenarios_run=$((scenarios_run+1))
    for attempt in 1 2; do
      fail=$fail_before
      step "$scenario_name"
      # saves are written back to the .d2s they were loaded from: every run starts from a fresh copy
      if [ -z "${OD2_VERIFY_SAVE:-}" ]; then cp -f "$D2S_SAMPLE_BODY" "$save"; rm -f "$save.bak"; fi
      n=${f:t:r}
      cmd=$tmp/$n.command log=$tmp/$n.log; rm -f $log
      {
        echo '#!/bin/zsh'
        echo "export OD2_PORT=$OD2_PORT"
        echo "export OD2_AUTOGAME=\"$save\" OD2_AUTOEXIT=1"
        # muted unless OD2_VERIFY_SOUND=1 or the scenario sets scenario_unmuted=1 (real audio, uses the sound device)
        [ -n "${OD2_VERIFY_SOUND:-}" ] || [ -n "$scenario_unmuted" ] || echo "export OD2_AUTOTEST_MUTE=1"
        # game clock x4 (OD2_AUTOSPEED) unless the scenario needs real time (perf, multiplayer); OD2_VERIFY_SPEED=1 turns it off
        [ -n "$scenario_realtime" ] || [ "${OD2_VERIFY_SPEED:-8}" = 1 ] || echo "export OD2_AUTOSPEED=${OD2_VERIFY_SPEED:-8}"
        # OD2_VERIFY_CONFIG_DIR: a private config folder (a copy of a valid config.json) instead of the user's own;
        # a scenario that needs its own OD2_CONFIG_DIR overrides it below
        [ -n "${OD2_VERIFY_CONFIG_DIR:-}" ] && echo "export OD2_CONFIG_DIR=\"$OD2_VERIFY_CONFIG_DIR\""
        scenario_env
        echo "$tmp/od2 2>&1 | tee $log"
      } > $cmd
      chmod +x $cmd; rm -f $log
      slot=$(./scripts/gameslot.sh acquire $$)
      launch_game $cmd
      wait_run
      ./scripts/gameslot.sh release $slot
      sed 's/\x1b\[[0-9;]*m//g' $log > $log.txt
      scenario_check
      if [ -z "$scenario_warnings_ok" ] && grep -E "\[(ERROR|WARNING)\]|panic" $log.txt | grep -v "skipping missing" | grep -v "KILL giving up for now" | grep -v "D2S export: container item"; then
        echo "FAIL: warnings/errors in the $scenario_name log"; fail=1
      fi
      [ $fail -eq $fail_before ] && break
      [ $attempt -eq 1 ] && echo "RETRY: $scenario_name failed once; running it again (scenarios are timing sensitive on a loaded machine)"
    done
  done
fi

echo
echo "SCENARIOS RUN: $scenarios_run"
if [ $scenarios_run -eq 0 ]; then echo "VERIFY FAILED: the scenario loop ran 0 scenarios (D2S_SAMPLE_BODY unset or OD2_VERIFY_ONLY matched nothing)"; exit 1; fi
[ $fail -eq 0 ] && echo "ALL CHECKS PASSED" || { echo "SOME CHECKS FAILED"; exit 1; }
