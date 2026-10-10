# Shared by scripts/verify.d/9r-pop-<level>.sh: the natural population of one Act 1, 2, 4 or 5 outdoor / cave / maze level.
# Generalises act3pop.sh. The game starts in the level (OD2_REALMAPS=1 OD2_AUTOLEVEL=<id>, population forced on); the
# "POPULATE types / packs / groups" and "density rolls" log lines are checked by scripts/pop_check.py against the
# Levels.txt row (mon1..mon10, umon, NumMon, MonDen) and monstats.txt (minions, spawn, isSpawn) read from D2_TABLES:
# legal classes, at most NumMon drawn types, MonDen 0 spawns nothing, a minimum count, and the number of groups against
# the density rolls. Needs D2_TABLES, else the check is skipped.
poplevel_env() {
  echo "export OD2_REALMAPS=1 OD2_AUTOLEVEL=$1 OD2_POPULATE=1"
  echo "export OD2_AUTOSCRIPT='wait:2;exit'"
}
# poplevel_check <level> <min monsters> [<density rolls> [<rooms>]]: the optional numbers are the oracle's for the sample
# hero's seed (0x101D574A, normal): the density rolls of the real game's logic regions (testdata/logic_regions.json) and
# the number of rooms, which the engine must reproduce exactly.
poplevel_check() {
  local lvl=$1 min=$2 rolls=${3:-} rooms=${4:-} rc row
  # scripts/verify.d/lib/pop_rolls.txt: "<level> <rooms> <density rolls>" of the real game for the sample seed, from the emulator
  if [ -z "$rolls" ] && [ -f scripts/verify.d/lib/pop_rolls.txt ]; then
    row=$(grep "^$lvl " scripts/verify.d/lib/pop_rolls.txt | head -1)
    if [ -n "$row" ]; then rooms=${rooms:-$(echo $row | cut -d' ' -f2)}; rolls=$(echo $row | cut -d' ' -f3); fi
  fi
  grep -E "POPULATE (level|types|packs|groups)|AUTOSCRIPT RESULT|density rolls" $log.txt | cut -c1-260
  grep -q "AUTOSCRIPT RESULT PASS" $log.txt || { echo "FAIL: population scenario for level $lvl did not pass"; fail=1; }
  if [ -n "$rolls" ]; then
    grep -E "population: [0-9]+ rooms" $log.txt | grep -q " $rolls density rolls at" || { echo "FAIL: level $lvl density rolls are not $rolls (the real game's logic regions)"; fail=1; }
  fi
  if [ -n "$rooms" ]; then
    grep -E "population: [0-9]+ rooms" $log.txt | grep -q "population: $rooms rooms" || { echo "FAIL: level $lvl does not have $rooms rooms"; fail=1; }
  fi
  python3 -I scripts/pop_check.py $log.txt $lvl $min; rc=$?
  [ $rc -eq 1 ] && { echo "FAIL: level $lvl population does not match its Levels.txt row"; fail=1; }
  [ $rc -eq 2 ] && echo "SKIP: D2_TABLES unset"
  return 0
}
