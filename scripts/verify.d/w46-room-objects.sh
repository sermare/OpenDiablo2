scenario_name="w46 random room objects (OBJRGN_SpawnRoomObjects port): Blood Moor and the Den of Evil get chests/barrels/shrines by ObjGrp density"
scenario_warnings_ok=1
# Wave 46. The engine used to place objects only from DS1 presets; d2object.Scatter now runs for every room of a
# real-map level right before its natural monsters (d2mapgen real_objects.go). The hero stands in the Blood Moor (2),
# walks into the Den of Evil (8); each level logs a "ROOMOBJ level N: K objects in R rooms" census at generation and a
# "ROOMOBJ-SPAWN level N: made=M of K planned" line when the game screen creates them. Counts must be positive on the
# outdoor level and stay below the density bound (a room makes at most 12*(((w*h>>7)*127)>>8)+4 of one kind; rooms are
# ~64x64 sub-tiles, so 200 per room is far above anything legal) and the entity creation must lose nothing big.
scenario_env() {
  echo 'export OD2_AUTOLEVEL=2 OD2_AUTOSPEED=3 OD2_AUTOMONSTER_DIFF=0 OD2_ROOM_OBJECTS=1'
  echo "export OD2_AUTOSCRIPT='wait:3;walkto:exit=8;expect:level=8;wait:3;exit'"
}
scenario_check() {
  grep -E "ROOMOBJ|AUTOSCRIPT RESULT|LEVEL CHANGE" $log.txt | cut -c1-300 | tail -20
  grep -q "AUTOSCRIPT RESULT PASS" $log.txt || { echo "FAIL: room objects scenario did not pass"; fail=1; }
  local lvl n rooms made planned
  for lvl in 2 8; do
    line=$(grep -m1 "ROOMOBJ level $lvl:" $log.txt)
    [ -n "$line" ] || { echo "FAIL: no ROOMOBJ census for level $lvl"; fail=1; continue; }
    n=$(echo "$line" | sed -E 's/.*ROOMOBJ level [0-9]+: ([0-9]+) objects in ([0-9]+) rooms.*/\1/')
    rooms=$(echo "$line" | sed -E 's/.*ROOMOBJ level [0-9]+: ([0-9]+) objects in ([0-9]+) rooms.*/\2/')
    [ "$n" -le $((rooms * 200)) ] || { echo "FAIL: level $lvl made $n objects in $rooms rooms (above the density bound)"; fail=1; }
    [ "$lvl" = 2 ] && { [ "$n" -gt 0 ] || { echo "FAIL: Blood Moor got no random room objects"; fail=1; }; }
    sp=$(grep -m1 "ROOMOBJ-SPAWN level $lvl:" $log.txt)
    if [ -n "$sp" ]; then
      made=$(echo "$sp" | sed -E 's/.*made=([0-9]+) of ([0-9]+) planned.*/\1/')
      planned=$(echo "$sp" | sed -E 's/.*made=([0-9]+) of ([0-9]+) planned.*/\2/')
      [ "$made" -ge $((planned * 9 / 10)) ] || { echo "FAIL: level $lvl created only $made of $planned planned objects"; fail=1; }
    elif [ "$n" -gt 0 ]; then
      echo "FAIL: level $lvl planned $n objects but the game screen made none"; fail=1
    fi
  done
  if grep -E "panic" $log.txt; then echo "FAIL: panic in the room objects log"; fail=1; fi
}
scenario_timeout=600
