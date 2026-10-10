scenario_name="uniques and champions (Dark Wood, Normal: pack counts and leader classes legal for the Levels.txt row, ranks applied to the monsters)"
# Level 4 (Act 1 Wilderness 3: umon skeleton1 / zombie2, MonUMin 1 MonUMax 2). The game starts in the level with the
# real-map population forced on; scripts/unique_pop_check.py compares the log with the tables (needs D2_TABLES).
scenario_env() {
  echo "export OD2_REALMAPS=1 OD2_AUTOLEVEL=4 OD2_POPULATE=1"
  echo "export OD2_AUTOSCRIPT='wait:2;exit'"
}
scenario_check() {
  grep -E "population ranks|POPULATE (level|ranks|rank leaders)|AUTOSCRIPT RESULT" $log.txt | cut -c1-260
  grep -q "AUTOSCRIPT RESULT PASS" $log.txt || { echo "FAIL: uniques/champions scenario did not pass"; fail=1; }
  python3 -I scripts/unique_pop_check.py $log.txt 4; rc=$?
  [ $rc -eq 1 ] && { echo "FAIL: unique / champion population of level 4 is not legal for its Levels.txt row"; fail=1; }
  [ $rc -eq 2 ] && echo "SKIP: D2_TABLES unset"
  # the exe's rules on the real tables (monumod picks, rare minions 3..6, champion minions, party packs, super uniques)
  go test -count=1 ./d2common/d2monreg/ 2>&1 | grep -v "ld: warning\|^# " | grep -v "^ok" && fail=1
  return 0
}
