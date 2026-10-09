scenario_name="imported hero UI (character, skills, inventory panels log their values; screenshot)"
scenario_env() {
  echo "export OD2_AUTOSCRIPT='wait:1;panel:inventory;wait:1;panel:character;wait:1;panel:skills;wait:1;shot:$tmp/panels.png;wait:2;panel:close;exit'"
}
scenario_check() {
  grep -E "AUTOSCRIPT RESULT|PANEL" $log.txt | cut -c1-300
  grep -q "AUTOSCRIPT RESULT PASS" $log.txt || { echo "FAIL: imported hero scenario did not pass"; fail=1; }
  grep -qE "PANEL character: level=[0-9]+ name=.* str=[0-9]+ dex=[0-9]+ vit=[0-9]+ ene=[0-9]+ hp=[0-9]+/[0-9]+ .* exp=[0-9]+ next=[0-9]+ " $log.txt || { echo "FAIL: no PANEL character line"; fail=1; }
  grep -qE "PANEL skills: class=.* unspent=[0-9]+ spent=[0-9]+ icons=" $log.txt || { echo "FAIL: no PANEL skills line"; fail=1; }
  grep -qE "PANEL skills active: left=.*\(id=[0-9]+,lvl=[0-9]+\) right=.*\(id=[0-9]+,lvl=[0-9]+\)" $log.txt || { echo "FAIL: no PANEL skills active line"; fail=1; }
  grep -qE "PANEL inventory: gold=[0-9]+ items=[0-9]+ worn=\[" $log.txt || { echo "FAIL: no PANEL inventory line"; fail=1; }
  [ -s $tmp/panels.png ] || { echo "FAIL: no screenshot from the shot: step"; fail=1; }
}
