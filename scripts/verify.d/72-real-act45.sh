scenario_name="real Act 5 outdoor (DRLG Frigid Highlands: Act 5 world placement, barricade presets, plain rooms, screenshot)"
shot=$tmp/realact45.png
scenario_env() {
  echo 'export OD2_AUTOLEVEL=111'
  echo "export OD2_AUTOSCRIPT='wait:1;say:capframe $shot;wait:1;exit'"
}
scenario_check() {
  grep -E "real outdoor:|AUTOSCRIPT RESULT" $log.txt | cut -c1-200
  grep -q "AUTOSCRIPT RESULT PASS" $log.txt || { echo "FAIL: real Act 5 outdoor scenario did not pass"; fail=1; }
  grep -q "real outdoor: level 111 .* presets stamped" $log.txt || { echo "FAIL: the Act 5 DRLG outdoor level was not generated"; fail=1; }
  [ -s $shot ] || { echo "FAIL: no screenshot"; fail=1; }
  if grep -E "Unknown tile|panic" $log.txt; then echo "FAIL: unknown tiles or panic in real Act 5 log"; fail=1; fi
}
