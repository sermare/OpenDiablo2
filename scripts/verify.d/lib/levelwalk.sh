# Shared by the 9s-walk-*.sh scenarios: walk the revived level 94 sample hero through levels no other scenario enters.
# Usage in a scenario file:
#   source scripts/verify.d/lib/levelwalk.sh
#   scenario_env() { lw_s=""; lw_town=<town level>; lw_into N; lw_hop M; lw_home ...; lw_env <dir> <act> "$lw_s"; }
#   scenario_check() { lw_check <dir> <levels...>; }
# lw_into N: town portal into N from anywhere; lw_hop N: walk to the exit that leads to N; lw_home: town portal back.
# The hero's vitals are restored on every arrival (the engine has no natural regeneration), and the check requires a
# LEVEL CHANGE line for every level the scenario claims.
lw_s=""
lw_town=1
lw_into() { lw_s+=";wait:3;say:spawnportal $1;wait:3;use:Portal;expect:level=$1;wait:2;say:restorevitals"; }
lw_hop()  { lw_s+=";walkto:exit=$1;expect:level=$1;wait:2;say:restorevitals"; }
lw_kill() { lw_s+=";kill:near=${1:-25},${2:-30}"; }
lw_home() { lw_s+=";wait:3;say:spawnportal $lw_town;wait:3;use:Portal;expect:level=$lw_town;wait:3;say:restorevitals"; }
# lw_env <dir> <act> <route>: the revived hero, the quests that gate the act, then the route
lw_env() {
  local d=$tmp/$1 act=$2 s
  mkdir -p $d/s94 $d/wb94; rm -f $d/s94/*.d2s(N) $d/wb94/*.d2s(N)
  if [ -n "${D2_TABLES:-}" ] && go run scripts/d2s-revive.go "$D2S_SAMPLE_BODY" $d/s94/Hero.d2s "$D2_TABLES" >/dev/null 2>&1; then
    s="wait:1;say:resetquests;say:restorevitals"
    case $act in
      2) s+=";say:completequest 1 6;travel:2;expect:level=40;say:completequest 2 5" ;;
      3) s+=";say:completequest 1 6;travel:2;expect:level=40;say:completequest 2 6;travel:3;expect:level=75" ;;
      4) s+=";say:completequest 1 6;travel:2;expect:level=40;say:completequest 2 6;travel:3;expect:level=75;say:completequest 3 6;travel:4;expect:level=103" ;;
      5) s+=";say:completequest 1 6;travel:2;expect:level=40;say:completequest 2 6;travel:3;expect:level=75;say:completequest 3 6;travel:4;expect:level=103;wait:2;say:completequest 4 2;travel:5;expect:level=109;say:completequest 5 5;wait:8" ;;
    esac
    s+="$3;wait:2;exit"
    echo "export OD2_AUTOGAME=\"$d/s94/Hero.d2s\" OD2_D2S_WRITEBACK=\"$d/wb94\" OD2_REALMAPS=1 OD2_AUTOSPEED=8 OD2_AUTOMONSTER_DIFF=0"
    echo "export OD2_AUTOSCRIPT='$s'"
  else
    echo "# no revived sample (D2_TABLES unset or the tool failed)" >&2
    echo "export OD2_AUTOSCRIPT='wait:1;exit'"
  fi
}
# lw_check <dir> <level...>: the script passed and every level was entered, with no walk that gave up
lw_check() {
  local d=$tmp/$1 l; shift
  grep -E "LEVEL CHANGE|AUTOSCRIPT step [0-9]+ FAIL|AUTOSCRIPT RESULT|EXIT (gave up|no way)" $log.txt | cut -c1-200 | tail -40
  [ -s $d/s94/Hero.d2s ] || { echo "SKIP: no revived sample hero in this run"; return; }
  grep -q "AUTOSCRIPT RESULT PASS" $log.txt || { echo "FAIL: the level walk did not pass"; fail=1; }
  for l in "$@"; do
    grep -qE "LEVEL CHANGE from=[0-9]+ to=$l " $log.txt || { echo "FAIL: level $l was never entered"; fail=1; }
  done
  if grep -E "EXIT gave up|EXIT no way found|Unknown tile|panic" $log.txt; then echo "FAIL: a walk gave up, unknown tiles or a panic"; fail=1; fi
}
