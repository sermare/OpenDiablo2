scenario_name="visual fidelity (dungeon: dark cave lit by the hero and the object lights, logged draw statistics)"
shot=$tmp/dungeon.png
scenario_env() {
  echo "export OD2_AUTOLEVEL=9 OD2_DRAWSTATS=1"
  echo "export OD2_AUTOSCRIPT='wait:4;say:capframe $shot;wait:1;exit'"
}
scenario_check() {
  grep -E "DRAWSTATS|LIGHT level|AUTOSCRIPT RESULT" $log.txt | cut -c1-220 | tail -5
  grep -q "AUTOSCRIPT RESULT PASS" $log.txt || { echo "FAIL: dungeon visual script did not pass"; fail=1; }
  grep -qE "DRAWSTATS level=9 floors=[1-9][0-9]* .* lit=true" $log.txt || { echo "FAIL: dungeon not drawn lit"; fail=1; }
  [ -s $shot ] || { echo "FAIL: no dungeon screenshot"; fail=1; }
}
