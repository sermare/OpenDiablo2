scenario_name="waypoint activation persists (clear the town waypoint bit, activate it at the object, exported .d2s has it again)"
wpsave=$tmp/wpsave.d2s
wb=$tmp/wpwriteback
scenario_env() {
  cp "$D2S_SAMPLE_BODY" $wpsave; mkdir -p $wb
  echo "export OD2_AUTOGAME=\"$wpsave\" OD2_D2S_WRITEBACK=\"$wb\""
  echo "export OD2_AUTOSCRIPT='wait:1;say:setwaypoint 1 0;wait:1;use:Waypoint;expect:log=WAYPOINT activated level=1;wait:1;exit'"
}
scenario_check() {
  grep -E "WAYPOINT (saved|activated)|AUTOSCRIPT RESULT" $log.txt | cut -c1-200
  grep -q "AUTOSCRIPT RESULT PASS" $log.txt || { echo "FAIL: waypoint persistence scenario did not pass"; fail=1; }
  grep -q "WAYPOINT saved .*level=1 bit=0 active=false .*mask=0x7ffffffffe" $log.txt || { echo "FAIL: cleared bit not saved"; fail=1; }
  last=$(grep "D2S EXPORT reparse" $log.txt | tail -1)
  case "$last" in *waypoints=0x7fffffffff*) ;; *) echo "FAIL: exported .d2s lacks the activated waypoint: $last"; fail=1;; esac
}
