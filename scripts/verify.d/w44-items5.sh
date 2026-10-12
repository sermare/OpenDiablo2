scenario_name="items5 (inventory charms gated by the exe's identified and level checks still give the real save its 1241/477/801 totals; refill and durability tooltip unit tests)"
scenario_env() {
  echo "export OD2_AUTOSCRIPT='wait:2;panel:character;wait:1;panel:close;exit'"
}
scenario_check() {
  grep -E "stats: .* level=|PANEL character:|AUTOSCRIPT RESULT" $log.txt | cut -c1-300
  # INV_ItemMeetsRequirementsNow 0x55b780 chain: the save's charms are identified and within level, so the totals with charms stay
  grep -qE "stats: .* level=94 .*life=1241/1241 mana=464/477 stamina=801/801" $log.txt || { echo "FAIL: totals with the inventory charms are not 1241/1241, 464/477, 801/801"; fail=1; }
  grep -q "AUTOSCRIPT RESULT PASS" $log.txt || { echo "FAIL: scripted scenario did not pass"; fail=1; }
  go test ./d2core/d2hero/ -run CharmActive 2>&1 | grep -v "ld: warning\|^# " | grep -v "^ok" && fail=1
  go test ./d2core/d2item/diablo2item/ -run Refill 2>&1 | grep -v "ld: warning\|^# " | grep -v "^ok" && fail=1
  go test ./d2core/d2item/d2itemdesc/ -run DurabilityMaxBlue 2>&1 | grep -v "ld: warning\|^# " | grep -v "^ok" && fail=1
}
