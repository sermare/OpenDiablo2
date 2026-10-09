#!/bin/zsh
# Sound: scenarios are muted (OD2_AUTOTEST_MUTE) unless OD2_VERIFY_SOUND=1.
OD2_VERIFY_MUTE_ENV="OD2_AUTOTEST_MUTE=1"
[ -n "$OD2_VERIFY_SOUND" ] && OD2_VERIFY_MUTE_ENV=""
export OD2_VERIFY_MUTE_ENV
# One-command verification for the macOS fork. Needs a Diablo II 1.14b + LoD
# install (see docs/macos-quickstart.md). Environment variables (all optional):
#   D2_TABLES        folder with extracted game tables (itemstatcost.bin, armor.txt, ...)
#   D2S_SAMPLE_BODY  a real .d2s save with a body; D2S_SAMPLE_BODY_JSON its expected parse
#   OD2_VERIFY_SAVE  a .d2s file to start in the game (default: $D2S_SAMPLE_BODY copied to a .d2s)
set -u
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
  for i in {1..120}; do pgrep -f "$tmp/od2" >/dev/null && break; sleep 1; done
  for i in {1..420}; do pgrep -f "$tmp/od2" >/dev/null || break; sleep 1; done
}
# every run gets its own scratch folder and server port, so parallel runs (e.g. several agents) do not collide
tmp=$(mktemp -d /tmp/od2-verify.XXXXXX)
step() { printf '\n== %s\n' "$1"; }

# every run uses its own server port so parallel runs (e.g. several agents) do not collide
export OD2_PORT=$(( 20000 + RANDOM % 20000 ))
while lsof -nP -iTCP:$OD2_PORT -sTCP:LISTEN >/dev/null 2>&1; do export OD2_PORT=$(( 20000 + RANDOM % 20000 )); done

step "build"
go build -o $tmp/od2 . 2>&1 | grep -v "ld: warning" ; [ ${pipestatus[1]} -eq 0 ] || { echo "BUILD FAILED"; exit 1; }

step "unit tests"
go test ./... 2>&1 | grep -v "ld: warning\|no test files\|^# " | grep -v "^ok" ; [ ${pipestatus[1]} -eq 0 ] || fail=1
echo "(only failures are printed above)"

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
#   scenario_warnings_ok=1   (optional) do not fail on [ERROR]/[WARNING] lines
# Adding a scenario = adding one small file; no edits to this runner are needed.
# A GUI session is required (the game is started with `open`).
# ---------------------------------------------------------------------------
if [ -n "${D2S_SAMPLE_BODY:-}" ]; then
  save="${OD2_VERIFY_SAVE:-$tmp/save.d2s}"
  [ -f "$save" ] || cp "$D2S_SAMPLE_BODY" "$save"

  for f in scripts/verify.d/*.sh(N); do
    unset -f scenario_env scenario_check 2>/dev/null; scenario_name="${f:t}"; scenario_warnings_ok=""
    source "$f"
    step "$scenario_name"
    n=${f:t:r}
    cmd=$tmp/$n.command log=$tmp/$n.log
    {
      echo '#!/bin/zsh'
      echo "export OD2_PORT=$OD2_PORT"
      echo "export OD2_AUTOGAME=\"$save\" ${OD2_VERIFY_MUTE_ENV} OD2_AUTOEXIT=1"
      scenario_env
      echo "$tmp/od2 2>&1 | tee $log"
    } > $cmd
    chmod +x $cmd; rm -f $log
    launch_game $cmd
    wait_run
    sed 's/\x1b\[[0-9;]*m//g' $log > $log.txt
    scenario_check
    if [ -z "$scenario_warnings_ok" ] && grep -E "\[(ERROR|WARNING)\]|panic" $log.txt | grep -v "skipping missing"; then
      echo "FAIL: warnings/errors in the $scenario_name log"; fail=1
    fi
  done
fi

echo
[ $fail -eq 0 ] && echo "ALL CHECKS PASSED" || { echo "SOME CHECKS FAILED"; exit 1; }
