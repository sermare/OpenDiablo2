scenario_name="hardcore (a dead hardcore character is refused at load; a hardcore death is permanent and flagged dead in the .d2s)"
# two runs would be needed to play a death and then load it, so the load refusal marks the start hero
# dead (OD2_AUTOMARKDEAD, the state a hardcore death leaves); the death itself is 8d-hardcore-death.sh
scenario_env() {
  echo "export OD2_AUTOMARKDEAD=1"
}
scenario_check() {
  grep -E "HARDCORE" $log.txt | cut -c1-260
  grep -qE "HARDCORE marked .* dead \(kind=2\)" $log.txt || { echo "FAIL: hero not marked dead hardcore"; fail=1; }
  grep -q "HARDCORE refused: hardcore character .* is dead and cannot be played" $log.txt || { echo "FAIL: dead hardcore character was loaded"; fail=1; }
  if grep -q "GAME START\|DEATHTEST\|AUTOTEST" $log.txt; then echo "FAIL: the game started with a dead hardcore character"; fail=1; fi
  go test -run 'Kind|Permanent|ExportDeathStatus|IsDeadHardcore' ./d2core/d2hero/ 2>&1 | grep -v "ld: warning\|^# " | grep -v "^ok" && fail=1
}
