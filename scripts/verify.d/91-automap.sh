scenario_name="automap (Tab map reveals as the hero walks: Rogue Encampment, then Jail Level 1; full and mini; screenshots)"
scenario_warnings_ok=1   # the maze renderer logs tile messages of its own; only a crash counts here
am=$tmp/automap
scenario_env() {
  echo 'export OD2_REALMAPS=1'
  # town: map on (full), walk to Akara, look; waypoint to Jail Level 1, walk, look at full and mini; map off
  echo "export OD2_AUTOSCRIPT='wait:1;automap:full;wait:1;automap:stats;say:capframe $am-town0.png;move:npc=Akara;wait:12;automap:stats;say:capframe $am-town1.png;use:Waypoint;waypoint:29;expect:level=29;wait:2;automap:stats;move:30,34;wait:10;automap:stats;say:capframe $am-jail1.png;move:12,45;wait:16;automap:stats;say:capframe $am-jail2.png;automap:mini;wait:1;say:capframe $am-jail3.png;automap:off;wait:1;automap:stats;say:capframe $am-off.png;exit'"
}
scenario_check() {
  grep -E "AUTOMAP table|AUTOMAP state|AUTOSCRIPT level=|AUTOSCRIPT RESULT" $log.txt | cut -c1-200
  grep -q "AUTOSCRIPT RESULT PASS" $log.txt || { echo "FAIL: automap scenario did not pass"; fail=1; }
  grep -q "AUTOMAP table: .* 3286 rows" $log.txt || { echo "FAIL: the AutoMap table did not load (3286 rows expected)"; fail=1; }
  # the count of revealed cells must grow as the hero walks (town: start vs after the walk; Jail: arrival vs the end)
  town=$(grep "AUTOMAP state on=true size=full level=1 " $log.txt | sed 's/.* cells=\([0-9]*\) .*/\1/')
  jail=$(grep "AUTOMAP state on=.* level=29 " $log.txt | sed 's/.* cells=\([0-9]*\) .*/\1/')
  t0=$(echo $town | head -1); t1=$(echo $town | tail -1)
  j0=$(echo $jail | head -1); j1=$(echo $jail | tail -1)
  echo "revealed cells: town $t0 -> $t1, jail $j0 -> $j1"
  [ -n "$t0" ] && [ "$t0" -gt 0 ] || { echo "FAIL: nothing revealed in the town"; fail=1; }
  [ "$t1" -gt "$t0" ] || { echo "FAIL: walking in the town revealed no new cells"; fail=1; }
  [ "$j1" -gt "$j0" ] || { echo "FAIL: walking in Jail Level 1 revealed no new cells"; fail=1; }
  grep -q "AUTOMAP state on=false" $log.txt || { echo "FAIL: automap:off did not turn the map off"; fail=1; }
  grep -q "AUTOMAP state on=true size=mini" $log.txt || { echo "FAIL: automap:mini did not switch"; fail=1; }
  for s in town0 town1 jail1 jail2 jail3 off; do
    [ -s $am-$s.png ] || { echo "FAIL: no screenshot $s"; fail=1; }
  done
  echo "screenshots: $am-*.png"
  if grep -E "panic" $log.txt; then echo "FAIL: panic in the automap log"; fail=1; fi
}
