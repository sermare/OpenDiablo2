scenario_name="ambient environment (Blood Moor day then night: music, ambience, events)"
scenario_env() { echo 'export OD2_AUTOAMBIENT="Blood Moor" OD2_AUTOAMBIENT_SECONDS=8 OD2_AUTOAMBIENT_SPEED=10'; }
scenario_check() {
  grep -E "AUTOAMBIENT (phase|summary)" $log.txt | cut -c1-260
  grep -q "AUTOAMBIENT phase=2 .*night=false .*ambience=scene_wilderness_day" $log.txt || { echo "FAIL: no day ambience"; fail=1; }
  grep -q "AUTOAMBIENT phase=4 .*night=true .*ambience=scene_wilderness_night" $log.txt || { echo "FAIL: no night ambience"; fail=1; }
  grep -q "SOUNDAT kind=ambient-event" $log.txt || { echo "FAIL: no ambient event played"; fail=1; }
  grep -q "AUTOAMBIENT summary" $log.txt || { echo "FAIL: no summary"; fail=1; }
}
