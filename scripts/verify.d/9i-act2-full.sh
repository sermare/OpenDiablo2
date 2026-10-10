scenario_name="Act 2 full (desert to the Valley of Snakes, Maggot Lair, Ancient Tunnels, Claw Viper Temple)"
a2=$tmp/act2full
# The rest of Act 2 after 9e: Dry Hills -> Far Oasis (43) -> Lost City (44) -> Valley of Snakes (45) over the seamless borders,
# the dungeon behind each of them (Maggot Lair 62-64, Ancient Tunnels 65, Claw Viper Temple 58 and 61), and the way back.
# Fights happen on the way (the hero walks through what attacks him); the long clears are 9e's job.
scenario_warnings_ok=1
scenario_env() {
  mkdir -p $a2/s94 $a2/wb94; rm -f $a2/s94/*.d2s $a2/wb94/*.d2s
  if [ -n "${D2_TABLES:-}" ] && make_hero $a2/s94/Hero.d2s; then
    local s="wait:1;say:resetquests;say:completequest 1 6;travel:2;expect:level=40"
    s+=";walkto:exit=41;expect:level=41;walkto:exit=42;expect:level=42;walkto:exit=43;expect:level=43;wait:3;say:capframe $tmp/act2-oasis.png"
    s+=";walkto:exit=62;expect:level=62;wait:3;walkto:exit=63;expect:level=63;walkto:exit=64;expect:level=64;walkto:exit=63;expect:level=63;walkto:exit=62;expect:level=62;walkto:exit=43;expect:level=43"
    s+=";walkto:exit=44;expect:level=44;wait:3;say:capframe $tmp/act2-lostcity.png"
    s+=";walkto:exit=65;expect:level=65;wait:3;walkto:exit=44;expect:level=44"
    s+=";walkto:exit=45;expect:level=45;wait:3;say:capframe $tmp/act2-valley.png"
    s+=";walkto:exit=58;expect:level=58;wait:3;walkto:exit=61;expect:level=61;wait:3;walkto:exit=58;expect:level=58;walkto:exit=45;expect:level=45;exit"
    echo "export OD2_AUTOGAME=\"$a2/s94/Hero.d2s\" OD2_D2S_WRITEBACK=\"$a2/wb94\" OD2_REALMAPS=1 OD2_AUTOSPEED=3 OD2_AUTOMONSTER_DIFF=0"
    echo "export OD2_AUTOSCRIPT='$s'"
  else
    echo "# no revived sample (D2_TABLES unset or the tool failed)" >&2
    echo "export OD2_AUTOSCRIPT='wait:1;exit'"
  fi
}
scenario_check() {
  grep -E "LEVEL CHANGE|EXIT |AUTOSCRIPT step [0-9]+ FAIL|AUTOSCRIPT RESULT" $log.txt | cut -c1-200 | tail -40
  grep -q "AUTOSCRIPT RESULT PASS" $log.txt || { echo "FAIL: the Act 2 walk did not pass"; fail=1; }
  [ -s $a2/s94/Hero.d2s ] || { echo "SKIP: no revived sample hero in this run"; return; }
  for route in "from=42 to=43 .*via=edge" "from=43 to=62 .*via=warp" "from=62 to=63" "from=63 to=64" "from=64 to=63" "from=43 to=44 .*via=edge" "from=44 to=65 .*via=warp" \
               "from=65 to=44" "from=44 to=45 .*via=edge" "from=45 to=58 .*via=warp" "from=58 to=61" "from=61 to=58" "from=58 to=45"; do
    grep -qE "LEVEL CHANGE $route" $log.txt || { echo "FAIL: missing level change $route"; fail=1; }
  done
  for s in oasis lostcity valley; do [ -s $tmp/act2-$s.png ] || { echo "FAIL: no screenshot act2-$s.png"; fail=1; }; done
  if grep -E "EXIT gave up|EXIT no way found|Unknown tile|panic" $log.txt; then echo "FAIL: a walk gave up or unknown tiles"; fail=1; fi
}
