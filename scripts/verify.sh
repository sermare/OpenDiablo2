#!/bin/zsh
# One-command verification for the macOS fork. Needs a Diablo II 1.14b + LoD
# install (see docs/macos-quickstart.md). Environment variables (all optional):
#   D2_TABLES        folder with extracted game tables (itemstatcost.bin, armor.txt, ...)
#   D2S_SAMPLE_BODY  a real .d2s save with a body; D2S_SAMPLE_BODY_JSON its expected parse
#   OD2_VERIFY_SAVE  a .d2s file to start in the game (default: $D2S_SAMPLE_BODY copied to a .d2s)
set -u
cd "${0:A:h}/.."

fail=0
# every run gets its own scratch folder and server port, so parallel runs (e.g. several agents) do not collide
tmp=$(mktemp -d /tmp/od2-verify.XXXXXX)
step() { printf '\n== %s\n' "$1"; }
# run_cmd <script.command> <logfile> <seconds>: start the game in a GUI session (open), wait until it has
# begun logging (otherwise the exit poll below can see "no process yet" and give up at once), then wait for
# the process to end.
# OD2_VERIFY_LAUNCH can name another launcher (for example "launchctl asuser 501 /bin/zsh") when Terminal is
# overloaded; it is run in the background and must start the .command file given as its last argument.
run_cmd() {
  if [ -n "${OD2_VERIFY_LAUNCH:-}" ]; then ${=OD2_VERIFY_LAUNCH} $1 >/dev/null 2>&1 &
  else open $1; fi
  for i in {1..60}; do [ -f $2 ] && break; sleep 1; done
  for i in {1..$3}; do sleep 1; pgrep -f $tmp/od2 >/dev/null || break; done
}

# every run uses its own server port so parallel runs (e.g. several agents) do not collide
export OD2_PORT=$(( 20000 + RANDOM % 20000 ))

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

if [ -n "${D2S_SAMPLE_BODY:-}" ]; then
  step "in-game autotest (imports the save, starts it, checks NPC menus)"
  save="${OD2_VERIFY_SAVE:-$tmp/save.d2s}"
  [ -f "$save" ] || cp "$D2S_SAMPLE_BODY" "$save"
  cmd=$tmp/run.command log=$tmp/run.log
  cat > $cmd <<EOT
#!/bin/zsh
export OD2_PORT=$OD2_PORT
export OD2_AUTOGAME="$save" OD2_AUTOMENU="Akara,Charsi,Gheed,Warriv,Kashya" OD2_AUTOMENU_CHOOSE=Talk
export OD2_AUTOTRADE="Akara,Charsi" OD2_AUTOTRADE_SEED=1 OD2_AUTOTRADE_LEVEL=8
export OD2_AUTOTEST_MUTE=1 OD2_AUTOEXIT=1
$tmp/od2 2>&1 | tee $log
EOT
  chmod +x $cmd; rm -f $log
  run_cmd $cmd $log 90
  sed 's/\x1b\[[0-9;]*m//g' $log > $log.txt
  grep -E "imported|equipment:|NPC menu opened" $log.txt | cut -c1-200
  grep -E "AUTOTRADE (buy|sell|repair)" $log.txt | cut -c1-200
  grep -qE "NPC menu opened: npc=\"Akara\"" $log.txt || { echo "FAIL: no Akara menu"; fail=1; }
  for v in Akara Charsi; do
    grep -qE "AUTOTRADE buy vendor=$v .*err=<nil>" $log.txt || { echo "FAIL: no scripted buy at $v"; fail=1; }
    grep -qE "AUTOTRADE sell vendor=$v .*err=<nil>" $log.txt || { echo "FAIL: no scripted sell at $v"; fail=1; }
  done
  grep -qE "AUTOTRADE repair vendor=Charsi .*err=<nil>" $log.txt || { echo "FAIL: no Charsi repair"; fail=1; }
  if grep -E "\[(ERROR|WARNING)\]|panic" $log.txt | grep -v "skipping missing"; then echo "FAIL: warnings/errors in log"; fail=1; fi
fi

if [ -n "${D2S_SAMPLE_BODY:-}" ]; then
  step "scripted scenario (walk to Akara, menu opens, inventory panel, exit)"
  save="${OD2_VERIFY_SAVE:-$tmp/save.d2s}"
  cmd=$tmp/script.command log=$tmp/script.log
  cat > $cmd <<EOT
