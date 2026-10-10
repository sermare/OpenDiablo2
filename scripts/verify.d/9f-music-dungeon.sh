scenario_name="dungeon music and footsteps (unmuted: Den of Evil on real maps plays its music, the hero's footsteps and monster sounds; no load errors)"
scenario_unmuted=1
scenario_env() {
  echo 'export OD2_SOUNDLOG=1 OD2_AUTOLEVEL=9'
  echo "export OD2_AUTOSCRIPT='wait:3;move:33,56;wait:32;exit'"
}
scenario_check() {
  grep -E "AUTOSCRIPT RESULT|AMBIENT env change" $log.txt | cut -c1-260
  grep -q "AUTOSCRIPT RESULT PASS" $log.txt || { echo "FAIL: the dungeon walk did not pass"; fail=1; }
  if grep -E "could not (load|play)|load-failed|skipping missing" $log.txt | head -10 | grep .; then
    echo "FAIL: audio load/play errors in the unmuted run"; fail=1
  fi
  echo "sounds started (kind handle: count):"
  grep -oE "SOUNDLOG t=[0-9.]+ kind=[a-z-]+ music=[a-z]+ handle=[a-z0-9_]+" $log.txt | sed -E 's/t=[0-9.]+ //' | sort | uniq -c | sort -rn | head -40
  grep -qE "SOUNDLOG .* music=true .*decision=(played|stolen|queued)" $log.txt || { echo "FAIL: no area music started"; fail=1; }
  grep -qE "SOUNDLOG .* kind=footstep .*decision=(played|stolen|queued)" $log.txt || { echo "FAIL: no footsteps"; fail=1; }
}
