scenario_name="w51 Arreat Summit renders (lower-wall cliff faces under the later floors, void fillers under everything); frame captured for a human look"
shot=$tmp/w51summit.png
scenario_env() {
  echo 'export OD2_AUTOLEVEL=120'
  echo "export OD2_AUTOSCRIPT='wait:2;say:capframe $shot;wait:1;exit'"
}
scenario_check() {
  grep -E "TILESTATS level=120|AUTOSCRIPT RESULT" $log.txt | cut -c1-200
  grep -q "AUTOSCRIPT RESULT PASS" $log.txt || { echo "FAIL: Arreat Summit scenario did not pass"; fail=1; }
  [ -s $shot ] || { echo "FAIL: no screenshot"; fail=1; }
  go test ./d2core/d2map/d2maprenderer/ -run "DiagonalOrder|IsBlankFloor" 2>&1 | grep -v "ld: warning\|^# " | grep -v "^ok" && fail=1
}
