scenario_name="automap save (the revealed town is written to <name>.map / <name>.ma0 next to the .d2s on exit)"
scenario_warnings_ok=1
scenario_env() {
  # sidecars of earlier scenarios in this run would pre-fill the town: start clean
  rm -f $tmp/*.map(N) $tmp/*.ma[0-3](N) $tmp/am45.cells
  echo "export OD2_AUTOSCRIPT='wait:1;automap:full;move:npc=Akara;wait:12;automap:stats;exit'"
}
scenario_check() {
  grep -E "AUTOMAP (state|saved|loaded)|AUTOSCRIPT RESULT" $log.txt | cut -c1-200
  grep -q "AUTOSCRIPT RESULT PASS" $log.txt || { echo "FAIL: automap save scenario did not pass"; fail=1; }
  n=$(grep "AUTOMAP saved " $log.txt | tail -1 | sed "s/.* cells=\([0-9]*\) .*/\1/")
  [ -n "$n" ] && [ "$n" -gt 0 ] || { echo "FAIL: no AUTOMAP saved line with cells"; fail=1; }
  echo "$n" > $tmp/am45.cells
  [ -n "$(echo $tmp/*.map(N))" ] && [ -n "$(echo $tmp/*.ma0(N))" ] || { echo "FAIL: no .map/.ma0 sidecar files in $tmp"; fail=1; }
  [ "$(stat -f %z $tmp/*.map(N[1]) 2>/dev/null)" = 24 ] || { echo "FAIL: .map header is not 24 bytes"; fail=1; }
  if grep -E "panic" $log.txt; then echo "FAIL: panic"; fail=1; fi
}
