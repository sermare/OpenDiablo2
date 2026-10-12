scenario_name="w47 ui3: character sheet with unspent points (Stat Points / Remaining captions, attack rating caption) and the merc screen (value rows right aligned)"
scenario_turbo=1
scenario_warnings_ok=1
# Frames (read by eye, not committed): $tmp/w47-char.png and $tmp/w47-merc.png. Run it at 800x600 (the column of the
# interface is the same at any size, the frame is just easier to read): OD2_DISPLAY=800x600 is set below.
scenario_env() {
  echo "export OD2_DISPLAY=800x600 OD2_AUTOSCRIPT='wait:2;say:levelup 1;wait:1;panel:character;wait:1;say:capframe $tmp/w47-char.png;wait:1;panel:close;exit'"
}
scenario_check() {
  grep -E "AUTOSCRIPT step [0-9]+ FAIL|AUTOSCRIPT RESULT|PANEL character" $log.txt | cut -c1-300 | tail -6
  grep -q "AUTOSCRIPT RESULT PASS" $log.txt || { echo "FAIL: script did not pass"; fail=1; }
  [ -s $tmp/w47-char.png ] || { echo "FAIL: no frame captured"; fail=1; }
}
