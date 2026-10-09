scenario_name="hardcore death (OD2_AUTODEATH_HARDCORE: the hero dies for good, no respawn, saved dead; softcore respawn unchanged in the death scenario)"
wb=$tmp/writeback-hcdeath
scenario_env() {
  mkdir -p $wb
  echo "export OD2_D2S_WRITEBACK=\"$wb\" OD2_AUTODEATH=1 OD2_AUTODEATH_HARDCORE=1 OD2_AUTODEATH_LEVEL=10 OD2_AUTODEATH_HP=20 OD2_AUTOMONSTER_DIFF=1"
}
scenario_check() {
  grep -E "DEATH |DEATHTEST|D2S EXPORT reparse" $log.txt | cut -c1-300 | head -20
  grep -qE "DEATH hero=.* hardcore=true " $log.txt || { echo "FAIL: no hardcore death"; fail=1; }
  grep -q "DEATH hardcore: the character is dead" $log.txt || { echo "FAIL: hardcore death not final"; fail=1; }
  if grep -q "DEATH respawn" $log.txt; then echo "FAIL: a hardcore hero respawned"; fail=1; fi
  grep -qE "D2S EXPORT reparse: .*hardcore=true .*died=true" $log.txt || { echo "FAIL: the saved .d2s is not flagged dead hardcore"; fail=1; }
}
