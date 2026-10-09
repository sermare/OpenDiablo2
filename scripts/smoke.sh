#!/bin/zsh
# Fast smoke check: about 10 minutes instead of the hour-plus full scripts/verify.sh on a loaded machine.
#
#   1. repo hygiene, build, go vet, unit tests (they include the emulator goldens in d2common/d2combat/testdata
#      and the oracle cross-checks; with D2_TABLES / D2S_SAMPLE_BODY set the real-data oracle tests run too)
#   2. six in-game scenarios, one per area, each started from a FRESH copy of the sample save:
#        skill bar     98-skillbar        quest         85-quests          Act 1 walk   20-script-walk
#        multiplayer   96-multiplayer     hardcore      8c-hardcore        real map     70-real-maps
#      They are the scenario files of scripts/verify.d, run with the same environment contract as verify.sh.
#
# verify.sh is not changed and stays the full check; run it once per merge.
#
# Environment (all optional):
#   D2S_SAMPLE_BODY      sample .d2s (default: ~/git/nokka-d2s-ref/examples/nokkasorc when it exists)
#   D2_TABLES            extracted game tables for the real-data tests (default: ~/git/d2-tables when it exists)
#   SMOKE_SKIP_GAMES=1   only step 1 (no game window needed)
#   SMOKE_DRY_RUN=1      print what would run, run nothing
#   SMOKE_SCENARIOS      space separated scenario file basenames (without .sh) instead of the six above
#   SMOKE_RETRY          extra attempts after a failed scenario (default 1; scenarios are timing sensitive)
#   SMOKE_START_WAIT     seconds to wait for a game to appear (default 60)
#   SMOKE_RUN_WAIT       seconds a game may run before it is killed as stuck (default 240)
#   SMOKE_RACE=1         also run the race detector over d2common/d2core/d2game (adds a few minutes)
#   SMOKE_BIN            use this prebuilt game binary instead of building one
#   SMOKE_SKIP_CHECKS=1  skip hygiene, go vet and the unit tests (still prepares the binary); for re-running scenarios
#   OD2_MAX_GAMES        game windows at once, shared with every other run on the machine (scripts/gameslot.sh)
#
# Exit status 0 = every step passed, 1 = something failed. Games are started through scripts/gameslot.sh and never
# without a slot; a game that is still alive after SMOKE_RUN_WAIT is killed by its own scratch folder only.
set -u
cd "${0:A:h}/.."

typeset -a default_scenarios
default_scenarios=(98-skillbar 85-quests 20-script-walk 96-multiplayer 8c-hardcore 70-real-maps)
typeset -a scenarios
if [ -n "${SMOKE_SCENARIOS:-}" ]; then scenarios=(${=SMOKE_SCENARIOS}); else scenarios=($default_scenarios); fi

retry=${SMOKE_RETRY:-1}
start_wait=${SMOKE_START_WAIT:-60}
run_wait=${SMOKE_RUN_WAIT:-240}

[ -z "${D2S_SAMPLE_BODY:-}" ] && [ -f "$HOME/git/nokka-d2s-ref/examples/nokkasorc" ] && export D2S_SAMPLE_BODY="$HOME/git/nokka-d2s-ref/examples/nokkasorc"
[ -z "${D2S_SAMPLE_BODY_JSON:-}" ] && [ -f "$HOME/git/nokka-d2s-ref/examples/nokkasorc.json" ] && export D2S_SAMPLE_BODY_JSON="$HOME/git/nokka-d2s-ref/examples/nokkasorc.json"
[ -z "${D2_TABLES:-}" ] && [ -d "$HOME/git/d2-tables" ] && export D2_TABLES="$HOME/git/d2-tables"

if [ -n "${SMOKE_DRY_RUN:-}" ]; then
  echo "smoke.sh dry run"
  echo "  steps: hygiene, build, go vet ./..., go test ./...${SMOKE_RACE:+, go test -race}"
  echo "  D2S_SAMPLE_BODY=${D2S_SAMPLE_BODY:-<unset>}  D2_TABLES=${D2_TABLES:-<unset>}"
  if [ -n "${SMOKE_SKIP_GAMES:-}" ]; then echo "  games: skipped (SMOKE_SKIP_GAMES)"; else
    for s in $scenarios; do
      f=scripts/verify.d/$s.sh
      [ -f "$f" ] || { echo "  MISSING scenario file $f"; continue; }
      echo "  scenario $s: $(sed -n 's/^scenario_name="\(.*\)"$/\1/p' $f | head -1 | cut -c1-90)"
    done
  fi
  echo "  retry=$retry start_wait=${start_wait}s run_wait=${run_wait}s"
  exit 0
fi

