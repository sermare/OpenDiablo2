scenario_name="real outdoor (DRLG Blood Moor: generated from the layout, presets stamped, plain rooms built, screenshot)"
shot=$tmp/realoutdoor.png
scenario_env() {
  echo 'export OD2_AUTOLEVEL=2'
  echo "export OD2_AUTOSCRIPT='wait:1;say:capframe $shot;wait:1;exit'"
}
scenario_check() {
  grep -E "real outdoor:|AUTOSCRIPT RESULT" $log.txt | cut -c1-200
  grep -q "AUTOSCRIPT RESULT PASS" $log.txt || { echo "FAIL: real outdoor scenario did not pass"; fail=1; }
  grep -q "real outdoor: level 2 .* presets stamped" $log.txt || { echo "FAIL: the DRLG outdoor level was not generated"; fail=1; }
  [ -s $shot ] || { echo "FAIL: no screenshot"; fail=1; }
  if grep -E "Unknown tile|panic" $log.txt; then echo "FAIL: unknown tiles or panic in real outdoor log"; fail=1; fi
}
