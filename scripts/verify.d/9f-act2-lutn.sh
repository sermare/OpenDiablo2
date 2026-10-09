scenario_name="Act 2 with the other Lut Gholein (OD2_AUTOMAPSEED=1: LutN, Rocky Waste to the north of the town): out of the gate and back"
a2=$tmp/act2n
# The town comes in two presets by the world layout: LutW (Rocky Waste to the west, 9e) and LutN (to the north).
# The sample hero's seed gives LutW; this seed gives LutN. Short walk: out of the north gate and back.
scenario_env() {
  mkdir -p $a2/s94 $a2/wb94; rm -f $a2/s94/*.d2s $a2/wb94/*.d2s
  if [ -n "${D2_TABLES:-}" ] && go run scripts/d2s-revive.go "$D2S_SAMPLE_BODY" $a2/s94/Hero.d2s "$D2_TABLES" >/dev/null 2>&1; then
    echo "export OD2_AUTOGAME=\"$a2/s94/Hero.d2s\" OD2_D2S_WRITEBACK=\"$a2/wb94\" OD2_REALMAPS=1 OD2_AUTOSPEED=3 OD2_AUTOMONSTER_DIFF=0 OD2_AUTOMAPSEED=1"
    echo "export OD2_AUTOSCRIPT='wait:1;say:resetquests;say:completequest 1 6;travel:2;expect:level=40;wait:4;say:capframe $tmp/act2n-town.png;walkto:exit=41;expect:level=41;wait:3;say:capframe $tmp/act2n-rocky.png;walkto:exit=40;expect:level=40;exit'"
  else
    echo "# no revived sample (D2_TABLES unset or the tool failed)" >&2
    echo "export OD2_AUTOSCRIPT='wait:1;exit'"
  fi
}
scenario_check() {
  grep -E "act town|LEVEL CHANGE|AUTOSCRIPT RESULT" $log.txt | cut -c1-230 | tail -8
  grep -q "AUTOSCRIPT RESULT PASS" $log.txt || { echo "FAIL: the LutN walk did not pass"; fail=1; }
  [ -s $a2/s94/Hero.d2s ] || { echo "SKIP: no revived sample hero in this run"; return; }
  grep -qE "act town: level 40 .* Act2/Town/LutN\.ds1" $log.txt || { echo "FAIL: the seed did not give the LutN town"; fail=1; }
  for route in "from=40 to=41 .*via=edge" "from=41 to=40 .*via=edge"; do
    grep -qE "LEVEL CHANGE $route" $log.txt || { echo "FAIL: missing level change $route"; fail=1; }
  done
  for s in town rocky; do [ -s $tmp/act2n-$s.png ] || { echo "FAIL: no screenshot act2n-$s.png"; fail=1; }; done
  if grep -E "Unknown tile|panic" $log.txt; then echo "FAIL: unknown tiles or panic in the LutN log"; fail=1; fi
}
