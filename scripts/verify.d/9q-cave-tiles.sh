scenario_name="cave tiles (OD2_REALMAPS=1, Cave Level 1: maze rooms built from the exact tile records, TILESTATS numeric checks, frame captured)"
# Numbers only: TILESTATS level=<id> exact=<bool> ... is written by the maze generator after the tiles are set. The records equal
# the emulator's for 92% of the maze rooms of the 70 maze levels measured (d2common/d2drlg/drlgoutdoor TestMazeTilesGolden,
# testdata/tiles_maze.json); docs/tile-diagnosis.md has the rest.
cv_dir=$tmp
scenario_env() {
  echo 'export OD2_REALMAPS=1 OD2_AUTOLEVEL=9'
  echo "export OD2_AUTOSCRIPT='wait:2;say:capframe $cv_dir/9q-cave.png;wait:1;exit'"
}
scenario_check() {
  grep -E "TILESTATS|maze tiles:|AUTOSCRIPT RESULT" $log.txt | cut -c1-240
  grep -q "AUTOSCRIPT RESULT PASS" $log.txt || { echo "FAIL: the cave scenario did not pass"; fail=1; }
  line=$(grep -E "TILESTATS level=9 " $log.txt | tail -1)
  [ -n "$line" ] || { echo "FAIL: no TILESTATS line for level 9"; fail=1; }
  echo "$line" | grep -q "exact=true" || { echo "FAIL: level 9 did not use the exact tile records"; fail=1; }
  echo "$line" | grep -q "fallbacks=0 " || { echo "FAIL: level 9 has tile references without a DT1 tile"; fail=1; }
  [ -s $cv_dir/9q-cave.png ] || { echo "FAIL: no frame captured"; fail=1; }
  if grep -E "Unknown tile|Could not locate tile|panic" $log.txt; then echo "FAIL: unknown tiles or panic in the cave log"; fail=1; fi
}
