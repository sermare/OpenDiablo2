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
poplevel_check() {
  local lvl=$1 min=$2 rc
  grep -E "POPULATE (level|types|packs|groups)|AUTOSCRIPT RESULT|density rolls" $log.txt | cut -c1-260
  grep -q "AUTOSCRIPT RESULT PASS" $log.txt || { echo "FAIL: population scenario for level $lvl did not pass"; fail=1; }
  python3 -I scripts/pop_check.py $log.txt $lvl $min; rc=$?
  [ $rc -eq 1 ] && { echo "FAIL: level $lvl population does not match its Levels.txt row"; fail=1; }
  [ $rc -eq 2 ] && echo "SKIP: D2_TABLES unset"
  return 0
}
