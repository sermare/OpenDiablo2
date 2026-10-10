scenario_name="visual fidelity (screenshots of the five act towns with logged draw statistics; no Act-1 tiles in later acts, no marker lattices)"
# One game walks Act 1 -> 5 (quest flags forced, see 99-act-travel.sh) and takes a frame in every town. OD2_DRAWSTATS makes the
# renderer log what it drew (DRAWSTATS level=<id> floors=.. walls=.. ..). Screenshots stay in $tmp, never in the repo.
scenario_env() {
  [ -n "${D2S_SAMPLE_BODY:-}" ] && cp "$D2S_SAMPLE_BODY" "$save"
  local s="wait:1;say:resetquests;wait:3;say:capframe $tmp/town1.png"
  s+=";say:completequest 1 6;travel:2;expect:level=40;wait:4;say:capframe $tmp/town2.png"
  s+=";say:completequest 2 6;travel:3;expect:level=75;wait:4;say:capframe $tmp/town3.png"
  s+=";say:completequest 3 6;travel:4;expect:level=103;wait:4;say:capframe $tmp/town4.png"
  s+=";say:completequest 4 2;travel:5;expect:level=109;wait:4;say:capframe $tmp/town5.png;wait:1;exit"
  echo "export OD2_REALMAPS=1 OD2_DRAWSTATS=1"
  echo "export OD2_AUTOSCRIPT='$s'"
}
scenario_check() {
  grep -E "DRAWSTATS|AUTOSCRIPT RESULT" $log.txt | cut -c1-220 | tail -12
  grep -q "AUTOSCRIPT RESULT PASS" $log.txt || { echo "FAIL: visual fidelity script did not pass"; fail=1; }
  local lv
  for lv in 1 40 75 103 109; do
    grep -qE "DRAWSTATS level=$lv floors=[1-9][0-9]* " $log.txt || { echo "FAIL: no floors drawn in level $lv"; fail=1; }
    grep -qE "DRAWSTATS level=$lv .* lit=true" $log.txt || { echo "FAIL: level $lv was not lit by the light map"; fail=1; }
  done
  for n in 1 2 3 4 5; do
    [ -s $tmp/town$n.png ] || { echo "FAIL: no screenshot of town $n"; fail=1; }
  done
}
