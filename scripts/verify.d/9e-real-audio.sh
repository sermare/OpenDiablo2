scenario_realtime=1
scenario_name="real audio (unmuted Act 1 quest run: NPC speech, quest messages, music and sfx all load and play)"
scenario_unmuted=1
scenario_env() {
  echo 'export OD2_AUTOQUEST=act1'
}
scenario_check() {
  grep -q "AUTOQUEST RESULT PASS" $log.txt || { echo "FAIL: the unmuted quest run did not pass"; fail=1; }
  # the runner tolerates "skipping missing" music lines elsewhere; unmuted, any load or play failure is a bug
  if grep -E "could not (load|play)|skipping missing" $log.txt | head -10 | grep .; then
    echo "FAIL: audio load/play errors in the unmuted run"; fail=1
  fi
  grep -q "QUEST SPEECH" $log.txt || { echo "FAIL: no NPC speech played"; fail=1; }
}
