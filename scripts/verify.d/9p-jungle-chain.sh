scenario_name="Act 3 jungle chain (OD2_REALMAPS=1: Kurast Docks -> the six jungle and Kurast levels by waypoint through real level changes; TILESTATS of each level)"
# The tile image cache must not carry pictures from one level to the next: the levels are entered in travel order and every
# one must show the exact tile records (TILESTATS exact=true, no cell without a floor, no tile reference without a DT1 tile).
jc_dir=$tmp
scenario_env() {
  echo 'export OD2_REALMAPS=1'
  echo "export OD2_AUTOSCRIPT='wait:1;say:resetquests;say:travelfree 1;say:setwaypoint 76 1;say:setwaypoint 77 1;say:setwaypoint 78 1;say:setwaypoint 79 1;say:setwaypoint 80 1;say:setwaypoint 81 1;travel:3;expect:level=75;wait:3;say:capframe $jc_dir/9p-75.png;wait:4;use:Waypoint;waypoint:76;expect:level=76;wait:3;say:capframe $jc_dir/9p-76.png;wait:4;use:Waypoint;waypoint:77;expect:level=77;wait:3;say:capframe $jc_dir/9p-77.png;wait:4;use:Waypoint;waypoint:78;expect:level=78;wait:3;say:capframe $jc_dir/9p-78.png;wait:4;use:Waypoint;waypoint:79;expect:level=79;wait:3;say:capframe $jc_dir/9p-79.png;wait:4;use:Waypoint;waypoint:80;expect:level=80;wait:3;say:capframe $jc_dir/9p-80.png;wait:4;use:Waypoint;waypoint:81;expect:level=81;wait:3;say:capframe $jc_dir/9p-81.png;exit'"
}
scenario_check() {
  grep -E "TILESTATS|AUTOSCRIPT RESULT" $log.txt | cut -c1-240
  grep -q "AUTOSCRIPT RESULT PASS" $log.txt || { echo "FAIL: the jungle chain did not pass"; fail=1; }
  for l in 75 76 77 78 79 80 81; do
    line=$(grep -E "TILESTATS level=$l " $log.txt | tail -1)
    [ -n "$line" ] || { echo "FAIL: no TILESTATS line for level $l"; fail=1; continue; }
    echo "$line" | grep -q "exact=true" || { echo "FAIL: level $l did not use the exact tile records"; fail=1; }
    echo "$line" | grep -q "nofloor=0 " || { echo "FAIL: level $l has cells without a floor"; fail=1; }
    echo "$line" | grep -q "fallbacks=0 " || { echo "FAIL: level $l has tile references without a DT1 tile"; fail=1; }
  done
  if grep -E "Unknown tile|Could not locate tile|panic" $log.txt; then echo "FAIL: unknown tiles or panic in the jungle chain log"; fail=1; fi
}
