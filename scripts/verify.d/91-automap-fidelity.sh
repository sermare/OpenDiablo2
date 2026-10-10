scenario_name="automap fidelity (fade, names, markers, per-level persistence: town -> Jail Level 1 -> town; screenshots)"
scenario_warnings_ok=1
am=$tmp/amfid
scenario_env() {
  echo 'export OD2_REALMAPS=1'
  # town with names+party markers on (stash label, NPC names), fade on; cave (Jail 1) with fade, then mini (flat fade);
  # back to the town: the revealed cells of the town must still be there
  echo "export OD2_AUTOSCRIPT='wait:1;automap:full;automap:names;automap:party;wait:1;move:npc=Akara;wait:12;automap:stats;say:capframe $am-town-names.png;automap:fade;wait:1;say:capframe $am-town-fade.png;automap:nofade;automap:stats;use:Waypoint;waypoint:29;expect:level=29;wait:2;automap:fade;move:30,34;wait:10;automap:stats;say:capframe $am-cave-fade.png;automap:mini;wait:1;say:capframe $am-cave-mini.png;automap:full;automap:nofade;use:Waypoint;waypoint:1;expect:level=1;wait:3;automap:stats;say:capframe $am-town-back.png;exit'"
}
scenario_check() {
  grep -E "AUTOMAP (table|state|option)|AUTOSCRIPT level=|AUTOSCRIPT RESULT" $log.txt | cut -c1-200
  grep -q "AUTOSCRIPT RESULT PASS" $log.txt || { echo "FAIL: automap fidelity scenario did not pass"; fail=1; }
  grep -q "AUTOMAP option fade=true" $log.txt || { echo "FAIL: automap:fade did not take"; fail=1; }
  grep -q "AUTOMAP option names=true" $log.txt || { echo "FAIL: automap:names did not take"; fail=1; }
  town=$(grep "AUTOMAP state on=true size=full level=1 " $log.txt | sed 's/.* cells=\([0-9]*\) .*/\1/')
  left=$(echo "$town" | sed -n 2p); back=$(echo "$town" | tail -1)
  echo "town cells: $town (left with $left, back with $back)"
  [ -n "$left" ] && [ "$back" -ge "$left" ] || { echo "FAIL: the town lost its revealed cells after the round trip"; fail=1; }
  for s in town-names town-fade cave-fade cave-mini town-back; do
    [ -s $am-$s.png ] || { echo "FAIL: no screenshot $s"; fail=1; }
  done
  echo "screenshots: $am-*.png"
  if grep -E "panic" $log.txt; then echo "FAIL: panic in the automap fidelity log"; fail=1; fi
}
