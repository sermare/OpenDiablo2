scenario_name="w48 gold dialogs: stash deposit/withdraw with the carry and stash limits, drop gold, the modal vendor reply dialog"
scenario_env() {
  echo "export OD2_AUTOSCRIPT='wait:2;say:setgold 5000;say:goldmove stash 100;say:goldmove deposit 3000;say:goldmove withdraw 500;say:goldmove drop 700;say:goldmove withdraw 999999;say:goldmove deposit 99999;say:goldmove stash 2499000;say:goldmove deposit 1500;say:goldmove msg 12;wait:1;exit'"
}
scenario_check() {
  grep -E "GOLDMOVE|vendor dialog|vendor message|AUTOGROUND drop gold" $log.txt | cut -c1-200
  grep -qE "GOLDMOVE deposit amount=3000 ok=true gold=2000 stash=3100" $log.txt || { echo "FAIL: deposit"; fail=1; }
  grep -qE "GOLDMOVE withdraw amount=500 ok=true gold=2500 stash=2600" $log.txt || { echo "FAIL: withdraw"; fail=1; }
  grep -qE "GOLDMOVE drop amount=700 ok=true gold=1800" $log.txt || { echo "FAIL: drop"; fail=1; }
  grep -q "AUTOGROUND drop gold amount=700" $log.txt || { echo "FAIL: no pile on the ground"; fail=1; }
  grep -qE "GOLDMOVE withdraw amount=999999 ok=false gold=1800 stash=2600" $log.txt || { echo "FAIL: over-withdraw was not refused"; fail=1; }
  grep -qE "GOLDMOVE deposit amount=99999 ok=false gold=1800" $log.txt || { echo "FAIL: deposit above the purse was not refused"; fail=1; }
  # stash 2499000: only 1000 fits, the rest stays in the purse
  grep -qE "GOLDMOVE deposit amount=1500 ok=true gold=800 stash=2500000" $log.txt || { echo "FAIL: stash cap"; fail=1; }
  grep -qE "vendor dialog open=true text=\"You do not have enough gold for that.\"" $log.txt || { echo "FAIL: modal reply dialog"; fail=1; }
  go test ./d2core/d2inventory/ -run "Gold" 2>&1 | grep -v "ld: warning\|^# " | grep -v "^ok" && fail=1
  go test ./d2common/d2trade/ -run "Tome|ResultMessage" 2>&1 | grep -v "ld: warning\|^# " | grep -v "^ok" && fail=1
}
