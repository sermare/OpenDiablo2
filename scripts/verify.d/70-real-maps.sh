scenario_name="real maps (DRLG Den of Evil cave: renders, hero walks, monsters aggro, screenshot)"
shot=$tmp/realmaps.png
scenario_env() {
  echo 'export OD2_AUTOLEVEL=9'
  echo "export OD2_AUTOSCRIPT='wait:1;say:capframe $shot;wait:1;move:33,56;wait:32;expect:log=aggro=1;exit'"
}
scenario_check() {
  grep -E "real maze:|AUTOSCRIPT RESULT" $log.txt | cut -c1-200
  grep -q "AUTOSCRIPT RESULT PASS" $log.txt || { echo "FAIL: real maps scenario did not pass"; fail=1; }
  grep -q "real maze: level 9 .* stamps placed" $log.txt || { echo "FAIL: the DRLG level was not generated"; fail=1; }
  [ -s $shot ] || { echo "FAIL: no screenshot"; fail=1; }
  if grep -E "Unknown tile" $log.txt; then echo "FAIL: unknown tiles in real maps log"; fail=1; fi
}
