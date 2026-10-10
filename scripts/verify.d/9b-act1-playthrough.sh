scenario_name="Act 1 playthrough (fresh Sorceress: Akara, gate to Blood Moor, fights, loot, Den of Evil, reward, exported .d2s)"
a1=$tmp/act1
# The first hour of Act 1 played by OD2_AUTOSCRIPT (see docs/PLAYTEST.md): talk to Akara (intro, Den of
# Evil quest, topic), walk out of the town through its border into Blood Moor (real level change), fight
# on the way, enter the Den of Evil cave through its entrance tile, clear it, come back and collect
# the reward from Akara, then exit (which saves the hero to a .d2s). 9c reloads that .d2s.
scenario_env() {
  mkdir -p $a1/new $a1/wb; rm -f $a1/new/*.d2s(N) $a1/wb/*.d2s(N)
  # a brand new Sorceress through the hero creation path (a 335 byte .d2s without items)
  {
    echo '#!/bin/zsh'
    echo "export OD2_PORT=$OD2_PORT OD2_AUTONEWCHAR=Sorceress OD2_AUTONEWCHAR_NAME=Playtest OD2_D2S_WRITEBACK=$a1/new OD2_AUTOEXIT=1"
    echo "$tmp/od2 > $a1/newchar.log 2>&1"
  } > $a1/newchar.command
  chmod +x $a1/newchar.command; launch_game $a1/newchar.command; wait_run
  echo "export OD2_POPULATE_DENSITY=25 OD2_AUTOGAME=\"$a1/new/Playtest.d2s\" OD2_D2S_WRITEBACK=\"$a1/wb\" OD2_REALMAPS=1 OD2_AUTOSPEED=3"
  echo "export OD2_AUTOSCRIPT='wait:1;say:capframe $tmp/act1-town.png;move:npc=Akara;until:NPC menu opened: npc=\"Akara\",60;menu:Talk;until:QUEST intro flag set,10;move:npc=Akara;until:NPC menu opened: npc=\"Akara\",30;menu:Talk;until:A1Q1 slot=1 bits 0x0000->0x0004,10;move:npc=Akara;until:NPC menu opened: npc=\"Akara\",30;menu:Talk;until:QUEST TOPICS,10;say:capframe $tmp/act1-topics.png;menu:Den of Evil;until:msg=65 mode=2,10;walkto:exit=2;expect:level=2;say:capframe $tmp/act1-bloodmoor.png;kill:near=30,45;loot:30,40;walkto:exit=8;expect:level=8;kill:all,200;until:Den of Evil cleared,10;say:capframe $tmp/act1-den.png;loot:40,40;walkto:exit=2;expect:level=2;walkto:exit=1;expect:level=1;move:npc=Akara;until:NPC menu opened: npc=\"Akara\",60;menu:Talk;until:QUEST EFFECT skill-point,15;say:capframe $tmp/act1-reward.png;exit'"
}
scenario_check() {
  grep -E "NEWCHAR parse|LEVEL CHANGE|POPULATE|KILL (start|done)|LOOT (start|done)|HERO (LEVEL UP|state)|QUEST EFFECT|AUTOSCRIPT RESULT|D2S EXPORT reparse" $log.txt | cut -c1-260 | tail -40
  grep -q "AUTOSCRIPT RESULT PASS" $log.txt || { echo "FAIL: the Act 1 playthrough did not pass"; fail=1; }
  grep -E "AUTOSCRIPT step [0-9]+ FAIL" $log.txt | cut -c1-200
  grep -q "NEWCHAR parse: .*class=Sorceress level=1" $a1/newchar.log 2>/dev/null || { echo "FAIL: no new Sorceress was made"; fail=1; }
  grep -q "HERO state at start: level=1 exp=0" $log.txt || { echo "FAIL: the playthrough did not start with a level 1 hero"; fail=1; }
  grep -q "containers loaded: .*belt_items=4" $log.txt || { echo "FAIL: the new hero has no starting potions"; fail=1; }
  # every level change of the route happened through the real exit
  for route in "from=1 to=2 .*via=edge" "from=2 to=8 .*via=warp" "from=8 to=2 .*via=warp" "from=2 to=1 .*via=edge"; do
    grep -qE "LEVEL CHANGE $route" $log.txt || { echo "FAIL: missing level change $route"; fail=1; }
  done
  # the quest line spoke and finished
  grep -qE "QUEST SPEECH npc=\"Akara\" .* msg=64 " $log.txt || { echo "FAIL: Akara did not give the Den of Evil quest"; fail=1; }
  grep -qE "QUEST SPEECH npc=\"Akara\" .* msg=65 mode=2" $log.txt || { echo "FAIL: the Den of Evil topic was not spoken"; fail=1; }
  grep -q "Den of Evil cleared" $log.txt || { echo "FAIL: the Den of Evil was not cleared"; fail=1; }
  grep -q "QUEST EFFECT skill-point +1" $log.txt || { echo "FAIL: no skill point from Akara"; fail=1; }
  # both levels were populated from Levels.txt and the hero fought
  for lvl in 2 8; do
    grep -E "POPULATE level $lvl " $log.txt | grep -qE "[1-9][0-9]* monsters" || { echo "FAIL: level $lvl got no monsters"; fail=1; }
  done
  grep -q "MONSTER death" $log.txt || { echo "FAIL: nothing was killed"; fail=1; }
  # the hero left something on screen
  for s in town topics bloodmoor den reward; do [ -s $tmp/act1-$s.png ] || { echo "FAIL: no screenshot act1-$s.png"; fail=1; }; done
  # state saved to the real .d2s: Den of Evil done, the skill point unspent, the same hero as at the end of the game
  last=$(grep "D2S EXPORT reparse" $log.txt | tail -1)
  echo "$last" > $a1/play.reparse
  grep "HERO state at exit" $log.txt | tail -1 > $a1/play.state
  echo "$last" | grep -oE "act1quests=\[[^]]*\]"
  echo "$last" | grep -qE "act1quests=\[([^]]* )?1:0x[0-9a-f]{3}[13579bdf]" || { echo "FAIL: the Den of Evil is not saved as done in the .d2s"; fail=1; }
  echo "$last" | grep -qE "unused=[1-9]" || { echo "FAIL: the skill point of the quest reward is not in the .d2s"; fail=1; }
  echo "$last" | grep -q "checksum=ok" || { echo "FAIL: exported .d2s checksum"; fail=1; }
  [ -s $a1/wb/Playtest.d2s ] || { echo "FAIL: no exported Playtest.d2s"; fail=1; }
  if grep -E "Unknown tile|panic" $log.txt; then echo "FAIL: unknown tiles or panic in the playthrough log"; fail=1; fi
}
