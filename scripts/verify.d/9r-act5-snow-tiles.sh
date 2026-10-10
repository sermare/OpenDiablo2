scenario_name="Act 5 snow maze tiles (OD2_REALMAPS=1, Glacial Trail 115: exact tile records, TILESTATS numeric checks, frame captured)"
# Numbers only. Before the exact maze build the Act 5 snow caves drew the dark mossy floor diamonds of the stamped
# (style, sequence, type) lookup; the records now equal the emulator's (TestMazeTilesGolden, level 115).
scenario_env() {
  echo 'export OD2_REALMAPS=1 OD2_AUTOLEVEL=115'
  echo "export OD2_AUTOSCRIPT='wait:2;say:capframe $tmp/9r-act5.png;wait:1;exit'"
}
scenario_check() {
  grep -E "TILESTATS|maze tiles:|AUTOSCRIPT RESULT" $log.txt | cut -c1-240
  grep -q "AUTOSCRIPT RESULT PASS" $log.txt || { echo "FAIL: the Act 5 snow scenario did not pass"; fail=1; }
  line=$(grep -E "TILESTATS level=115 " $log.txt | tail -1)
  [ -n "$line" ] || { echo "FAIL: no TILESTATS line for level 115"; fail=1; }
  echo "$line" | grep -q "exact=true" || { echo "FAIL: level 115 did not use the exact tile records"; fail=1; }
  echo "$line" | grep -q "fallbacks=0 " || { echo "FAIL: level 115 has tile references without a DT1 tile"; fail=1; }
  [ -s $tmp/9r-act5.png ] || { echo "FAIL: no frame captured"; fail=1; }
  if grep -E "Unknown tile|Could not locate tile|panic" $log.txt; then echo "FAIL: unknown tiles or panic in the Act 5 log"; fail=1; fi
}
