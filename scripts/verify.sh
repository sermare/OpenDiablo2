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
  open $cmd   # a GUI session is required; running the binary from a plain shell fails
  for i in {1..90}; do sleep 1; pgrep -f $tmp/od2 >/dev/null || break; done
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
export OD2_AUTOSCRIPT='wait:1;move:npc=Akara;wait:14;expect:log=NPC menu opened;panel:inventory;wait:1;panel:character;wait:1;panel:skills;wait:1;shot:$tmp/panels.png;wait:2;panel:close;exit'
$tmp/od2 2>&1 | tee $log
EOT
  chmod +x $cmd; rm -f $log
  open $cmd
  for i in {1..90}; do sleep 1; pgrep -f $tmp/od2 >/dev/null || break; done
  sed 's/\x1b\[[0-9;]*m//g' $log > $log.txt
  grep -E "AUTOSCRIPT|PANEL" $log.txt | cut -c1-300
  grep -q "AUTOSCRIPT RESULT PASS" $log.txt || { echo "FAIL: scripted scenario did not pass"; fail=1; }
  # the panels log what they show, so the imported hero can be checked without a screenshot
  grep -qE "PANEL character: level=[0-9]+ name=.* str=[0-9]+ dex=[0-9]+ vit=[0-9]+ ene=[0-9]+ hp=[0-9]+/[0-9]+ .* exp=[0-9]+ next=[0-9]+ " $log.txt || { echo "FAIL: no PANEL character line"; fail=1; }
  grep -qE "PANEL skills: class=.* unspent=[0-9]+ spent=[0-9]+ icons=" $log.txt || { echo "FAIL: no PANEL skills line"; fail=1; }
  grep -qE "PANEL skills active: left=.*\(id=[0-9]+,lvl=[0-9]+\) right=.*\(id=[0-9]+,lvl=[0-9]+\)" $log.txt || { echo "FAIL: no PANEL skills active line"; fail=1; }
  grep -qE "PANEL inventory: gold=[0-9]+ items=[0-9]+ worn=\[" $log.txt || { echo "FAIL: no PANEL inventory line"; fail=1; }
  [ -s $tmp/panels.png ] || { echo "FAIL: no screenshot from the shot: step"; fail=1; }
  if grep -E "\[(ERROR|WARNING)\]|panic" $log.txt | grep -v "skipping missing"; then echo "FAIL: warnings/errors in scripted log"; fail=1; fi
fi

if [ -n "${D2S_SAMPLE_BODY:-}" ]; then
  step "character select lists the imported .d2s (and takes a screenshot)"
  mkdir -p $tmp/d2s && cp "$D2S_SAMPLE_BODY" $tmp/d2s/Sample.d2s
  cmd=$tmp/charsel.command log=$tmp/charsel.log
  cat > $cmd <<EOT
#!/bin/zsh
export OD2_PORT=$OD2_PORT
export OD2_D2S_DIR="$tmp/d2s" OD2_AUTOSCREEN=charselect OD2_AUTOSHOT="$tmp/charselect.png"
export OD2_AUTOTEST_MUTE=1 OD2_AUTOEXIT=1
$tmp/od2 2>&1 | tee $log
EOT
  chmod +x $cmd; rm -f $log
  open $cmd
  for i in {1..60}; do sleep 1; pgrep -f $tmp/od2 >/dev/null || break; done
  sed 's/\x1b\[[0-9;]*m//g' $log > $log.txt
  grep -E "CHARSELECT|AUTOSHOT" $log.txt | cut -c1-200
  grep -qE "CHARSELECT slot=[0-9]+ name=.* class=[A-Za-z]+ level=[0-9]+ hardcore=(true|false) expansion=(true|false) ladder=(true|false) dead=(true|false) imported=true" $log.txt || { echo "FAIL: imported hero not listed"; fail=1; }
  [ -s $tmp/charselect.png ] || { echo "FAIL: no character select screenshot"; fail=1; }
  if grep -E "\[(ERROR|WARNING)\]|panic" $log.txt | grep -v "skipping missing"; then echo "FAIL: warnings/errors in character select log"; fail=1; fi
fi

echo
[ $fail -eq 0 ] && echo "ALL CHECKS PASSED" || { echo "SOME CHECKS FAILED"; exit 1; }
