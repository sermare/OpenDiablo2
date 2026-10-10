scenario_name="visual fidelity (outdoor at night: Blood Moor lit by the day/night ambient and the hero light, logged draw statistics)"
shot=$tmp/outdoor-night.png
scenario_env() {
  echo "export OD2_REALMAPS=1 OD2_AUTOLEVEL=2 OD2_AUTOTIME=night OD2_DRAWSTATS=1"
  echo "export OD2_AUTOSCRIPT='wait:4;say:capframe $shot;wait:1;exit'"
}
scenario_check() {
  grep -E "DRAWSTATS|LIGHT level|AUTOSCRIPT RESULT" $log.txt | cut -c1-220 | tail -5
  grep -q "AUTOSCRIPT RESULT PASS" $log.txt || { echo "FAIL: outdoor visual script did not pass"; fail=1; }
  grep -qE "DRAWSTATS level=2 floors=[1-9][0-9]* .* lit=true" $log.txt || { echo "FAIL: outdoor not drawn lit"; fail=1; }
  [ -s $shot ] || { echo "FAIL: no outdoor screenshot"; fail=1; }
}
