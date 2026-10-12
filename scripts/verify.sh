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
launch_now() {
  if [ -n "${OD2_VERIFY_LAUNCH:-}" ]; then ${=OD2_VERIFY_LAUNCH} $1 >/dev/null 2>&1 &
  elif launchctl asuser $(id -u) /usr/bin/true >/dev/null 2>&1; then launchctl asuser $(id -u) /bin/zsh $1 >/dev/null 2>&1 &
  else open -g $1 # -g: do not bring the app to the front
  fi
}
# While a scenario's command file is being generated (defer_launch=1) the scenario's extra processes (joiners of the
# multi-process scenarios) are NOT started at once: they would run before the host got its game slot, and a joiner
# that waits for the host's server line for 300 s gave up on a loaded machine (slot waits of minutes with
# OD2_MAX_GAMES=3), so the join never happened (false red 9d-idle-kick / 96 / 9d-party-trade ...).
# They start right after the host's own launch (flush_deferred), or when wait_run is called (scenarios that run a
# helper game to completion inside scenario_env).
typeset -ga deferred_cmds; deferred_cmds=()
defer_launch=""
launch_game() {
  if [ -n "$defer_launch" ]; then deferred_cmds+=("$1"); else launch_now $1; fi
}
flush_deferred() {
  local c
  for c in $deferred_cmds; do launch_now $c; done
  deferred_cmds=()
}

# the launcher returns before the game starts (slowly, on a busy machine): wait until the game process appears, then
# until it exits. $1 (optional): the .command file of the game; its `.done` marker (written when the game process
# ends, holding its exit status) is the authority on "finished". A pgrep that misses once on a loaded machine
# (or a process that is mid-exec) no longer counts as "finished": three misses in a row are needed.
wait_run() {
  local i miss=0 done_f="${1:+$1.done}"
  flush_deferred
  for i in {1..240}; do
    [ -n "$done_f" ] && [ -f "$done_f" ] && break
    pgrep -f "$tmp/od2" >/dev/null && break
    # a game that crashed or finished instantly is already gone but left its log: do not wait the full start timeout
    [ $i -gt 5 ] && [ -s "${log:-/nonexistent}" ] && { sleep 3; pgrep -f "$tmp/od2" >/dev/null || return; break; }
    sleep 1
  done
  for i in {1..${scenario_timeout:-420}}; do
    [ -n "$done_f" ] && [ -f "$done_f" ] && break
    if pgrep -f "$tmp/od2" >/dev/null; then miss=0; else miss=$((miss+1)); [ $miss -ge 3 ] && [ -z "$done_f" ] && break; [ $miss -ge 10 ] && break; fi
    sleep 1
  done
  # the host is done: other games of this run (joiners) get a grace period to finish their own scripts and exports
  if [ -n "$done_f" ]; then for i in {1..90}; do pgrep -f "$tmp/od2" >/dev/null || break; sleep 1; done; fi
  # a game still alive now is stuck (e.g. on the title screen): kill it, never leave windows behind.
  # The pattern is this run's private scratch dir, so other runs/agents/the user's own games are untouched.
  if pgrep -f "$tmp/od2" >/dev/null; then reaped=1; echo "REAPED: game of this scenario did not finish (stuck?), killing it"; pkill -f "$tmp/od2"; sleep 1; pkill -9 -f "$tmp/od2" 2>/dev/null; fi
}
# every run gets its own scratch folder and server port, so parallel runs (e.g. several agents) do not collide
tmp=$(mktemp -d /tmp/od2-verify.XXXXXX)
# OD2_VERIFY_TMPFILE: write the scratch folder's path there (scripts/verify_classes.sh reads the game logs from it)
[ -n "${OD2_VERIFY_TMPFILE:-}" ] && echo "$tmp" > "$OD2_VERIFY_TMPFILE"
cleanup_games() { pkill -f "$tmp/od2" 2>/dev/null; pkill -f "$tmp/[0-9a-z-]*\.command" 2>/dev/null; }
# A run leaves ~77 MB in its scratch folder; 1400 runs filled the disk. A run that passed removes its own folder
# (a failed one keeps it for triage). Kept when OD2_VERIFY_KEEP=1 or OD2_VERIFY_TMPFILE is set (verify_classes.sh
# reads the game logs from it after this script exits). Folders of other runs older than 6 h are pruned too: the
# longest scenario is limited to 20 min, so nothing that old can still be running.
cleanup_run_dir() {
  cleanup_games
  (( $+functions[release_ports] )) && release_ports
  # only the `open` fallback leaves Terminal windows behind; close the idle ones (never D2MOO / claude windows)
  [ -z "${OD2_VERIFY_LAUNCH:-}" ] && ! launchctl asuser $(id -u) /usr/bin/true >/dev/null 2>&1 && scripts/close_idle_terminals.sh >/dev/null 2>&1
  case "$tmp" in /tmp/od2-verify.??????) ;; *) return ;; esac
  [ "${fail:-1}" = 0 ] && [ -z "${OD2_VERIFY_KEEP:-}" ] && [ -z "${OD2_VERIFY_TMPFILE:-}" ] && rm -rf "$tmp"
  find /tmp/ -maxdepth 1 -name 'od2-verify.??????' -type d -mmin +360 -exec rm -rf {} + 2>/dev/null
  return 0
}
trap cleanup_run_dir EXIT
trap 'cleanup_games; exit 130' INT
trap 'cleanup_games; exit 143' TERM
step() { printf '\n== %s\n' "$1"; }

