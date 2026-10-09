scenario_name="waypoint and portal (walk to the Rogue Encampment waypoint, panel lists the act 1 entries, portal level change)"
scenario_env() {
  echo 'export OD2_AUTOSCRIPT='wait:1;use:Waypoint;expect:log=WAYPOINT PANEL act=1 level=1 rows=9 active=9 enabled=1;expect:log=1:Rogue Encampment:on;expect:log=3:Cold Plains:grey;expect:log=35:Catacombs Level 2:grey;say:spawnportal 1;use:Portal;expect:level=1;expect:log=LEVEL CHANGE from=1 to=1;wait:1;exit''
}
scenario_check() {
  grep -E "AUTOSCRIPT level=|WAYPOINT PANEL|LEVEL CHANGE|AUTOSCRIPT RESULT" $log.txt | cut -c1-260
  grep -q "AUTOSCRIPT RESULT PASS" $log.txt || { echo "FAIL: waypoint/portal scenario did not pass"; fail=1; }
}
