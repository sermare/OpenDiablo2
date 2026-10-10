scenario_name="town tiles (OD2_REALMAPS=1: the five act towns are built from the exact tile records of the real game; TILESTATS must show no fallback and no floor-less cell; screenshots)"
# The tile records of the towns equal the emulator's (d2common/d2drlg drlgoutdoor TestOracleTiles, tiles_towns.json).
# The log line TILESTATS level=<id> exact=<bool> cells= nofloor= floors= walls= shadows= fallbacks= is written by the town
# generators after the tiles are set; fallbacks counts tile references with no DT1 tile (the renderer would log
# "Could not locate tile"), nofloor the cells that draw no ground.
tt_dir=$tmp
scenario_env() {
  [ -n "${D2S_SAMPLE_BODY:-}" ] && cp "$D2S_SAMPLE_BODY" "$save"
  local s="wait:1;say:resetquests;say:completequest 1 6;say:capframe $tt_dir/9n-a1.png"
  s+=";travel:2;expect:level=40;wait:3;say:capframe $tt_dir/9n-a2.png"
  s+=";say:completequest 2 6;travel:3;expect:level=75;wait:3;say:capframe $tt_dir/9n-a3.png;move:npc=Alkor;wait:14;say:capframe $tt_dir/9n-a3-alkor.png"
  s+=";say:completequest 3 6;travel:4;expect:level=103;wait:3;say:capframe $tt_dir/9n-a4.png"
  s+=";say:completequest 4 2;travel:5;expect:level=109;wait:3;say:capframe $tt_dir/9n-a5.png;exit"
  echo "export OD2_REALMAPS=1 OD2_AUTOSCRIPT=\"$s\""
}
scenario_check() {
  grep -E "TILESTATS|AUTOSCRIPT RESULT" $log.txt | sed 's/.*\(TILESTATS.*\)/\1/' | cut -c1-200
  grep -q "AUTOSCRIPT RESULT PASS" $log.txt || { echo "FAIL: the town tour did not pass"; fail=1; }
  for l in 1 40 75 103 109; do
    line=$(grep -E "TILESTATS level=$l " $log.txt | tail -1)
    [ -n "$line" ] || { echo "FAIL: no TILESTATS line for town $l"; fail=1; continue; }
    echo "$line" | grep -q "exact=true" || { echo "FAIL: town $l did not use the exact tile records"; fail=1; }
    echo "$line" | grep -q "nofloor=0 " || { echo "FAIL: town $l has cells without a floor"; fail=1; }
    echo "$line" | grep -q "fallbacks=0 " || { echo "FAIL: town $l has tile references without a DT1 tile (fallbacks)"; fail=1; }
  done
  for s in a1 a2 a3 a3-alkor a4 a5; do [ -s $tt_dir/9n-$s.png ] || { echo "FAIL: no screenshot 9n-$s.png"; fail=1; }; done
  if grep -E "Unknown tile|Could not locate tile|panic" $log.txt; then echo "FAIL: unknown tiles or panic in the town log"; fail=1; fi
}
