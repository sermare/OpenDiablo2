scenario_name="Act 2 playthrough (level 94 sample hero: Lut Gholein NPCs, Rocky Waste, Dry Hills, Halls of the Dead and back)"
a2=$tmp/act2
# The start of Act 2 played by OD2_AUTOSCRIPT (see docs/PLAYTEST.md): the hero travels east with Warriv
# (the quest flag of Andariel is forced), talks to Warriv, Fara, Atma, Drognan and Greiz in the town of the
# Act 2 world layout (LutW or LutN, the preset the world search chose), walks out of the gate into the
# desert levels (the borders of 40-42 are real level changes), fights and loots, enters the Halls of the Dead
# through the tomb entrance of Dry Hills, takes the stairs down and up again and walks back to town.
# The sample character is a dead hardcore Sorceress; scripts/d2s-revive.go makes a living copy (needs D2_TABLES).
# "KILL giving up for now on <monster>" warnings are part of a fight in the desert (a monster behind a dune the
# hero cannot get close to); errors still fail the scenario
scenario_warnings_ok=1
scenario_env() {
  mkdir -p $a2/s94 $a2/wb94; rm -f $a2/s94/*.d2s(N) $a2/wb94/*.d2s(N)
  if [ -n "${D2_TABLES:-}" ] && go run scripts/d2s-revive.go "$D2S_SAMPLE_BODY" $a2/s94/Hero.d2s "$D2_TABLES" >/dev/null 2>&1; then
    local s="wait:1;say:resetquests;say:completequest 1 6;travel:2;expect:level=40;wait:4;say:capframe $tmp/act2-town.png"
    local n
    for n in Warriv Fara Atma Drognan Greiz; do
      s+=";move:npc=$n;until:NPC menu opened: npc=\"$n\",60;menu:Talk;wait:3"
    done
    s+=";walkto:exit=41;expect:level=41;wait:3;say:capframe $tmp/act2-rocky.png;kill:near=30,45;loot:30,40"
    s+=";walkto:exit=42;expect:level=42;wait:3;say:capframe $tmp/act2-dryhills.png;kill:near=30,45;loot:30,40"
    s+=";walkto:exit=56;expect:level=56;wait:4;say:capframe $tmp/act2-halls.png;kill:near=20,30"
    s+=";walkto:exit=57;expect:level=57;kill:near=20,30;walkto:exit=56;expect:level=56"
    s+=";walkto:exit=42;expect:level=42;walkto:exit=41;expect:level=41;walkto:exit=40;expect:level=40;exit"
    echo "export OD2_AUTOGAME=\"$a2/s94/Hero.d2s\" OD2_D2S_WRITEBACK=\"$a2/wb94\" OD2_REALMAPS=1 OD2_AUTOSPEED=3 OD2_AUTOMONSTER_DIFF=0"
    echo "export OD2_AUTOSCRIPT='$s'"
  else
    echo "# no revived sample (D2_TABLES unset or the tool failed)" >&2
    echo "export OD2_AUTOSCRIPT='wait:1;exit'"
  fi
}
scenario_check() {
  grep -E "act town|HERO state|LEVEL CHANGE|POPULATE level|KILL (start|done)|LOOT (start|done)|exit tile|AUTOSCRIPT RESULT" $log.txt | cut -c1-230 | tail -40
  grep -q "AUTOSCRIPT RESULT PASS" $log.txt || { echo "FAIL: the Act 2 playthrough did not pass"; fail=1; }
  [ -s $a2/s94/Hero.d2s ] || { echo "SKIP: no revived sample hero in this run"; return; }
  grep -E "AUTOSCRIPT step [0-9]+ FAIL" $log.txt | cut -c1-200
  # the real town preset of the world layout, in its level rectangle
  grep -qE "act town: level 40 .* Act2/Town/Lut[WN]\.ds1 map 5[0-9]x5[0-9]" $log.txt || { echo "FAIL: Lut Gholein was not built from a town preset"; fail=1; }
  # every level change of the route happened through the real exit (borders of the desert, tomb stairs)
  for route in "from=1 to=40 .*via=act:npc" "from=40 to=41 .*via=edge" "from=41 to=42 .*via=edge" "from=42 to=56 .*via=warp" \
               "from=56 to=57 .*via=warp" "from=57 to=56 .*via=warp" "from=56 to=42 .*via=warp" "from=42 to=41 .*via=edge" "from=41 to=40 .*via=edge"; do
    grep -qE "LEVEL CHANGE $route" $log.txt || { echo "FAIL: missing level change $route"; fail=1; }
  done
  # the hero did not fall into the tomb of Rocky Waste while chasing a monster next to its entrance
  grep -qE "LEVEL CHANGE from=41 to=55" $log.txt && { echo "FAIL: the hero chased a monster into the Stony Tomb"; fail=1; }
  # the tomb entrance presets lead somewhere (they did to level 0)
  grep -qE "exit tile style=[0-9]+ at .*TombEnt.* leads to level 0" $log.txt && { echo "FAIL: a tomb entrance leads nowhere"; fail=1; }
  grep -qE "TombEnt2.ds1 leads to level 56" $log.txt || { echo "FAIL: the Dry Hills entrance does not lead to the Halls of the Dead"; fail=1; }
  # the five town NPCs talked
  for npc in Warriv Fara Atma Drognan Greiz; do
    grep -qE "NPC menu: Talk with \"$npc\"" $log.txt || { echo "FAIL: the hero did not talk to $npc"; fail=1; }
  done
  grep -qE "QUEST SPEECH npc=\"Atma\" .* msg=304 " $log.txt || { echo "FAIL: Atma did not speak (Radament quest)"; fail=1; }
  # the levels were populated from Levels.txt (Rocky Waste, Dry Hills, Halls of the Dead 1 and 2) and the hero fought
  for lvl in 41 42 56 57; do
    grep -E "POPULATE level $lvl " $log.txt | grep -qE "[1-9][0-9]* monsters" || { echo "FAIL: level $lvl got no monsters"; fail=1; }
  done
  grep -q "MONSTER death" $log.txt || { echo "FAIL: nothing was killed"; fail=1; }
  grep -q "DEATH hero=" $log.txt && { echo "FAIL: the level 94 hero died on the way"; fail=1; }
  # the hero ends in Lut Gholein: the exported .d2s says Act 2 (it said Act 1: the client's player never learns the act)
  last=$(grep "D2S EXPORT reparse" $log.txt | tail -1)
  echo "$last" | grep -q " act=2 " || { echo "FAIL: the exported .d2s of a hero in Lut Gholein is not an Act 2 save: $last" | cut -c1-200; fail=1; }
  for s in town rocky dryhills halls; do [ -s $tmp/act2-$s.png ] || { echo "FAIL: no screenshot act2-$s.png"; fail=1; }; done
  if grep -E "\[ERROR\]|EXIT gave up|EXIT no way found" $log.txt | grep -v "skipping missing"; then echo "FAIL: errors or a walk that gave up in the Act 2 log"; fail=1; fi
  if grep -E "Unknown tile|panic" $log.txt; then echo "FAIL: unknown tiles or panic in the Act 2 log"; fail=1; fi
}
