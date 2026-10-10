scenario_name="automap outdoors (DRLG Blood Moor: cells accumulate with the walk, fade and names options; screenshots)"
scenario_warnings_ok=1
am=$tmp/amout
scenario_env() {
  echo 'export OD2_REALMAPS=1 OD2_AUTOLEVEL=2'
  echo "export OD2_AUTOSCRIPT='wait:1;automap:full;automap:names;automap:party;wait:2;automap:stats;say:capframe $am-0.png;move:30,40;wait:14;automap:stats;say:capframe $am-1.png;automap:fade;wait:1;say:capframe $am-fade.png;automap:mini;wait:1;say:capframe $am-mini.png;exit'"
}
scenario_check() {
  grep -E "AUTOMAP (table|state|option)|AUTOSCRIPT RESULT" $log.txt | cut -c1-200
  grep -q "AUTOSCRIPT RESULT PASS" $log.txt || { echo "FAIL: automap outdoor scenario did not pass"; fail=1; }
  c=$(grep "AUTOMAP state on=true size=full level=2 " $log.txt | sed 's/.* cells=\([0-9]*\) .*/\1/')
  c0=$(echo "$c" | head -1); c1=$(echo "$c" | tail -1)
  echo "outdoor cells: $c0 -> $c1"
  [ -n "$c0" ] && [ "$c1" -gt "$c0" ] || { echo "FAIL: walking outdoors revealed no new cells"; fail=1; }
  for s in 0 1 fade mini; do [ -s $am-$s.png ] || { echo "FAIL: no screenshot $s"; fail=1; }; done
  echo "screenshots: $am-*.png"
  if grep -E "panic" $log.txt; then echo "FAIL: panic in the automap outdoor log"; fail=1; fi
}