#!/bin/zsh
export OD2_PORT=$OD2_PORT
export OD2_AUTOGAME="$save" OD2_AUTOTEST_MUTE=1 OD2_AUTOEXIT=1
export OD2_AUTOSCRIPT='wait:1;move:npc=Akara;wait:25;expect:log=NPC menu opened;panel:inventory;wait:1;panel:character;wait:1;panel:close;exit'
$tmp/od2 2>&1 | tee $log
EOT
  chmod +x $cmd; rm -f $log
  run_cmd $cmd $log 90
  sed 's/\x1b\[[0-9;]*m//g' $log > $log.txt
  grep -E "AUTOSCRIPT" $log.txt | cut -c1-200
  grep -q "AUTOSCRIPT RESULT PASS" $log.txt || { echo "FAIL: scripted scenario did not pass"; fail=1; }
  if grep -E "\[(ERROR|WARNING)\]|panic" $log.txt | grep -v "skipping missing"; then echo "FAIL: warnings/errors in scripted log"; fail=1; fi
fi

if [ -n "${D2S_SAMPLE_BODY:-}" ]; then
  step "waypoint and portal (walk to the Rogue Encampment waypoint, panel lists the act 1 entries, portal level change)"
  save="${OD2_VERIFY_SAVE:-$tmp/save.d2s}"
  cmd=$tmp/wp.command log=$tmp/wp.log
  cat > $cmd <<EOT
#!/bin/zsh
export OD2_PORT=$OD2_PORT
export OD2_AUTOGAME="$save" OD2_AUTOTEST_MUTE=1 OD2_AUTOEXIT=1
export OD2_AUTOSCRIPT='wait:1;use:Waypoint;expect:log=WAYPOINT PANEL act=1 level=1 rows=9 active=9 enabled=1;expect:log=1:Rogue Encampment:on;expect:log=3:Cold Plains:grey;expect:log=35:Catacombs Level 2:grey;say:spawnportal 1;use:Portal;expect:level=1;expect:log=LEVEL CHANGE from=1 to=1;wait:1;exit'
$tmp/od2 2>&1 | tee $log
EOT
  chmod +x $cmd; rm -f $log
  run_cmd $cmd $log 90
  sed 's/\x1b\[[0-9;]*m//g' $log > $log.txt
  grep -E "AUTOSCRIPT level=|WAYPOINT PANEL|LEVEL CHANGE|AUTOSCRIPT RESULT" $log.txt | cut -c1-260
  grep -q "AUTOSCRIPT RESULT PASS" $log.txt || { echo "FAIL: waypoint/portal scenario did not pass"; fail=1; }
  if grep -E "\[(ERROR|WARNING)\]|panic" $log.txt | grep -v "skipping missing"; then echo "FAIL: warnings/errors in waypoint log"; fail=1; fi
fi

if [ -n "${D2S_SAMPLE_BODY:-}" ]; then
  step "waypoint activation persists (clear the town waypoint bit, activate it at the object, exported .d2s has it again)"
  save=$tmp/wpsave.d2s; cp "$D2S_SAMPLE_BODY" $save
  wb=$tmp/wpwriteback; mkdir -p $wb
  cmd=$tmp/wpsave.command log=$tmp/wpsave.log
  cat > $cmd <<EOT
#!/bin/zsh
export OD2_PORT=$OD2_PORT
export OD2_AUTOGAME="$save" OD2_D2S_WRITEBACK="$wb" OD2_AUTOTEST_MUTE=1 OD2_AUTOEXIT=1
export OD2_AUTOSCRIPT='wait:1;say:setwaypoint 1 0;wait:1;use:Waypoint;expect:log=WAYPOINT activated level=1;wait:1;exit'
$tmp/od2 2>&1 | tee $log
EOT
  chmod +x $cmd; rm -f $log
  run_cmd $cmd $log 90
  sed 's/\x1b\[[0-9;]*m//g' $log > $log.txt
  grep -E "WAYPOINT (saved|activated)|AUTOSCRIPT RESULT" $log.txt | cut -c1-200
  grep -q "AUTOSCRIPT RESULT PASS" $log.txt || { echo "FAIL: waypoint persistence scenario did not pass"; fail=1; }
  grep -q "WAYPOINT saved .*level=1 bit=0 active=false .*mask=0x7ffffffffe" $log.txt || { echo "FAIL: cleared bit not saved"; fail=1; }
  last=$(grep "D2S EXPORT reparse" $log.txt | tail -1)
  case "$last" in *waypoints=0x7fffffffff*) ;; *) echo "FAIL: exported .d2s lacks the activated waypoint: $last"; fail=1;; esac
  if grep -E "\[(ERROR|WARNING)\]|panic" $log.txt | grep -v "skipping missing"; then echo "FAIL: warnings/errors in waypoint persistence log"; fail=1; fi
