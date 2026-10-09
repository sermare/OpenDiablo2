scenario_name="autosave (OD2_AUTOSAVE=1: set gold in game, exit, exported .d2s re-parses)"
wb=$tmp/writeback
scenario_env() {
  mkdir -p $wb   # keeps the exported file out of the Saves folder
  echo "export OD2_AUTOSAVE=1 OD2_D2S_WRITEBACK=\"$wb\""
}
scenario_check() {
  grep -E "D2S EXPORT|AUTOSCRIPT RESULT" $log.txt | cut -c1-300
  grep -q "AUTOSCRIPT RESULT PASS" $log.txt || { echo "FAIL: autosave script did not pass"; fail=1; }
  grep -q "D2S EXPORT reparse: .*gold=31337 .*checksum=ok" $log.txt || { echo "FAIL: exported .d2s does not show the new gold"; fail=1; }
  ls $wb/*.d2s >/dev/null 2>&1 || { echo "FAIL: no exported .d2s in $wb"; fail=1; }
}
