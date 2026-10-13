scenario_name="belt state potions (antidote joins its column by shift-click, a stamina potion without a column is refused by shift-click but takes a clicked-in cell, Q drinks them: poison resist +50 on the character screen, stamina refilled)"
scenario_realtime=1
scenario_timeout=300
scenario_warnings_ok=1 # setbelt takes the belt off the hero: the save export warns that the worn belt is kept in the file
bs=$tmp/beltstates
# belt cell centre in the 800x600 column: x = 437 + 31*col, y = 576 - 32*row (the front row is the bottom one)
_cx() { echo $((437 + 31 * $1)); }
_cy() { echo $((576 - 32 * $1)); }
scenario_env() {
  mkdir -p $bs/s94 $bs/wb94; rm -f $bs/s94/*.d2s(N) $bs/wb94/*.d2s(N)
  if [ -n "${D2_TABLES:-}" ] && go run scripts/d2s-revive.go "$D2S_SAMPLE_BODY" $bs/s94/Hero.d2s "$D2_TABLES" >/dev/null 2>&1; then
    echo "export OD2_AUTOGAME=\"$bs/s94/Hero.d2s\" OD2_D2S_WRITEBACK=\"$bs/wb94\" OD2_AUTOSPEED=3"
  fi
  echo 'export OD2_REALMAPS=1 OD2_AUTOLEVEL=2'
  local s='expect:level=2;press:I;until:inventory=true,60'
  s="$s;say:setbelt lbl;say:clearinv"
  # first antidote: a real click on the potion and on belt cell 0 (the direct drop has no column rule)
  s="$s;say:giveitem yps;wait:0.3;click:left@ui:694,416;wait:0.4;click:left@ui:$(_cx 0),$(_cy 0);wait:0.4"
  # second antidote: shift-click from the inventory goes behind the first (column 0, row 1)
  s="$s;say:giveitem yps;wait:0.3;click:left+shift@ui:694,416;wait:0.5;say:beltlist"
  # a stamina potion has no column of its own and autobelt 0: the shift-click leaves it in the inventory
  s="$s;say:giveitem vps;wait:0.3;click:left+shift@ui:694,416;wait:0.5;say:beltlist"
  # the refused shift-click leaves it on the cursor: dropping into front cell 1 puts it in
  s="$s;click:left@ui:$(_cx 1),$(_cy 0);wait:0.4;say:beltlist"
  s="$s;press:I;wait:0.4;panel:character;wait:0.5;panel:close"
  # Q drinks column 0 (the antidote), then the second one, then the stamina potion
  s="$s;press:Q;wait:1.5;wait:1;panel:character;wait:0.5;panel:close"
  s="$s;press:Q;wait:1.5;press:Q;wait:1.5;wait:0.5;say:beltlist;exit"
  echo "export OD2_AUTOSCRIPT='$s'"
}
scenario_check() {
  grep -E "BELT contents|BELT drink|AUTOSCRIPT RESULT|AUTOSCRIPT step [0-9]* FAIL" $log.txt | cut -c1-220 | head -30
  grep -q "AUTOSCRIPT RESULT PASS" $log.txt || { echo "FAIL: the script did not pass"; fail=1; }
  grep -q "BELT contents n=2 \[yps@a yps@e\]" $log.txt || { echo "FAIL: the second antidote did not join column 0 (cell e)"; fail=1; }
  grep -q "BELT contents n=3 \[yps@a vps@b yps@e\]" $log.txt || { echo "FAIL: the stamina potion did not take front cell 1 by a click"; fail=1; }
  [ "$(grep -c 'BELT contents n=2 \[yps@a yps@e\]' $log.txt)" -ge 2 ] || { echo "FAIL: the shift-click placed the stamina potion although no column held one"; fail=1; }
  local p1 p2
  p1=$(grep "PANEL character:.*poison:" $log.txt | sed -n 1p | sed -E 's/.*poison:(-?[0-9]+).*/\1/')
  p2=$(grep "PANEL character:.*poison:" $log.txt | sed -n 2p | sed -E 's/.*poison:(-?[0-9]+).*/\1/')
  echo "poison resist shown before=$p1 after=$p2"
  [ -n "$p1" ] && [ -n "$p2" ] && [ $((p2 - p1)) -gt 0 ] && [ $((p2 - p1)) -le 50 ] || { echo "FAIL: the antidote did not add 50 poison resist to the character screen (shown values are capped at 75, 85 with the antidote's max-resist +10, so a hero already above the cap gains only 10)"; fail=1; }
  grep -q "BELT drink code=yps state=antidote seconds=30.0 .*poisonresist" $log.txt || { echo "FAIL: no antidote state"; fail=1; }
  grep -q "BELT drink code=vps state=staminapot seconds=30.0" $log.txt || { echo "FAIL: no stamina state"; fail=1; }
}
