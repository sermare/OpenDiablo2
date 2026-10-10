# Shared by scripts/verify.d/9g-act3-pop-<level>.sh: the natural population of one Act 3 outdoor level.
# The game starts in the level (OD2_REALMAPS=1 OD2_AUTOLEVEL=<id>, population forced on); the "POPULATE types level N:"
# log line is checked by scripts/act3_pop_check.py against the Levels.txt row (mon1..mon10 + the minions of those
# classes, isSpawn) read from D2_TABLES. Needs D2_TABLES, else the check is skipped.
act3pop_env() {
  echo "export OD2_REALMAPS=1 OD2_AUTOLEVEL=$1 OD2_POPULATE=1"
  echo "export OD2_AUTOSCRIPT='wait:2;exit'"
}
act3pop_check() {
  local lvl=$1 min=$2 rc
  grep -E "POPULATE (level|types)|AUTOSCRIPT RESULT" $log.txt | cut -c1-260
  grep -q "AUTOSCRIPT RESULT PASS" $log.txt || { echo "FAIL: act3 population scenario for level $lvl did not pass"; fail=1; }
  python3 -I scripts/act3_pop_check.py $log.txt $lvl $min; rc=$?
  [ $rc -eq 1 ] && { echo "FAIL: level $lvl population is not legal for its Levels.txt row"; fail=1; }
  [ $rc -eq 2 ] && echo "SKIP: D2_TABLES unset"
  return 0
}
