scenario_name="items6 (set bonus tiers, max durability blue split; real save totals 1241/477/801 unchanged)"
scenario_env() {
  echo "export OD2_AUTOSCRIPT='wait:2;panel:character;wait:1;panel:close;exit'"
}
scenario_check() {
  grep -E "stats: .* level=|AUTOSCRIPT RESULT" $log.txt | cut -c1-300
  grep -qE "stats: .* level=94 .*life=1241/1241 mana=464/477 stamina=801/801" $log.txt || { echo "FAIL: totals are not 1241/1241, 464/477, 801/801"; fail=1; }
  grep -q "AUTOSCRIPT RESULT PASS" $log.txt || { echo "FAIL: scripted scenario did not pass"; fail=1; }
  go test ./d2common/d2statlist/ -run SetBonusTiers 2>&1 | grep -v "ld: warning\|^# " | grep -v "^ok" && fail=1
  go test ./d2core/d2item/d2itemdesc/ -run "DurabilityLine|DurabilityMaxBlue" 2>&1 | grep -v "ld: warning\|^# " | grep -v "^ok" && fail=1
}