# every run uses its own server port so parallel runs (e.g. several agents) do not collide
# Ports are reserved in /tmp/od2-ports (mkdir is atomic; a reservation of a dead owner or older than 2 h is reclaimed),
# so two runs can never pick the same port in the gap between "is it free" and "the server binds it". Every scenario
# attempt gets a FRESH port (pick_port): the previous scenario's server (or a joiner that is still reconnecting with
# OD2_JOIN_RETRY) can otherwise still hold or hit the old one.
mkdir -p /tmp/od2-ports
typeset -ga my_ports; my_ports=()
pick_port() {
  local p d o i
  for i in {1..200}; do
    p=$(( 20000 + RANDOM % 20000 )); d=/tmp/od2-ports/$p
    if [ -d $d ]; then
      o=$(cat $d/owner 2>/dev/null)
      if { [ -n "$o" ] && ! kill -0 $o 2>/dev/null; } || [ -n "$(find $d -maxdepth 0 -mmin +120 2>/dev/null)" ]; then rm -rf $d; else continue; fi
    fi
    lsof -nP -iTCP:$p >/dev/null 2>&1 && continue # listening or connected (a reservation covers the rest)
    mkdir $d 2>/dev/null || continue
    echo $$ > $d/owner; my_ports+=($p); export OD2_PORT=$p; return 0
  done
  echo "VERIFY ABORTED: no free OD2_PORT found"; exit 1
}
release_ports() { local p; for p in $my_ports; do rm -rf /tmp/od2-ports/$p; done; my_ports=(); }
pick_port

# a verify without the real data silently skipped the scenario loop (false green): refuse to run
for v in D2S_SAMPLE_BODY D2S_SAMPLE_BODY_JSON D2_TABLES; do
  [ -n "${(P)v:-}" ] || { echo "VERIFY ABORTED: $v is unset; without it the real-data checks and the in-game scenarios would be skipped (false green)"; exit 1; }
done
scenarios_run=0

if [ -z "${SKIP_HYGIENE:-}" ]; then # (scripts/flake_loop.sh skips it: it takes minutes on a loaded machine and the loop repeats a run)
step "repo hygiene (no game files / decompiled code)"
scripts/check_repo_hygiene.sh || { echo "REPO HYGIENE FAILED"; exit 1; }
fi

step "build"
if [ -n "${OD2_VERIFY_BIN:-}" ]; then # a prebuilt binary (scripts/flake_loop.sh builds once for a whole loop)
  cp -f "$OD2_VERIFY_BIN" $tmp/od2 || { echo "BUILD FAILED (cannot copy $OD2_VERIFY_BIN)"; exit 1; }
