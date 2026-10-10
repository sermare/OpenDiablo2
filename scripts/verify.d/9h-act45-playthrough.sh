scenario_timeout=1200
scenario_name="Act 4 playthrough and the start of Act 5 (level 94 sample hero: Fortress to the Chaos Sanctuary and back, Harrogath to the Crystalline Passage)"
a45=$tmp/act45
# Played by OD2_AUTOSCRIPT (see docs/PLAYTEST.md, "Act 4 and 5"). Act 4 walks out of the Pandemonium Fortress across
# the seamless borders of Outer Steppes, Plains of Despair and City of the Damned (real level changes, fights on the
# way), takes the lava warp into the River of Flame (level 107, a maze with preset rooms), walks the bridge room into
# the Chaos Sanctuary and back to the City of the Damned, then takes the waypoint back to the Fortress. Act 5 travels
# with the portal rules (the quest flag of Act 4 is forced), leaves Harrogath through the Bloody Foothills, Frigid
# Highlands and Arreat Plateau and enters the Crystalline Passage. 9i goes on down the ice caves to the Throne.
# The sample character is a dead hardcore Sorceress; scripts/d2s-revive.go makes a living copy (needs D2_TABLES).
# "KILL giving up for now" / "KILL time limit" lines are part of fights in big outdoor levels; errors still fail it.
scenario_warnings_ok=1
scenario_env() {
  mkdir -p $a45/s94 $a45/wb94; rm -f $a45/s94/Hero.d2s $a45/wb94/*.d2s
  if [ -n "${D2_TABLES:-}" ] && make_hero $a45/s94/Hero.d2s; then
    local s="wait:1;say:resetquests;say:completequest 1 6;travel:2;expect:level=40;say:completequest 2 6;travel:3;expect:level=75;say:completequest 3 6;travel:4;expect:level=103;wait:3;say:capframe $tmp/act4-fortress.png"
    # Act 4: the chain of outdoor levels by their borders, the lava warp, the maze, the bridge room, and back
    s+=";walkto:exit=104;expect:level=104;wait:3;say:capframe $tmp/act4-steppes.png;kill:near=30,45"
    s+=";walkto:exit=105;expect:level=105;wait:3"
    s+=";walkto:exit=106;expect:level=106;wait:3;say:capframe $tmp/act4-city.png"
    s+=";walkto:exit=107;expect:level=107;wait:3;say:capframe $tmp/act4-river.png"
    s+=";walkto:exit=108;expect:level=108;wait:3;say:capframe $tmp/act4-chaos.png"
    s+=";walkto:exit=107;expect:level=107;walkto:exit=106;expect:level=106"
    s+=";wait:11;use:Waypoint;waypoint:103;expect:level=103;wait:2"
    # Act 5
    s+=";say:completequest 4 2;travel:5;expect:level=109;wait:3;say:capframe $tmp/act5-harrogath.png"
    s+=";walkto:exit=110;expect:level=110;wait:3;say:capframe $tmp/act5-foothills.png"
    s+=";walkto:exit=111;expect:level=111;wait:3;kill:near=30,45"
    s+=";walkto:exit=112;expect:level=112;wait:3"
    s+=";walkto:exit=113;expect:level=113;wait:3;say:capframe $tmp/act5-crystal.png;exit"
    echo "export OD2_AUTOGAME=\"$a45/s94/Hero.d2s\" OD2_D2S_WRITEBACK=\"$a45/wb94\" OD2_AUTOSPEED=4 OD2_AUTOMONSTER_DIFF=0"
    echo "export OD2_AUTOSCRIPT='$s'"
  else
    echo "# no revived sample (D2_TABLES unset or the tool failed)" >&2
    echo "export OD2_AUTOSCRIPT='wait:1;exit'"
  fi
}
scenario_check() {
  grep -E "act town|LEVEL CHANGE|POPULATE level|EXIT (gave up|no way)|AUTOSCRIPT RESULT" $log.txt | cut -c1-230 | tail -40
  grep -q "AUTOSCRIPT RESULT PASS" $log.txt || { echo "FAIL: the Act 4/5 playthrough did not pass"; fail=1; }
  [ -s $a45/s94/Hero.d2s ] || { echo "SKIP: no revived sample hero in this run"; return; }
  grep -E "AUTOSCRIPT step [0-9]+ FAIL" $log.txt | cut -c1-200
  # the towns are built from their presets
  grep -qE "real preset: level 103 .*Act4/Fort/Fortress.ds1" $log.txt || { echo "FAIL: the Pandemonium Fortress was not built from its preset"; fail=1; }
  grep -qE "real preset: level 109 .*townWest.ds1" $log.txt || { echo "FAIL: Harrogath was not built from its preset"; fail=1; }
  # every level change of the route happened through the real exit: the seamless borders (edge), the lava warp,
  # the walk-through bridge exit, the cave stairs of Act 5, the waypoint home
  for route in "from=103 to=104 .*via=edge" "from=104 to=105 .*via=edge" "from=105 to=106 .*via=edge" "from=106 to=107 .*via=warp" \
               "from=107 to=108 .*via=warp" "from=108 to=107 .*via=warp" "from=107 to=106 .*via=warp" "from=106 to=103 .*via=waypoint" \
               "from=103 to=109 .*actChange=true" "from=109 to=110 .*via=edge" "from=110 to=111 .*via=edge" "from=111 to=112 .*via=edge" \
               "from=112 to=113 .*via=warp"; do
    grep -qE "LEVEL CHANGE $route" $log.txt || { echo "FAIL: missing level change $route"; fail=1; }
  done
  # the exit presets lead somewhere (the lava warp and the Arreat Plateau cave led to level 0)
  grep -qE "exit tile style=[0-9]+ at .* leads to level 0" $log.txt && { echo "FAIL: an exit tile of Act 4/5 leads nowhere"; fail=1; }
  grep -qE "exit tile style=[0-9]+ at .*WarpLava1.ds1 leads to level 107" $log.txt || { echo "FAIL: the City of the Damned lava warp does not lead to the River of Flame"; fail=1; }
  grep -qE "exit tile style=[0-9]+ at .*Entry1.ds1 leads to level 107" $log.txt || { echo "FAIL: the Chaos Sanctuary entry does not lead back to the River of Flame"; fail=1; }
  grep -qE "exit tile style=[0-9]+ at .*WestEntrance_Dirt.ds1 leads to level 113" $log.txt || { echo "FAIL: the Arreat Plateau cave does not lead to the Crystalline Passage"; fail=1; }
  # the River of Flame is the maze with its preset rooms (34 rooms, one stamp each)
  grep -qE "real maze: level 107 type 28 .*34 rooms, 34 stamps placed" $log.txt || { echo "FAIL: the River of Flame maze was not built (34 rooms expected)"; fail=1; }
  # the hero arrives next to the way back (the south warp), not at the fallback entry of the maze
  grep -qE "LEVEL CHANGE from=106 to=107 .*arrival=\(59\.5,129\.5\)" $log.txt || { echo "FAIL: the hero did not arrive at the south warp of the River of Flame"; fail=1; }
  # the levels were populated from Levels.txt and the hero fought
  for lvl in 104 105 106 107 111 113; do
    grep -E "POPULATE level $lvl " $log.txt | grep -qE "[1-9][0-9]* monsters" || { echo "FAIL: level $lvl got no monsters"; fail=1; }
  done
  grep -q "MONSTER death" $log.txt || { echo "FAIL: nothing was killed"; fail=1; }
  grep -q "DEATH hero=" $log.txt && { echo "FAIL: the level 94 hero died on the way"; fail=1; }
  # the exported .d2s says Act 5
  last=$(grep "D2S EXPORT reparse" $log.txt | tail -1)
  echo "$last" | grep -q " act=5 " || { echo "FAIL: the exported .d2s of a hero in Act 5 is not an Act 5 save: $last" | cut -c1-200; fail=1; }
  for s in act4-fortress act4-steppes act4-city act4-river act4-chaos act5-harrogath act5-foothills act5-crystal; do
    [ -s $tmp/$s.png ] || { echo "FAIL: no screenshot $s.png"; fail=1; }
  done
  if grep -E "\[ERROR\]|EXIT gave up|EXIT no way found" $log.txt | grep -v "skipping missing"; then echo "FAIL: errors or a walk that gave up in the Act 4/5 log"; fail=1; fi
  if grep -E "Unknown tile|panic" $log.txt; then echo "FAIL: unknown tiles or panic in the Act 4/5 log"; fail=1; fi
}
