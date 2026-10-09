scenario_name="cave stairs chain + level persistence (Cave Level 1 -> Cave Level 2 treasure preset -> back; Catacombs 3 -> 4; a killed monster stays dead)"
# Levels 13..16 and 37 are DrlgType 2 presets (drlgoutdoor.GeneratePreset, golden: testdata/preset_act1.json).
# The hero kills three monsters of level 9 (killnear), takes the stairs down to level 13 and back: the level is
# rebuilt from the seed but must not be populated again and keeps the same monsters and ground items (PERSIST lines,
# one POPULATE line for level 9; the 8 s wait is the portal cooldown). Level 37 is entered by the stairs of level 36.
scenario_warnings_ok=1   # the renderers log tile messages of their own; only a crash counts here
scenario_env() {
  echo 'export OD2_REALMAPS=1 OD2_AUTOLEVEL=9 OD2_AUTOSPEED=3 OD2_AUTOMONSTER_DIFF=0'
  echo "export OD2_AUTOSCRIPT='wait:2;say:killnear;say:killnear;say:killnear;wait:1;walkto:exit=13;expect:level=13;wait:1;walkto:exit=9;expect:level=9;wait:8;say:spawnportal 36;use:Portal;expect:level=36;walkto:exit=37;expect:level=37;wait:6;walkto:exit=36;expect:level=36;exit'"
}
scenario_check() {
  grep -E "AUTOSCRIPT level=|LEVEL CHANGE|real preset: level|PERSIST|AUTOSCRIPT RESULT" $log.txt | cut -c1-240
  grep -q "AUTOSCRIPT RESULT PASS" $log.txt || { echo "FAIL: cave chain scenario did not pass"; fail=1; }
  grep -q "real preset: level 13 " $log.txt || { echo "FAIL: level 13 was not built by the preset generator"; fail=1; }
  grep -q "real preset: level 37 " $log.txt || { echo "FAIL: level 37 was not built by the preset generator"; fail=1; }
  grep -q "LEVEL CHANGE from=9 to=13" $log.txt || { echo "FAIL: no stairs from level 9 to 13"; fail=1; }
  grep -q "LEVEL CHANGE from=13 to=9" $log.txt || { echo "FAIL: no stairs from level 13 to 9"; fail=1; }
  grep -q "LEVEL CHANGE from=36 to=37" $log.txt || { echo "FAIL: no stairs from level 36 to 37"; fail=1; }
  # persistence: level 9 is saved when left and restored (same monsters, no second population)
  saved=$(grep -m1 "PERSIST saved level 9 " $log.txt | sed -E 's/.*: ([0-9]+) monsters.*/\1/')
  back=$(grep -m1 "PERSIST restored level 9 " $log.txt | sed -E 's/.*visit [0-9]+: ([0-9]+) monsters back.*/\1/')
  [ -n "$saved" ] && [ "$saved" = "$back" ] || { echo "FAIL: level 9 monsters saved=$saved restored=$back"; fail=1; }
  [ "$(grep -c "POPULATE level 9 " $log.txt)" = 1 ] || { echo "FAIL: level 9 was populated more than once"; fail=1; }
  back36=$(grep -m1 "PERSIST restored level 36 " $log.txt | sed -E 's/.*visit [0-9]+: ([0-9]+) monsters back.*/\1/')
  saved36=$(grep -m1 "PERSIST saved level 36 " $log.txt | sed -E 's/.*: ([0-9]+) monsters.*/\1/')
  [ -n "$saved36" ] && [ "$saved36" = "$back36" ] || { echo "FAIL: level 36 monsters saved=$saved36 restored=$back36"; fail=1; }
  if grep -E "panic" $log.txt; then echo "FAIL: panic in cave chain log"; fail=1; fi
}
