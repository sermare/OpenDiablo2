scenario_name="Act 3 gate (Travincal stairs stay sealed until the Compelling Orb is smashed, then Durance 1 -> 2 -> 3)"
a3=$tmp/act3
# Needs the revived level-94 sample hero (D2_TABLES + D2S_SAMPLE_BODY, see 9e-act2-playthrough.sh).
# 1. Meshif takes the hero to Kurast Docks, the Travincal waypoint (forced on) takes him to level 83.
# 2. While Khalim's Will is not done the stairs to the Durance are refused (the walk step may time out: expected).
# 3. After `completequest 3 2` the same stairs work; Durance 1 -> 2 -> 3 follow by their stairs.
scenario_warnings_ok=1
scenario_env() {
  mkdir -p $a3/s94 $a3/wb94; rm -f $a3/s94/*.d2s $a3/wb94/*.d2s
  if [ -n "${D2_TABLES:-}" ] && go run scripts/d2s-revive.go "$D2S_SAMPLE_BODY" $a3/s94/Hero.d2s "$D2_TABLES" >/dev/null 2>&1; then
    local s="wait:1;say:resetquests;say:completequest 2 6;travel:3;expect:level=75;wait:3;say:setwaypoint 83 1"
    s+=";use:Waypoint;waypoint:83;expect:level=83;wait:3"
    s+=";say:completequest 3 2;walkto:exit=100;expect:level=100;wait:3"
    s+=";walkto:exit=101;expect:level=101;wait:3;walkto:exit=102;expect:level=102;wait:3;exit"
    echo "export OD2_AUTOGAME=\"$a3/s94/Hero.d2s\" OD2_D2S_WRITEBACK=\"$a3/wb94\" OD2_REALMAPS=1 OD2_AUTOSPEED=3 OD2_AUTOMONSTER_DIFF=0 OD2_NOPOPULATE=1"
    echo "export OD2_AUTOSCRIPT='$s'"
  else
    echo "# no revived sample (D2_TABLES unset or the tool failed)" >&2
    echo "export OD2_AUTOSCRIPT='wait:1;exit'"
  fi
}
scenario_check() {
  grep -E "LEVEL built|warp refused|AUTOSCRIPT RESULT" $log.txt | cut -c1-230 | tail -20
  [ -s $a3/s94/Hero.d2s ] || { echo "SKIP: no revived sample hero in this run"; return; }
  grep -q "AUTOSCRIPT RESULT PASS" $log.txt || { echo "FAIL: the Act 3 gate scenario did not pass"; fail=1; }
  for lvl in 83 100 101 102; do
    grep -q "LEVEL built: level $lvl " $log.txt || { echo "FAIL: level $lvl was never built"; fail=1; }
  done
}
