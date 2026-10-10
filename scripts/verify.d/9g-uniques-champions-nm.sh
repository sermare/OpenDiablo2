scenario_name="uniques and champions (Stony Field, Nightmare: MonUMin(N)..MonUMax(N) packs, leaders from the type list, ranks applied)"
# Level 4 in Nightmare (MonUMin(N) 4 MonUMax(N) 6; the leaders come from the level's nmon type list, not umon). The hero
# starts in the Nightmare town (OD2_AUTODIFFICULTY=1; the sample save has Nightmare unlocked) and takes the waypoint to level 4.
scenario_env() {
  echo "export OD2_REALMAPS=1 OD2_POPULATE=1 OD2_AUTODIFFICULTY=1"
  echo "export OD2_AUTOSCRIPT='wait:1;use:Waypoint;waypoint:4;expect:level=4;wait:3;exit'"
}
scenario_check() {
  grep -E "population ranks|POPULATE (level|ranks|rank leaders)|AUTOSCRIPT RESULT|DIFFICULTY chosen" $log.txt | cut -c1-260
  grep -q "AUTOSCRIPT RESULT PASS" $log.txt || { echo "FAIL: Nightmare uniques/champions scenario did not pass"; fail=1; }
  grep -q "population ranks level 4 diff 1:" $log.txt || { echo "FAIL: the level was not populated for Nightmare"; fail=1; }
  python3 -I scripts/unique_pop_check.py $log.txt 4; rc=$?
  [ $rc -eq 1 ] && { echo "FAIL: Nightmare unique / champion population of level 4 is not legal for its Levels.txt row"; fail=1; }
  [ $rc -eq 2 ] && echo "SKIP: D2_TABLES unset"
  return 0
}
