scenario_name="w44 character sheet: attribute/maxima/resist values are tinted per the exe (frame of the panel for the real level 94 sorceress)"
scenario_turbo=1
# Frame: $tmp/w44cs-char.png (read by eye, not committed): item-boosted attributes and maxima blue, resists white/gold.
scenario_env() {
  echo "export OD2_AUTOSCRIPT='wait:2;panel:character;wait:1;say:capframe $tmp/w44cs-char.png;panel:close;exit'"
}
scenario_check() {
  grep -E "AUTOSCRIPT step [0-9]+ FAIL|AUTOSCRIPT RESULT|PANEL character" $log.txt | cut -c1-300 | tail -6
  grep -q "AUTOSCRIPT RESULT PASS" $log.txt || { echo "FAIL: script did not pass"; fail=1; }
  [ -s $tmp/w44cs-char.png ] || { echo "FAIL: no frame captured"; fail=1; }
}
