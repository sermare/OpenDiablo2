scenario_name="combat sound (unmuted: Fire Bolt cast, missile travel and hit, monster attack, hit and death sounds start)"
scenario_unmuted=1
scenario_env() { echo 'export OD2_SOUNDLOG=1 OD2_AUTOMONSTER="zombie1,2" OD2_AUTOMONSTER_SECONDS=30 OD2_AUTOMONSTER_DIFF=0 OD2_AUTOCAST="Fire Bolt,4"'; }
scenario_check() {
  if grep -E "could not (load|play)|load-failed|skipping missing" $log.txt | head -10 | grep .; then
    echo "FAIL: audio load/play errors in the unmuted run"; fail=1
  fi
  echo "sounds started (kind handle: count):"
  grep -oE "SOUNDLOG t=[0-9.]+ kind=[a-z-]+ music=[a-z]+ handle=[a-z0-9_]+" $log.txt | sed -E 's/t=[0-9.]+ //' | sort | uniq -c | sort -rn | head -40
  # Fire Bolt: Skills.txt stsound, Missiles.txt TravelSound/HitSound
  for k in skill-start missile-travel missile-hit; do
    grep -qE "SOUNDLOG .* kind=$k .*decision=(played|stolen|queued)" $log.txt || { echo "FAIL: no $k sound"; fail=1; }
  done
  # the zombies die: monster sounds from MonSounds.txt
  grep -qE "SOUNDLOG .* kind=death .*decision=(played|stolen|queued)" $log.txt || { echo "FAIL: no monster death sound"; fail=1; }
}
