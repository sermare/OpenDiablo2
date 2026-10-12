scenario_name="automap load (a new game of the same hero finds the town revealed by 92-automap-save before any step)"
scenario_warnings_ok=1
scenario_env() {
  # no sidecar clean-up here: the files written by 92-automap-save are the input
  echo "export OD2_AUTOSCRIPT='wait:2;automap:stats;exit'"
}
scenario_check() {
  grep -E "AUTOMAP (state|saved|loaded)|AUTOSCRIPT RESULT" $log.txt | cut -c1-200
  grep -q "AUTOSCRIPT RESULT PASS" $log.txt || { echo "FAIL: automap load scenario did not pass"; fail=1; }
  [ -s $tmp/am45.cells ] || { echo "SKIP: 92-automap-save did not run in this run, nothing to reload"; return; }
  want=$(cat $tmp/am45.cells)
  got=$(grep "AUTOMAP loaded level=1 " $log.txt | head -1 | sed "s/.* cells=\([0-9]*\) .*/\1/")
  echo "saved cells $want, loaded cells ${got:-none}"
  [ "$got" = "$want" ] || { echo "FAIL: the reloaded game did not load the saved town cells"; fail=1; }
  now=$(grep "AUTOMAP state on=.* level=1 " $log.txt | tail -1 | sed "s/.* cells=\([0-9]*\) .*/\1/")
  [ -n "$now" ] && [ "$now" -ge "$want" ] || { echo "FAIL: the town shows fewer cells ($now) than were saved ($want)"; fail=1; }
  if grep -E "panic" $log.txt; then echo "FAIL: panic"; fail=1; fi
}
