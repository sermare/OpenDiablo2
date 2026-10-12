scenario_name="state colour (a Sorceress chills zombies with Ice Bolt: the colour shift of the cold state is applied, frame captured)"
# w49: states.txt colorpri/colorshift (chilled units turn blue, poisoned green). The engine logs STATE tint when a
# unit's colour shift first applies; OD2_SHOT_DIR (default $tmp) takes the PNG, never committed.
scenario_realtime=1
scenario_env() {
  local d="${OD2_SHOT_DIR:-$tmp}"
  rm -f "$d/st1.png"
  echo "export OD2_AUTOCAST_CLASS=sorceress OD2_AUTOCAST_MANA=900 OD2_AUTOCAST_LEVEL=1 OD2_AUTOMONSTER=\"zombie1,4\" OD2_AUTOMONSTER_SECONDS=60 OD2_AUTOMONSTER_DIFF=0"
  echo "export OD2_AUTOCAST=\"Ice Bolt,4\""
  echo "export OD2_AUTOSCRIPT='wait:6;say:capframe $d/st1.png;wait:2;exit'"
}
scenario_check() {
  grep -E "STATE tint|AUTOCAST summary" $log.txt | cut -c1-200 | head -8
  grep -qE "STATE tint unit=.* shift=108" $log.txt || { echo "FAIL: no cold colour shift applied"; fail=1; }
  [ -s "${OD2_SHOT_DIR:-$tmp}/st1.png" ] || { echo "FAIL: no frame captured"; fail=1; }
}
