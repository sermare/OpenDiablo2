scenario_timeout=1200
scenario_name="Act 3 playthrough (level 94 sample hero: Kurast Docks NPCs, jungle, Kurast, Travincal, the dungeon entrances and a town portal)"
a3=$tmp/act3
# The start of Act 3 played by OD2_AUTOSCRIPT (see docs/PLAYTEST.md): the hero sails east (the quest flags of
# Andariel and Duriel are forced), talks to Alkor, Ormus, Hratli, Asheara and Meshif in Kurast Docks (the real
# town preset), walks through the seamless borders Spider Forest (76), Great Marsh (77), Flayer Jungle (78),
# Lower Kurast (79), Kurast Bazaar (80), Upper Kurast (81), Kurast Causeway (82) and Travincal (83), enters the
# dungeons behind them (Spider Cavern, Flayer Dungeon, Sewers, Temples, Durance of Hate) and opens a town
# portal back to Kurast Docks.
# "KILL giving up for now on <monster>" warnings are part of a fight in the jungle (a monster in the
# undergrowth the hero cannot get close to); errors still fail the scenario
scenario_warnings_ok=1
scenario_env() {
  mkdir -p $a3/s94 $a3/wb94; rm -f $a3/s94/*.d2s(N) $a3/wb94/*.d2s(N)
  if [ -n "${D2_TABLES:-}" ] && make_hero $a3/s94/Hero.d2s; then
    local s="wait:1;say:resetquests;say:completequest 1 6;travel:2;expect:level=40;wait:3;say:completequest 2 6;travel:3;expect:level=75;wait:4;say:capframe $tmp/act3-docks.png"
    local n
    for n in Alkor Ormus Hratli Asheara Meshif; do
      s+=";move:npc=$n;until:NPC menu opened: npc=\"$n\",60;menu:Talk;wait:3"
    done
    # hop <level>: walk through the border or the entrance tile that leads there
    hop() { s+=";walkto:exit=$1;expect:level=$1;wait:3;say:capframe $tmp/act3-$1.png"; }
    hop 76; s+=";kill:near=30,45;loot:30,40"
    hop 85; s+=";kill:near=20,30"; hop 76
    hop 77
    hop 78; s+=";kill:near=30,45"
    hop 86; s+=";kill:near=20,30"; hop 78
    hop 88; s+=";kill:near=20,30"; hop 78
    hop 79
    hop 80
    hop 92; hop 80
    hop 94; hop 80
    hop 81
    hop 96; hop 81
    hop 82
    hop 83
    # the Travincal stairs stay sealed until the Compelling Orb is smashed (d2level.CheckActThreeWarp): Khalim's Will done
    s+=";say:completequest 3 2"; hop 100; hop 83
    s+=";wait:8;say:spawnportal 75;use:Portal;expect:level=75;wait:3;say:capframe $tmp/act3-portal.png"
    echo "export OD2_AUTOGAME=\"$a3/s94/Hero.d2s\" OD2_D2S_WRITEBACK=\"$a3/wb94\" OD2_REALMAPS=1 OD2_AUTOSPEED=3 OD2_AUTOMONSTER_DIFF=0"
    echo "export OD2_AUTOSCRIPT='$s;exit'"
  else
    echo "# no revived sample (D2_TABLES unset or the tool failed)" >&2
    echo "export OD2_AUTOSCRIPT='wait:1;exit'"
  fi
}
scenario_check() {
  grep -E "act town|HERO state|LEVEL CHANGE|POPULATE level|KILL (start|done)|exit tile|AUTOSCRIPT RESULT" $log.txt | cut -c1-230 | tail -50
  grep -q "AUTOSCRIPT RESULT PASS" $log.txt || { echo "FAIL: the Act 3 playthrough did not pass"; fail=1; }
  [ -s $a3/s94/Hero.d2s ] || { echo "SKIP: no revived sample hero in this run"; return; }
  grep -E "AUTOSCRIPT step [0-9]+ FAIL" $log.txt | cut -c1-200
  # the real town preset in its level rectangle of the Act 3 world
  grep -qE "act town: level 75 .* Act3/Docktown/DockTown3\.ds1" $log.txt || { echo "FAIL: Kurast Docks was not built from its town preset"; fail=1; }
  # every level change of the route happened through the real exit (seamless borders, entrance tiles)
  for route in "from=40 to=75 .*via=act:npc" "from=75 to=76 .*via=edge" "from=76 to=85 .*via=warp" "from=85 to=76 .*via=warp" \
               "from=76 to=77 .*via=edge" "from=77 to=78 .*via=edge" "from=78 to=86 .*via=warp" "from=86 to=78 .*via=warp" "from=78 to=88 .*via=warp" "from=88 to=78 .*via=warp" \
               "from=78 to=79 .*via=edge" "from=79 to=80 .*via=edge" "from=80 to=92 .*via=warp" "from=92 to=80 .*via=warp" \
               "from=80 to=94 .*via=warp" "from=94 to=80 .*via=warp" "from=80 to=81 .*via=edge" "from=81 to=96 .*via=warp" \
               "from=96 to=81 .*via=warp" "from=81 to=82 .*via=edge" "from=82 to=83 .*via=edge" "from=83 to=100 .*via=warp" \
               "from=100 to=83 .*via=warp" "from=83 to=75 .*via=portal"; do
    grep -qE "LEVEL CHANGE $route" $log.txt || { echo "FAIL: missing level change $route"; fail=1; }
  done
  # no entrance leads to level 0, and the hero was not sent into a dungeon by a fight next to its entrance
  grep -qE "exit tile style=[0-3] at .*(Spid|Sewer|Temple).* leads to level 0" $log.txt && { echo "FAIL: a dungeon entrance leads nowhere"; fail=1; }
  # the five town NPCs talked
  for npc in Alkor Ormus Hratli Asheara Meshif; do
    grep -qE "NPC menu: Talk with \"$npc\"" $log.txt || { echo "FAIL: the hero did not talk to $npc"; fail=1; }
  done
  # the levels were populated from Levels.txt and the hero fought
  for lvl in 76 77 78 79 80 81 83 92 100; do
    grep -E "POPULATE level $lvl " $log.txt | grep -qE "[1-9][0-9]* monsters" || { echo "FAIL: level $lvl got no monsters"; fail=1; }
  done
  # since pass5 the DS1 monsters of mazes are created by the population plan, so a POPULATE line counts too
  # Spider Cavern (a super unique), the Swampy Pit and the Flayer Dungeon carry their monsters in the DS1 (the natural groups can skip those blocks)
  for lvl in 85 86 88; do
    { grep -A3 "real maze: level $lvl " $log.txt | grep -qE "DS1 monsters: ([1-9][0-9]* direct|.* [1-9][0-9]* super uniques)" \
      || grep -E "POPULATE level $lvl " $log.txt | grep -qE "[1-9][0-9]* monsters"; } || { echo "FAIL: level $lvl got no monsters"; fail=1; }
  done
  grep -q "MONSTER death" $log.txt || { echo "FAIL: nothing was killed"; fail=1; }
  grep -q "DEATH hero=" $log.txt && { echo "FAIL: the level 94 hero died on the way"; fail=1; }
  last=$(grep "D2S EXPORT reparse" $log.txt | tail -1)
  echo "$last" | grep -q " act=3 " || { echo "FAIL: the exported .d2s of a hero in Kurast Docks is not an Act 3 save: $last" | cut -c1-200; fail=1; }
  for s in docks 76 77 78 79 80 81 82 83 85 86 88 92 94 96 100 portal; do [ -s $tmp/act3-$s.png ] || { echo "FAIL: no screenshot act3-$s.png"; fail=1; }; done
  if grep -E "\[ERROR\]|EXIT gave up|EXIT no way found" $log.txt | grep -v "skipping missing"; then echo "FAIL: errors or a walk that gave up in the Act 3 log"; fail=1; fi
  if grep -E "Unknown tile|panic" $log.txt; then echo "FAIL: unknown tiles or panic in the Act 3 log"; fail=1; fi
}