else
go build -o $tmp/od2 . 2>&1 | grep -v "ld: warning" ; [ ${pipestatus[1]} -eq 0 ] || { echo "BUILD FAILED"; exit 1; }
fi

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
if [ -n "$game_dir" ] && [ -f "$game_dir/d2data.mpq" ] && [ -z "${SKIP_BUDGET:-}" ]; then
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
#   scenario_turbo=1         (optional) run under OD2_TURBO=1 by default (OD2_TURBO=0 overrides)
#   scenario_timeout=<seconds> (optional) wall-clock limit before the game is reaped (default 420; long playthroughs)
#   scenario_unmuted=1       (optional) play real audio (no OD2_AUTOTEST_MUTE); OD2_VERIFY_SOUND=1 does it for all
#   scenario_timeout=<seconds> (optional) wall-clock limit of the game window (default 420; the reaper kills at 8 minutes)
#   scenario_warnings_ok=1   (optional) do not fail on [ERROR]/[WARNING] lines
# Adding a scenario = adding one small file; no edits to this runner are needed.
# A GUI session is required (the game is started with `open`).
# ---------------------------------------------------------------------------
if [ -n "${D2S_SAMPLE_BODY:-}" ]; then
  save="${OD2_VERIFY_SAVE:-$tmp/save.d2s}"
  [ -f "$save" ] || cp "$D2S_SAMPLE_BODY" "$save"
  # A private config folder per run unless OD2_VERIFY_CONFIG_DIR is set. The games import the sample .d2s into
  # <config>/Saves, and an import of the same character (name + map seed) reuses the same .od2 file: parallel runs
  # (verify_parallel.sh) shared one file, so a hero another game had moved to act 2 started "in Lut Gholein" in
  # 9d-act1-sample-hero (batch 16 flake: LEVEL CHANGE from=1 to=40 via=load).
  default_cfg="${OD2_VERIFY_CONFIG_SRC:-$HOME/Library/Application Support/OpenDiablo2/config.json}"
  if [ -z "${OD2_VERIFY_CONFIG_DIR:-}" ]; then
    if [ -f "$default_cfg" ]; then
      mkdir -p $tmp/cfg && cp "$default_cfg" $tmp/cfg/config.json && export OD2_VERIFY_CONFIG_DIR=$tmp/cfg
    fi
  elif [ ! -f "$OD2_VERIFY_CONFIG_DIR/config.json" ]; then
    # A private config dir WITHOUT config.json makes the game start its first-run setup and hang (w42-inferno-hold
    # logs: "could not load file stream config.json ... starting first-run setup"): fill it from the user's config.
    if [ -f "$default_cfg" ]; then
      mkdir -p "$OD2_VERIFY_CONFIG_DIR" && cp "$default_cfg" "$OD2_VERIFY_CONFIG_DIR/config.json" \
        && echo "(OD2_VERIFY_CONFIG_DIR had no config.json: copied $default_cfg into it)"
    else
      echo "VERIFY ABORTED: OD2_VERIFY_CONFIG_DIR=$OD2_VERIFY_CONFIG_DIR has no config.json and there is none to copy ($default_cfg); the game would hang in first-run setup"; exit 1
    fi
  fi
  # make_hero <out.d2s> for the level 94 playthroughs; OD2_HERO=barb picks the generated Barbarian (default: the Sorceress)
  source scripts/verify.d/lib/hero.sh

  scen_run=0 scen_pass=0
  for f in scripts/verify.d/*.sh(N); do
    unset -f scenario_env scenario_check 2>/dev/null; scenario_name="${f:t}"; scenario_warnings_ok=""; scenario_unmuted=""; scenario_realtime=""; scenario_timeout=""; scenario_turbo=""
    source "$f"
    # OD2_VERIFY_ONLY=<glob> (e.g. "83-*") runs only the scenarios whose file name matches
    [ -n "${OD2_VERIFY_ONLY:-}" ] && [[ ${f:t} != ${~OD2_VERIFY_ONLY} && ${f:t:r} != ${~OD2_VERIFY_ONLY} ]] && continue
    fail_before=$fail
    scenarios_run=$((scenarios_run+1))
    scen_run=$((scen_run + 1))
    attempt=0 infra=0
    while [ $attempt -lt 2 ]; do
      attempt=$((attempt + 1))
      fail=$fail_before
      step "$scenario_name"
      # a fresh server port per attempt (the previous scenario's server / a reconnecting joiner may still use the old one)
      pick_port
      # the private config folder must still hold config.json (another job or a cleanup may have removed it)
      [ -n "${OD2_VERIFY_CONFIG_DIR:-}" ] && [ ! -f "$OD2_VERIFY_CONFIG_DIR/config.json" ] && [ -f "$default_cfg" ] && { mkdir -p "$OD2_VERIFY_CONFIG_DIR"; cp "$default_cfg" "$OD2_VERIFY_CONFIG_DIR/config.json"; }
      defer_launch=1; deferred_cmds=()
      # saves are written back to the .d2s they were loaded from: every run starts from a fresh copy
      if [ -z "${OD2_VERIFY_SAVE:-}" ]; then cp -f "$D2S_SAMPLE_BODY" "$save"; rm -f "$save.bak"; fi
      n=${f:t:r}
      cmd=$tmp/$n.command log=$tmp/$n.log; rm -f $log
      {
        echo '#!/bin/zsh'
        echo "export OD2_PORT=$OD2_PORT"
        # logical screen size of every scenario (docs/DISPLAY.md): 1512x982 (the owner's MacBook) unless OD2_DISPLAY
        # is set; a scenario pinned to the legacy 800x600 layout exports OD2_DISPLAY=800x600 in its scenario_env
        echo "export OD2_DISPLAY=${OD2_DISPLAY:-1512x982}"
        # OD2_QUIET=1 (default for verify_parallel.sh): the window opens behind the others, never takes the focus and
        # ignores the real mouse/keyboard (scripted input is injected, so scenarios are unchanged); see docs/DISPLAY.md
        [ -n "${OD2_QUIET:-}" ] && echo "export OD2_QUIET=$OD2_QUIET"
        echo "export OD2_AUTOGAME=\"$save\" OD2_AUTOEXIT=1"
        # muted unless OD2_VERIFY_SOUND=1 or the scenario sets scenario_unmuted=1 (real audio, uses the sound device)
        [ -n "${OD2_VERIFY_SOUND:-}" ] || [ -n "$scenario_unmuted" ] || echo "export OD2_AUTOTEST_MUTE=1"
        # game clock x4 (OD2_AUTOSPEED) unless the scenario needs real time (perf, multiplayer); OD2_VERIFY_SPEED=1 turns it off
        [ -n "$scenario_realtime" ] || [ "${OD2_VERIFY_SPEED:-8}" = 1 ] || echo "export OD2_AUTOSPEED=${OD2_VERIFY_SPEED:-8}"
        # OD2_VERIFY_CONFIG_DIR: a private config folder (a copy of a valid config.json) instead of the user's own;
        # a scenario that needs its own OD2_CONFIG_DIR overrides it below
        [ -n "${OD2_VERIFY_CONFIG_DIR:-}" ] && echo "export OD2_CONFIG_DIR=\"$OD2_VERIFY_CONFIG_DIR\""
        # OD2_TURBO=1: virtual 25 Hz clock, many ticks per frame, draw every Nth frame (see d2common/d2util/turbo.go)
        # a scenario with scenario_turbo=1 (passed twice under turbo) runs turbo by default; OD2_TURBO=0 turns it off
        turbo_v="${OD2_TURBO:-$scenario_turbo}"
        [ -n "$turbo_v" ] && [ "$turbo_v" != 0 ] && [ -z "$scenario_realtime" ] && echo "export OD2_TURBO=$turbo_v"
        scenario_env
        # OD2_VERIFY_EXTRA_ENV: extra `export` words for the game (e.g. profiling: "OD2_PPROF_CPU=/tmp/x.prof OD2_AUTOPERF=1")
        [ -n "${OD2_VERIFY_EXTRA_ENV:-}" ] && echo "export $OD2_VERIFY_EXTRA_ENV"
        echo "$tmp/od2 2>&1 | tee $log"
        # the done marker (with the game's exit status) is what wait_run waits for; no pgrep guessing
        echo "echo \${pipestatus[1]} > $cmd.done"
      } > $cmd
      defer_launch=""
      chmod +x $cmd; rm -f $log $cmd.done
      slot=$(./scripts/gameslot.sh acquire $$)
      t_start=$SECONDS
      launch_game $cmd
      flush_deferred   # the joiners of the multi-process scenarios start only now that the host has its slot
      reaped=""
      wait_run $cmd
      echo "SCENARIO TIME ${n} ${turbo_v:+turbo=$turbo_v }attempt=$attempt wall=$((SECONDS - t_start))s"
      ./scripts/gameslot.sh release $slot
      sed 's/\x1b\[[0-9;]*m//g' $log > $log.txt
      # Infrastructure failures (not the scenario's fault) do not use up an attempt: the game never started
      # (display service error under quiet mode, empty config dir -> first-run setup, launcher failure) or was
      # killed from outside (exit status >= 128 / vanished without a script result). Re-run up to 3 times.
      infra_reason=""
      game_status=$(cat $cmd.done 2>/dev/null)
      if [ ! -s $log ]; then infra_reason="no game log (launcher or display service failed)"
      elif grep -q "starting first-run setup\|could not load file stream config.json" $log.txt; then
        infra_reason="config.json missing in OD2_CONFIG_DIR (game started its first-run setup)"
        [ -n "${OD2_VERIFY_CONFIG_DIR:-}" ] && [ ! -f "$OD2_VERIFY_CONFIG_DIR/config.json" ] && [ -f "$default_cfg" ] && { mkdir -p "$OD2_VERIFY_CONFIG_DIR"; cp "$default_cfg" "$OD2_VERIFY_CONFIG_DIR/config.json"; }
      elif ! grep -q "d2data.mpq" $log.txt; then infra_reason="game never loaded its data ($(grep -m1 'PlatformError' $log.txt | cut -c1-120))"
      elif [ -z "$reaped" ] && grep -q "AUTOSCRIPT step" $log.txt && ! grep -q "AUTOSCRIPT RESULT" $log.txt && { [ -z "$game_status" ] || [ "$game_status" -ge 128 ] 2>/dev/null; }; then
        infra_reason="game process vanished mid-script (exit status ${game_status:-unknown}; killed from outside?)"
      fi
      if [ -n "$infra_reason" ] && [ $infra -lt 3 ]; then
        infra=$((infra + 1))
        echo "INFRA: $infra_reason; relaunching without using up the attempt ($infra/3)"
        flake_dir="${OD2_VERIFY_FLAKE_DIR:-/tmp/od2-flake-logs}"; mkdir -p $flake_dir
        cp -f $log.txt $flake_dir/${n}.$$.infra${infra}.txt 2>/dev/null
        cleanup_games; sleep 5; pick_port
        attempt=$((attempt - 1)); continue
      fi
      scenario_check
      if [ -z "$scenario_warnings_ok" ] && grep -E "\[(ERROR|WARNING)\]|panic" $log.txt | grep -v "skipping missing" | grep -v "KILL giving up for now" | grep -v "D2S export: container item" | grep -v "D2S export: worn item .* has no rolled values" | grep -v "D2S export: merc item"; then
        echo "FAIL: warnings/errors in the $scenario_name log"; fail=1
      fi
      [ $fail -eq $fail_before ] && break
      # keep the failed first attempt's log: the retry overwrites $log, and a flake that passes on retry
      # would otherwise leave no evidence of why it failed (batch 16 had four such flakes and no logs)
      if true; then # every failed attempt keeps its log (the second attempt's was lost: w22 86-class-skills)
        # (outside $tmp: a run that passed removes its scratch folder)
        flake_dir="${OD2_VERIFY_FLAKE_DIR:-/tmp/od2-flake-logs}"; mkdir -p $flake_dir
        flake_log=$flake_dir/${n}.$$.attempt${attempt}.txt
        cp -f $log.txt $flake_log 2>/dev/null
        for jl in $tmp/*.log(N); do [ "$jl" = "$log" ] || { [ "$jl" -nt "$cmd" ] && sed "s/\x1b\[[0-9;]*m//g" $jl > $flake_dir/${n}.$$.attempt${attempt}.${jl:t}.txt 2>/dev/null; }; done # the joiners' logs too
        echo "(attempt $attempt log kept in $flake_log; its last failure lines:)"
        grep -E "AUTOSCRIPT .*FAIL|DEATH|CAST start .*ok=false|CAST do .*ok=false|AUTOCAST .*refused|\[(ERROR|WARNING)\]|panic|LEVEL CHANGE" $log.txt | tail -12 | cut -c1-220
      fi
      [ $attempt -eq 1 ] && echo "RETRY: $scenario_name failed once; running it again (scenarios are timing sensitive on a loaded machine)"
    done
    [ $fail -eq $fail_before ] && scen_pass=$((scen_pass + 1))
  done
  echo; echo "SCENARIOS run=$scen_run passed=$scen_pass failed=$((scen_run - scen_pass))"
fi

echo
echo "SCENARIOS RUN: $scenarios_run"
if [ $scenarios_run -eq 0 ]; then echo "VERIFY FAILED: the scenario loop ran 0 scenarios (D2S_SAMPLE_BODY unset or OD2_VERIFY_ONLY matched nothing)"; exit 1; fi
[ $fail -eq 0 ] && echo "ALL CHECKS PASSED" || { echo "SOME CHECKS FAILED"; exit 1; }
