scenario_name="Walkability (level 94 sample hero: lava, water and bridges of Blood Moor and the River of Flame)"
w9=$tmp/walk
# The DT1 sub-tile flags of a lava floor are all zero; the original blocks the lava through bit 17 of the DS1 floor
# cell (record flag 0x40 -> collision bit 1 in COLLISION_ApplyTileFlagsToGrid, see d2mapengine/map_tile.go).
# The console command "walkprobe" (d2gamescreen/game_walkprobe.go) counts the lava/water tiles the hero can walk to
# (flood fill of the player collision mask), orders a walk onto the nearest lava/water tile and logs whether the route
# ends there. Expected: 0 reachable lava or water tiles, the route never ends on lava, and the bridge room of the
# River of Flame is crossed (walkto:exit=108 passes).
scenario_warnings_ok=1
scenario_env() {
  mkdir -p $w9/s94 $w9/wb94; rm -f $w9/s94/Hero.d2s $w9/wb94/NokkaSorc.d2s
  if [ -n "${D2_TABLES:-}" ] && go run scripts/d2s-revive.go "$D2S_SAMPLE_BODY" $w9/s94/Hero.d2s "$D2_TABLES" >/dev/null 2>&1; then
    local s="wait:1;say:resetquests;say:completequest 1 6;walkto:exit=2;expect:level=2;wait:2;say:walkprobe;say:capframe $tmp/walk-bloodmoor.png"
    s+=";say:completequest 2 6;say:completequest 3 6;say:travelfree 1;travel:4;expect:level=103;wait:3"
    s+=";walkto:exit=104;expect:level=104;walkto:exit=105;expect:level=105;walkto:exit=106;expect:level=106"
    s+=";walkto:exit=107;expect:level=107;wait:3;say:walkprobe;say:capframe $tmp/walk-river.png"
    s+=";walkto:exit=108;expect:level=108;wait:3;say:walkprobe;say:capframe $tmp/walk-chaos.png;exit"
    echo "export OD2_AUTOGAME=\"$w9/s94/Hero.d2s\" OD2_D2S_WRITEBACK=\"$w9/wb94\" OD2_REALMAPS=1 OD2_AUTOSPEED=8 OD2_AUTOMONSTER_DIFF=0 OD2_NOPOPULATE=1"
    echo "export OD2_AUTOSCRIPT='$s'"
  else
    echo "# no revived sample (D2_TABLES unset or the tool failed)" >&2
    echo "export OD2_AUTOSCRIPT='wait:1;exit'"
  fi
}
scenario_check() {
  grep -E "WALKPROBE|LEVEL CHANGE|AUTOSCRIPT RESULT" $log.txt | cut -c1-230 | tail -30
  [ -s $w9/s94/Hero.d2s ] || { echo "SKIP: no revived sample hero in this run"; return; }
  grep -q "AUTOSCRIPT RESULT PASS" $log.txt || { echo "FAIL: the walkability walk did not pass"; fail=1; }
  for lvl in 2 107 108; do
    grep -qE "WALKPROBE level=$lvl lava=" $log.txt || { echo "FAIL: no walk probe in level $lvl"; fail=1; }
  done
  # the River of Flame is made of lava: there must be lava tiles, none reachable, and the bridge must be walkable
  grep -qE "WALKPROBE level=107 lava=[1-9][0-9]* lavaReachable=0 " $log.txt || { echo "FAIL: River of Flame lava missing or walkable"; fail=1; }
  grep -qE "WALKPROBE level=107 .*bridge=[1-9][0-9]* bridgeReachable=[1-9]" $log.txt || { echo "FAIL: no walkable bridge in the River of Flame"; fail=1; }
  grep -E "WALKPROBE level=[0-9]+ lava=" $log.txt | grep -vE "lavaReachable=0 water=[0-9]+ waterReachable=0 " && { echo "FAIL: the hero can walk on lava or water"; fail=1; }
  grep -E "WALKPROBE .*routeEndsOnLava=true" $log.txt && { echo "FAIL: a walk order ended on lava or water"; fail=1; }
  [ -s $tmp/walk-river.png ] || { echo "FAIL: no screenshot of the River of Flame"; fail=1; }
  if grep -E "Unknown tile|panic" $log.txt; then echo "FAIL: unknown tiles or panic"; fail=1; fi
}