# the same mute contract as verify.sh
OD2_VERIFY_MUTE_ENV="OD2_AUTOTEST_MUTE=1"
[ -n "${OD2_VERIFY_SOUND:-}" ] && OD2_VERIFY_MUTE_ENV=""
export OD2_VERIFY_MUTE_ENV

fail=0
tmp=$(mktemp -d /tmp/od2-smoke.XXXXXX)
typeset -a results   # "name|PASS/FAIL|seconds"
t0=$SECONDS

# kill only games started by THIS run (the pattern is this run's private scratch folder), then clean up
cleanup() {
  pkill -f "$tmp/od2" 2>/dev/null
  pkill -f "$tmp/[0-9a-z-]*\.command" 2>/dev/null
  sleep 1
  pkill -9 -f "$tmp/od2" 2>/dev/null
  return 0
}
trap 'cleanup' EXIT
trap 'cleanup; exit 130' INT TERM   # verify.sh's trap does not exit; this one does

step() { printf '\n== %s\n' "$1"; }
record() { results+=("$1|$2|$3"); }

# every run uses its own server port
export OD2_PORT=$(( 20000 + RANDOM % 20000 ))
while lsof -nP -iTCP:$OD2_PORT -sTCP:LISTEN >/dev/null 2>&1; do export OD2_PORT=$(( 20000 + RANDOM % 20000 )); done

# start a .command file in the user's GUI session without opening a Terminal window (same as verify.sh)
launch_game() {
  if [ -n "${OD2_VERIFY_LAUNCH:-}" ]; then ${=OD2_VERIFY_LAUNCH} $1 >/dev/null 2>&1 &
  elif launchctl asuser $(id -u) /usr/bin/true >/dev/null 2>&1; then launchctl asuser $(id -u) /bin/zsh $1 >/dev/null 2>&1 &
  else open $1
  fi
}

# Wait for the scenario's game. The launch file touches $done_marker when the game process exits, so a game that
# crashes at start-up (or finishes in a second) is noticed at once instead of after the start timeout. A game that
# never appears within start_wait, or is still running after start_wait + run_wait, is reported and killed (this
# run's scratch folder only). Other processes of the scenario (a multiplayer joiner) get 30 s to finish after it.
wait_run() {
  local i seen=0
  for ((i = 0; i < start_wait + run_wait; i++)); do
    [ -f "$done_marker" ] && break
    pgrep -f "$tmp/od2" >/dev/null && seen=1
    if [ $seen -eq 0 ] && [ $i -ge $start_wait ]; then
      echo "FAIL: the game never started within ${start_wait}s"
      return 1
    fi
    sleep 1
  done

  if [ ! -f "$done_marker" ]; then
    echo "REAPED: the game did not finish in ${run_wait}s (stuck?), killing it"
    pkill -f "$tmp/od2"; sleep 1; pkill -9 -f "$tmp/od2" 2>/dev/null
    return 1
  fi

  for ((i = 0; i < 30; i++)); do pgrep -f "$tmp/od2" >/dev/null || return 0; sleep 1; done

  echo "REAPED: another process of the scenario was still running 30s after the game, killing it"
  pkill -f "$tmp/od2"; sleep 1; pkill -9 -f "$tmp/od2" 2>/dev/null
  return 1
}

timed_step() {   # timed_step "title" command...   -> records PASS/FAIL
  local title=$1; shift
  local s=$SECONDS
  step "$title"
  if "$@"; then record "$title" PASS $((SECONDS - s)); else record "$title" FAIL $((SECONDS - s)); fail=1; fi
}

filter_build() { grep -v "ld: warning\|duplicate libraries" ; }

do_hygiene() { scripts/check_repo_hygiene.sh; }
do_build() {
  if [ -n "${SMOKE_BIN:-}" ]; then cp "$SMOKE_BIN" $tmp/od2 && chmod +x $tmp/od2; return; fi
  go build -o $tmp/od2 . 2>&1 | filter_build; [ ${pipestatus[1]} -eq 0 ]
}
do_vet()     { go vet ./... 2>&1 | grep -v "ld: warning\|duplicate libraries\|^# " ; [ ${pipestatus[1]} -eq 0 ]; }
do_tests()   { go test ./... 2>&1 | grep -v "ld: warning\|duplicate libraries\|no test files\|^# " | grep -v "^ok"; [ ${pipestatus[1]} -eq 0 ]; }
do_race()    { go test -race ./d2common/... ./d2core/... ./d2game/... 2>&1 | grep -v "ld: warning\|duplicate libraries\|no test files\|^# " | grep -v "^ok"; [ ${pipestatus[1]} -eq 0 ]; }

echo "smoke.sh: scratch $tmp, port $OD2_PORT"
echo "          D2S_SAMPLE_BODY=${D2S_SAMPLE_BODY:-<unset>}  D2_TABLES=${D2_TABLES:-<unset>}"

