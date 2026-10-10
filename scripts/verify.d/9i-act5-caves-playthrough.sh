scenario_name="Act 5 caves playthrough (level 94 sample hero: Crystalline Passage to the Throne of Destruction, waypoints on the way)"
a5=$tmp/act5deep
# The second half of Act 5 played by OD2_AUTOSCRIPT (see docs/PLAYTEST.md, "Act 4 and 5"): the hero travels to
# Harrogath (the quest flags are forced), takes the waypoint to the Crystalline Passage and descends the ice caves
# (Glacial Trail, Frozen Tundra, Ancients' Way) to the Arreat Summit, then the stairs of the Worldstone Keep down to
# the Throne of Destruction. The Frozen Tundra has two cave exits (one back, one on), which the presets tell apart.
# The sample character is a dead hardcore Sorceress; scripts/d2s-revive.go makes a living copy (needs D2_TABLES).
scenario_warnings_ok=1
scenario_env() {
  mkdir -p $a5/s94 $a5/wb94; rm -f $a5/s94/Hero.d2s $a5/wb94/NokkaSorc.d2s
  if [ -n "${D2_TABLES:-}" ] && go run scripts/d2s-revive.go "$D2S_SAMPLE_BODY" $a5/s94/Hero.d2s "$D2_TABLES" >/dev/null 2>&1; then
    local s="wait:1;say:resetquests;say:completequest 1 6;travel:2;expect:level=40;say:completequest 2 6;travel:3;expect:level=75;say:completequest 3 6;travel:4;expect:level=103;wait:2;say:completequest 4 2;travel:5;expect:level=109"
    s+=";wait:11;use:Waypoint;waypoint:113;expect:level=113;wait:3;say:capframe $tmp/act5-crystal.png"
    s+=";walkto:exit=115;expect:level=115;wait:3"
    s+=";walkto:exit=117;expect:level=117;wait:3;say:capframe $tmp/act5-tundra.png;kill:near=30,45"
    s+=";walkto:exit=118;expect:level=118;wait:3"
    s+=";walkto:exit=120;expect:level=120;wait:3;say:capframe $tmp/act5-summit.png"
    s+=";walkto:exit=128;expect:level=128;wait:3"
    s+=";walkto:exit=129;expect:level=129;wait:3"
    s+=";wait:11;use:Waypoint;waypoint:118;expect:level=118;wait:3"
    s+=";walkto:exit=120;expect:level=120;wait:3;walkto:exit=128;expect:level=128;wait:3;walkto:exit=129;expect:level=129;wait:3"
    s+=";walkto:exit=130;expect:level=130;wait:3"
    s+=";walkto:exit=131;expect:level=131;wait:3;say:capframe $tmp/act5-throne.png;exit"
    echo "export OD2_AUTOGAME=\"$a5/s94/Hero.d2s\" OD2_D2S_WRITEBACK=\"$a5/wb94\" OD2_AUTOSPEED=4 OD2_AUTOMONSTER_DIFF=0"
    echo "export OD2_AUTOSCRIPT='$s'"
  else
    echo "# no revived sample (D2_TABLES unset or the tool failed)" >&2
    echo "export OD2_AUTOSCRIPT='wait:1;exit'"
  fi
}
scenario_check() {
  grep -E "LEVEL CHANGE|POPULATE level|EXIT (gave up|no way)|AUTOSCRIPT RESULT" $log.txt | cut -c1-230 | tail -40
  grep -q "AUTOSCRIPT RESULT PASS" $log.txt || { echo "FAIL: the Act 5 caves playthrough did not pass"; fail=1; }
  [ -s $a5/s94/Hero.d2s ] || { echo "SKIP: no revived sample hero in this run"; return; }
  grep -E "AUTOSCRIPT step [0-9]+ FAIL" $log.txt | cut -c1-200
  for route in "from=109 to=113 .*via=waypoint" "from=113 to=115 .*via=warp" "from=115 to=117 .*via=warp" "from=117 to=118 .*via=warp" \
               "from=118 to=120 .*via=warp" "from=120 to=128 .*via=warp" "from=128 to=129 .*via=warp" "from=129 to=118 .*via=waypoint" \
               "from=129 to=130 .*via=warp" "from=130 to=131 .*via=warp"; do
    grep -qE "LEVEL CHANGE $route" $log.txt || { echo "FAIL: missing level change $route"; fail=1; }
  done
  # the cave and temple exits lead somewhere; the two exits of the Frozen Tundra go different ways
  grep -qE "exit tile style=[0-9]+ at .* leads to level 0" $log.txt && { echo "FAIL: an exit tile of Act 5 leads nowhere"; fail=1; }
  grep -qE "exit tile style=[0-9]+ at .*WestEntrance_Snow.ds1 leads to level 115" $log.txt || { echo "FAIL: the Frozen Tundra west cave does not lead to the Glacial Trail"; fail=1; }
  grep -qE "exit tile style=[0-9]+ at .*WestExit_Snow.ds1 leads to level 118" $log.txt || { echo "FAIL: the Frozen Tundra east cave does not lead to the Ancients' Way"; fail=1; }
  # the hero arrives next to the way back through the stairs of the cave levels (not at the fallback entry)
  grep -qE "LEVEL CHANGE from=113 to=115 .*arrival=\(5\.5,54\.5\)" $log.txt || { echo "FAIL: the hero did not arrive at the up stairs of the Glacial Trail"; fail=1; }
  for lvl in 113 117 118 128 129 130; do
    grep -E "POPULATE level $lvl " $log.txt | grep -qE "[1-9][0-9]* monsters" || { echo "FAIL: level $lvl got no monsters"; fail=1; }
  done
  grep -q "DEATH hero=" $log.txt && { echo "FAIL: the level 94 hero died on the way"; fail=1; }
  for s in act5-crystal act5-tundra act5-summit act5-throne; do
    [ -s $tmp/$s.png ] || { echo "FAIL: no screenshot $s.png"; fail=1; }
  done
  if grep -E "\[ERROR\]|EXIT gave up|EXIT no way found" $log.txt | grep -v "skipping missing"; then echo "FAIL: errors or a walk that gave up in the Act 5 caves log"; fail=1; fi
  if grep -E "Unknown tile|panic" $log.txt; then echo "FAIL: unknown tiles or panic in the Act 5 caves log"; fail=1; fi
}
