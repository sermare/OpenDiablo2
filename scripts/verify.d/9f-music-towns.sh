scenario_name="area music and sound by act (unmuted: towns of acts 1-5 each start their music, no asset load errors, sounds listed)"
scenario_unmuted=1
# One scenario at a time (the sound device is shared). The hero visits the town of every act (act travel is free in
# this scenario, see 99-act-travel.sh) and stays 6 s in each so the area music, ambience and events start. Every
# started sound is logged as SOUNDLOG (OD2_SOUNDLOG=1); the check lists them and fails on load errors.
scenario_env() {
  local s="wait:3;say:travelfree 1"
  for a in 2 3 4 5; do s+=";travel:$a;wait:6"; done
  s+=";travel:1;wait:6;exit"
  echo "export OD2_SOUNDLOG=1 OD2_AUTOSCRIPT=\"$s\""
}
scenario_check() {
  grep -E "AUTOSCRIPT RESULT" $log.txt | cut -c1-200
  grep -q "AUTOSCRIPT RESULT PASS" $log.txt || { echo "FAIL: the act tour did not pass"; fail=1; }
  if grep -E "could not (load|play)|load-failed|skipping missing" $log.txt | head -10 | grep .; then
    echo "FAIL: audio load/play errors in the unmuted run"; fail=1
  fi
  echo "sounds started (distinct handle: count):"
  grep -oE "SOUNDLOG t=[0-9.]+ kind=[a-z-]+ music=[a-z]+ handle=[a-z0-9_]+" $log.txt | sed -E 's/t=[0-9.]+ //' | sort | uniq -c | sort -rn | head -40
  # the town music of every act (Levels.txt SoundEnv -> SoundEnviron.txt Song): town 1..4 and the Act 5 town theme
  for m in music_town_1 music_town_2 music_town_3 music_town_4 music_xtown; do
    grep -qE "SOUNDLOG .* music=true handle=$m .*decision=(played|stolen|queued)" $log.txt || { echo "FAIL: $m never started"; fail=1; }
  done
}
