#!/bin/zsh
# One-command verification for the macOS fork. Needs a Diablo II 1.14b + LoD
# install (see docs/macos-quickstart.md). Environment variables (all optional):
#   D2_TABLES        folder with extracted game tables (itemstatcost.bin, armor.txt, ...)
#   D2S_SAMPLE_BODY  a real .d2s save with a body; D2S_SAMPLE_BODY_JSON its expected parse
#   OD2_VERIFY_SAVE  a .d2s file to start in the game (default: $D2S_SAMPLE_BODY copied to a .d2s)
set -u
cd "${0:A:h}/.."

fail=0
step() { printf '\n== %s\n' "$1"; }

step "build"
go build -o /tmp/od2-verify . 2>&1 | grep -v "ld: warning" ; [ ${pipestatus[1]} -eq 0 ] || { echo "BUILD FAILED"; exit 1; }

step "unit tests"
go test ./... 2>&1 | grep -v "ld: warning\|no test files\|^# " | grep -v "^ok" ; [ ${pipestatus[1]} -eq 0 ] || fail=1
echo "(only failures are printed above)"

if [ -n "${D2S_SAMPLE_BODY:-}" ]; then
  step "real .d2s oracle tests"
  go test -v ./d2common/d2fileformats/d2s/ 2>&1 | grep -E "^(--- |FAIL|ok)" || fail=1
fi

if [ -n "${D2S_SAMPLE_BODY:-}" ]; then
  step "in-game autotest (imports the save, starts it, checks NPC menus)"
  save="${OD2_VERIFY_SAVE:-/tmp/od2-verify-save.d2s}"
  [ -f "$save" ] || cp "$D2S_SAMPLE_BODY" "$save"
  cmd=/tmp/od2-verify.command log=/tmp/od2-verify.log
  cat > $cmd <<EOT
#!/bin/zsh
export OD2_AUTOGAME="$save" OD2_AUTOMENU="Akara,Charsi,Gheed,Warriv,Kashya" OD2_AUTOMENU_CHOOSE=Talk
export OD2_AUTOTRADE="Akara,Charsi" OD2_AUTOTRADE_SEED=1 OD2_AUTOTRADE_LEVEL=8
export OD2_AUTOTEST_MUTE=1 OD2_AUTOEXIT=1
/tmp/od2-verify 2>&1 | tee $log
EOT
  chmod +x $cmd; rm -f $log
  open $cmd   # a GUI session is required; running the binary from a plain shell fails
  for i in {1..90}; do sleep 1; pgrep -f /tmp/od2-verify >/dev/null || break; done
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

echo
[ $fail -eq 0 ] && echo "ALL CHECKS PASSED" || { echo "SOME CHECKS FAILED"; exit 1; }
