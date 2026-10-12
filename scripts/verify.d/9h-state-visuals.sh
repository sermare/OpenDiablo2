scenario_name="state visuals (curses on zombies keep their looping overlays and colour: STATE overlay lines, frames captured)"
# w49: states.txt overlay1..4 / colorshift on units. A fresh Necromancer curses zombies (Amplify Damage, Weaken,
# Terror: overlays curseamplifydamage / curseweaken / curseterror); the log must show each overlay created and the
# autoscript captures frames while the curses last. OD2_SHOT_DIR (default $tmp) takes the PNGs; they are never committed.
scenario_realtime=1
scenario_env() {
  local d="${OD2_SHOT_DIR:-$tmp}"
  rm -f "$d/sv1.png"
  echo "export OD2_AUTOCAST_CLASS=necromancer OD2_AUTOCAST_MANA=900 OD2_AUTOCAST_LEVEL=10 OD2_AUTOMONSTER=\"zombie1,4\" OD2_AUTOMONSTER_SECONDS=60 OD2_AUTOMONSTER_DIFF=0"
  echo "export OD2_AUTOCAST=\"Amplify Damage,1;Weaken,1;Terror,1\""
  echo "export OD2_AUTOSCRIPT='wait:10;say:capframe $d/sv1.png;wait:2;exit'"
}
scenario_check() {
  grep -E "STATE overlay|AUTOCAST summary|AUTOSCRIPT RESULT" $log.txt | cut -c1-200 | head -12
  grep -qE "STATE overlay on unit=.* state=amplifydamage overlay=curseamplifydamage" $log.txt || { echo "FAIL: no Amplify Damage overlay"; fail=1; }
  grep -qE "STATE overlay on unit=.* state=weaken overlay=curseweaken" $log.txt || { echo "FAIL: no Weaken overlay"; fail=1; }
  # the run ends with the cast list (OD2_AUTOEXIT) and may cut the script's last step: the frame is the proof
  [ -s "${OD2_SHOT_DIR:-$tmp}/sv1.png" ] || { echo "FAIL: no frame captured"; fail=1; }
}
