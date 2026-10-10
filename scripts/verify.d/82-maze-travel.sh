scenario_name="maze travel (default real maps: waypoint to Jail Level 1, open and close a door, portal back to town)"
scenario_warnings_ok=1   # the maze renderer logs tile messages of its own; only a crash counts here
scenario_env() {
  echo "export OD2_AUTOSCRIPT='wait:1;use:Waypoint;waypoint:29;expect:level=29;use:Door;use:Door;wait:6;say:spawnportal 1;use:Portal;expect:level=1;exit'"
}
scenario_check() {
  grep -E "AUTOSCRIPT level=|LEVEL CHANGE|OBJECT door|AUTOSCRIPT RESULT" $log.txt | cut -c1-260
  grep -q "AUTOSCRIPT RESULT PASS" $log.txt || { echo "FAIL: maze travel scenario did not pass"; fail=1; }
  grep -q "OBJECT door .*open=true .*blocks=false" $log.txt || { echo "FAIL: door did not open"; fail=1; }
  grep -q "OBJECT door .*open=false .*blocks=true" $log.txt || { echo "FAIL: door did not close and block"; fail=1; }
  if grep -E "panic" $log.txt; then echo "FAIL: panic in maze travel log"; fail=1; fi
}