fi

if [ -n "${D2S_SAMPLE_BODY:-}" ]; then
  step "maze travel (OD2_REALMAPS=1: waypoint to Jail Level 1, open and close a door, portal back to town)"
  save="${OD2_VERIFY_SAVE:-$tmp/save.d2s}"
  cmd=$tmp/maze.command log=$tmp/maze.log
  cat > $cmd <<EOT
#!/bin/zsh
export OD2_PORT=$OD2_PORT
export OD2_AUTOGAME="$save" OD2_AUTOTEST_MUTE=1 OD2_AUTOEXIT=1 OD2_REALMAPS=1
export OD2_AUTOSCRIPT='wait:1;use:Waypoint;waypoint:29;expect:level=29;use:Door;use:Door;wait:6;say:spawnportal 1;use:Portal;expect:level=1;exit'
$tmp/od2 2>&1 | tee $log
EOT
  chmod +x $cmd; rm -f $log
  run_cmd $cmd $log 120
  sed 's/\x1b\[[0-9;]*m//g' $log > $log.txt
  grep -E "AUTOSCRIPT level=|LEVEL CHANGE|OBJECT door|AUTOSCRIPT RESULT" $log.txt | cut -c1-260
  grep -q "AUTOSCRIPT RESULT PASS" $log.txt || { echo "FAIL: maze travel scenario did not pass"; fail=1; }
  grep -q "OBJECT door .*open=true .*blocks=false" $log.txt || { echo "FAIL: door did not open"; fail=1; }
  grep -q "OBJECT door .*open=false .*blocks=true" $log.txt || { echo "FAIL: door did not close and block"; fail=1; }
  # the maze renderer logs tile errors of its own; only a crash counts here
  if grep -E "panic" $log.txt; then echo "FAIL: panic in maze travel log"; fail=1; fi
fi

if [ -n "${D2S_SAMPLE_BODY:-}" ]; then
  step "autosave (OD2_AUTOSAVE=1: set gold in game, exit, exported .d2s re-parses)"
  save=$tmp/autosave.d2s; cp "$D2S_SAMPLE_BODY" $save
  wb=$tmp/writeback; mkdir -p $wb   # keeps the exported file out of the Saves folder
  cmd=$tmp/autosave.command log=$tmp/autosave.log
  cat > $cmd <<EOT
#!/bin/zsh
export OD2_PORT=$OD2_PORT
export OD2_AUTOGAME="$save" OD2_AUTOSAVE=1 OD2_D2S_WRITEBACK="$wb"
export OD2_AUTOTEST_MUTE=1 OD2_AUTOEXIT=1
$tmp/od2 2>&1 | tee $log
EOT
  chmod +x $cmd; rm -f $log
  run_cmd $cmd $log 90
  sed 's/\x1b\[[0-9;]*m//g' $log > $log.txt
  grep -E "D2S EXPORT|AUTOSCRIPT RESULT" $log.txt | cut -c1-300
  grep -q "AUTOSCRIPT RESULT PASS" $log.txt || { echo "FAIL: autosave script did not pass"; fail=1; }
  grep -q "D2S EXPORT reparse: .*gold=31337 .*checksum=ok" $log.txt || { echo "FAIL: exported .d2s does not show the new gold"; fail=1; }
  ls $wb/*.d2s >/dev/null 2>&1 || { echo "FAIL: no exported .d2s in $wb"; fail=1; }
  if grep -E "\[(ERROR|WARNING)\]|panic" $log.txt | grep -v "skipping missing"; then echo "FAIL: warnings/errors in autosave log"; fail=1; fi
fi

echo
[ $fail -eq 0 ] && echo "ALL CHECKS PASSED" || { echo "SOME CHECKS FAILED"; exit 1; }
