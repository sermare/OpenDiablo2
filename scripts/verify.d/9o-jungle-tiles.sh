scenario_name="Act 3 jungle tiles (OD2_REALMAPS=1, Lower Kurast: exact tile records, TILESTATS numeric checks, frame captured)"
# Numbers only: the log line TILESTATS level=<id> exact=<bool> cells= nofloor= ... fallbacks= comes from the generator after the
# tiles are set. The records themselves equal the emulator (d2common/d2drlg/drlgoutdoor TestOracleTiles, tiles_act23.json).
jt_dir=$tmp
scenario_env() {
  echo 'export OD2_REALMAPS=1 OD2_AUTOLEVEL=79 OD2_DRAWSTATS=1'
  echo "export OD2_AUTOSCRIPT='wait:2;say:capframe $jt_dir/9o-jungle.png;move:30,42;wait:8;say:capframe $jt_dir/9o-j2.png;move:60,40;wait:12;say:capframe $jt_dir/9o-j3.png;move:62,20;wait:12;say:capframe $jt_dir/9o-j4.png;exit'"
}
scenario_check() {
  grep -E "TILESTATS|real outdoor:|AUTOSCRIPT RESULT" $log.txt | cut -c1-240
  grep -q "AUTOSCRIPT RESULT PASS" $log.txt || { echo "FAIL: the jungle scenario did not pass"; fail=1; }
  line=$(grep -E "TILESTATS level=79 " $log.txt | tail -1)
  [ -n "$line" ] || { echo "FAIL: no TILESTATS line for level 79"; fail=1; }
  echo "$line" | grep -q "exact=true" || { echo "FAIL: level 79 did not use the exact tile records"; fail=1; }
  echo "$line" | grep -q "fallbacks=0 " || { echo "FAIL: level 79 has tile references without a DT1 tile"; fail=1; }
  [ -s $jt_dir/9o-jungle.png ] || { echo "FAIL: no frame captured"; fail=1; }
  if grep -E "Unknown tile|Could not locate tile|panic" $log.txt; then echo "FAIL: unknown tiles or panic in the jungle log"; fail=1; fi
}
