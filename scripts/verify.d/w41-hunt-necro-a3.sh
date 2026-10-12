scenario_name="w41 hunt: Necromancer in Act 3: Kurast Docks NPCs (menus, trade, gamble, hire), jungle, waypoint, dungeon"
scenario_turbo=1
scenario_warnings_ok=1
# Play-style hunt, wave 41. Real maps are the default; the Act 1 and 2 travel quests are forced.
h5=$tmp/w41a3
scenario_env() {
  mkdir -p $h5/new $h5/wb; rm -f $h5/new/*.d2s(N) $h5/wb/*.d2s(N)
  local cls=${OD2_HUNT_CLASS:-Necromancer} skill=${OD2_HUNT_SKILL:-Teeth}
  {
    echo '#!/bin/zsh'
    echo "export OD2_PORT=$OD2_PORT OD2_AUTONEWCHAR=$cls OD2_AUTONEWCHAR_NAME=Hunter OD2_D2S_WRITEBACK=$h5/new OD2_AUTOEXIT=1"
    echo "$tmp/od2 > $h5/newchar.log 2>&1"
  } > $h5/newchar.command
  chmod +x $h5/newchar.command; launch_game $h5/newchar.command; wait_run
  local s="wait:1;say:levelup 30"
  repeat 12 s+=";skill:spend=$skill"
  s+=";skill:left=$skill;say:completequest 1 6;travel:2;expect:level=40;wait:3;say:completequest 2 6;travel:3;expect:level=75;wait:4;say:capframe $tmp/a3-town.png"
  local n
  for n in Alkor Ormus Hratli Asheara Meshif; do
    s+=";move:npc=$n;until:NPC menu opened: npc=\"$n\",60;say:capframe $tmp/a3-$n.png;press:Escape;wait:1"
  done
  s+=";say:setgold 20000"
  s+=";move:npc=Alkor;until:NPC menu opened: npc=\"Alkor\",60;menu:Trade;wait:2;say:capframe $tmp/a3-trade-Alkor.png;press:Escape;wait:1"
  s+=";move:npc=Ormus;until:NPC menu opened: npc=\"Ormus\",60;menu:Trade;wait:2;say:capframe $tmp/a3-trade-Ormus.png;press:Escape;wait:1"
  s+=";move:npc=Hratli;until:NPC menu opened: npc=\"Hratli\",60;menu:Trade;wait:2;say:capframe $tmp/a3-trade-Hratli.png;press:Escape;wait:1"
  s+=";move:npc=Asheara;until:NPC menu opened: npc=\"Asheara\",60;menu:Hire;wait:2;say:capframe $tmp/a3-hire-Asheara.png;press:Escape;wait:1"
  s+=";use:Waypoint;wait:2;say:capframe $tmp/a3-wp-town.png;press:Escape;wait:1"
  hop() { s+=";walkto:exit=$1;expect:level=$1;wait:2;say:capframe $tmp/a3-$1.png"; }
  play() { s+=";kill:near=30,60;loot:30,40;say:capframe $tmp/a3-$1-after.png"; }
  hop 76; play 76
  s+=";say:spawnchest 5 2 118;wait:1;click:left@object:5;wait:6;click:left@object:2;wait:6;click:left@object:118;wait:6;say:capframe $tmp/a3-objs.png;loot:10,20"
  hop 85; play 85; hop 76
  hop 77; play 77
  echo "export OD2_AUTOGAME=\"$h5/new/Hunter.d2s\" OD2_D2S_WRITEBACK=\"$h5/wb\" OD2_REALMAPS=1 OD2_AUTOSPEED=3 OD2_POPULATE_DENSITY=25 OD2_AUTOSCRIPT_HEAL=1 OD2_AUTOSCRIPT_SURVIVE=1"
  echo "export OD2_AUTOSCRIPT='$s;exit'"
}
scenario_check() {
  grep -E "AUTOSCRIPT step [0-9]+ FAIL|AUTOSCRIPT RESULT|LEVEL objects|LEVEL CHANGE" $log.txt | cut -c1-200 | tail -30
  grep -q "AUTOSCRIPT RESULT PASS" $log.txt || { echo "FAIL: script did not pass"; fail=1; }
}
scenario_timeout=1200