if [ -z "${SMOKE_SKIP_CHECKS:-}" ]; then
  timed_step "repo hygiene (no game files / decompiled code)" do_hygiene
fi
timed_step "build" do_build
if [ -z "${SMOKE_SKIP_CHECKS:-}" ]; then
  timed_step "go vet ./..." do_vet
  timed_step "unit tests (emulator goldens and oracle cross-checks; only failures are printed)" do_tests
  [ -n "${SMOKE_RACE:-}" ] && timed_step "race detector (d2common, d2core, d2game)" do_race
fi

if [ -n "${SMOKE_SKIP_GAMES:-}" ]; then
  echo; echo "(in-game scenarios skipped: SMOKE_SKIP_GAMES)"
elif [ -z "${D2S_SAMPLE_BODY:-}" ] || [ ! -f "$D2S_SAMPLE_BODY" ]; then
  echo; echo "FAIL: no sample save (set D2S_SAMPLE_BODY); the in-game scenarios cannot start from a fresh save"
  record "in-game scenarios" FAIL 0; fail=1
elif [ ! -x $tmp/od2 ]; then
  echo; echo "FAIL: no game binary (the build failed), skipping the in-game scenarios"
  record "in-game scenarios" FAIL 0; fail=1
else
  save=$tmp/save.d2s
  for s in $scenarios; do
    f=scripts/verify.d/$s.sh
    [ -f "$f" ] || { echo "FAIL: scenario file $f does not exist"; record "$s" FAIL 0; fail=1; continue; }

    unset -f scenario_env scenario_check 2>/dev/null
    scenario_name="$s"; scenario_warnings_ok=""; scenario_unmuted=""
    source "$f"

    s_start=$SECONDS
    ok=0
    for ((attempt = 0; attempt <= retry; attempt++)); do
      fail=0   # scenario_check and the checks below set fail=1; the verdict is taken from the recorded results
      step "$scenario_name$([ $attempt -gt 0 ] && echo " (attempt $((attempt + 1)))")"
      cp "$D2S_SAMPLE_BODY" "$save"            # every attempt starts from a fresh copy of the sample save
      n=$s
      cmd=$tmp/$n.command log=$tmp/$n.log
      done_marker=$tmp/$n.done
      rm -f $done_marker
      {
        echo '#!/bin/zsh'
        echo "export OD2_PORT=$OD2_PORT"
        echo "export OD2_AUTOGAME=\"$save\" OD2_AUTOEXIT=1"
        [ -n "${OD2_VERIFY_SOUND:-}" ] || [ -n "$scenario_unmuted" ] || echo "export OD2_AUTOTEST_MUTE=1"
        scenario_env
        echo "$tmp/od2 2>&1 | tee $log"
        echo "touch $done_marker"
      } > $cmd
      chmod +x $cmd; rm -f $log

      slot=$(./scripts/gameslot.sh acquire $$)
      if [ -z "$slot" ]; then   # never run a game without a slot
        echo "FAIL: could not get a game slot, not starting the game"
        fail=1
        continue
      fi
      launch_game $cmd
      wait_run || fail=1
      ./scripts/gameslot.sh release "$slot"

      sed 's/\x1b\[[0-9;]*m//g' $log > $log.txt 2>/dev/null
      scenario_check
      if [ -z "$scenario_warnings_ok" ] && grep -E "\[(ERROR|WARNING)\]|panic" $log.txt 2>/dev/null | grep -v "skipping missing" | grep -v "KILL giving up for now"; then
        echo "FAIL: warnings/errors in the $scenario_name log"; fail=1
      fi
      pkill -f "$tmp/od2" 2>/dev/null   # nothing of this scenario may outlive it

      if [ $fail -eq 0 ]; then ok=1; break; fi
      [ $attempt -lt $retry ] && echo "RETRY: $s failed; running it again from a fresh save"
    done

    if [ $ok -eq 1 ]; then record "$s" PASS $((SECONDS - s_start)); else record "$s" FAIL $((SECONDS - s_start)); fi
  done
fi

# the verdict is the recorded results (a scenario that passed on a retry counts as passed)
fail=0
for r in $results; do parts=("${(@s:|:)r}"); [ "${parts[2]}" = FAIL ] && fail=1; done

echo
echo "== smoke summary ($((SECONDS - t0))s)"
for r in $results; do
  parts=("${(@s:|:)r}")
  printf '  %-4s %5ss  %s\n' "${parts[2]}" "${parts[3]}" "${parts[1]:0:100}"
done
echo
[ $fail -eq 0 ] && echo "SMOKE PASSED" || { echo "SMOKE FAILED"; exit 1; }
