scenario_name="w44 hunt: new Paladin (OD2_HUNT_CLASS) in Act 4 and 5: Fortress and Harrogath NPCs (menus, trade, gamble), fields"
scenario_turbo=1
scenario_warnings_ok=1
# Play-style hunt, wave 44. Frames are read by the hunter (Read tool) from $tmp, never committed.
h=$tmp/w44a45
scenario_env() {
  mkdir -p $h/new $h/wb; rm -f $h/new/*.d2s(N) $h/wb/*.d2s(N)
  local cls=${OD2_HUNT_CLASS:-Paladin}
  {
    echo '#!/bin/zsh'
    echo "export OD2_PORT=$OD2_PORT OD2_AUTONEWCHAR=$cls OD2_AUTONEWCHAR_NAME=Hunter OD2_D2S_WRITEBACK=$h/new OD2_AUTOEXIT=1"
    echo "$tmp/od2 > $h/newchar.log 2>&1"
  } > $h/newchar.command
  chmod +x $h/newchar.command; launch_game $h/newchar.command; wait_run
  local s="wait:1;say:levelup 40;say:setgold 50000"
  s+=";say:completequest 1 6;travel:2;expect:level=40;wait:2;say:completequest 2 6;travel:3;expect:level=75;wait:2;say:completequest 3 6;travel:4;expect:level=103;wait:3;say:capframe $tmp/a4-town.png"
  local n
  for n in Tyrael Halbu Jamella; do
    s+=";move:npc=$n;until:NPC menu opened: npc=\"$n\",60;say:capframe $tmp/a4-$n.png;press:Escape;wait:1"
  done
  s+=";move:npc=Halbu;until:NPC menu opened: npc=\"Halbu\",60;menu:Trade;wait:2;say:capframe $tmp/a4-trade-Halbu.png;press:Escape;wait:1"
  s+=";move:npc=Jamella;until:NPC menu opened: npc=\"Jamella\",60;menu:Trade;wait:2;say:capframe $tmp/a4-trade-Jamella.png;press:Escape;wait:1"
  s+=";move:npc=Jamella;until:NPC menu opened: npc=\"Jamella\",60;menu:Gamble;wait:2;say:capframe $tmp/a4-gamble-Jamella.png;press:Escape;wait:1"
  s+=";use:Waypoint;wait:2;say:capframe $tmp/a4-wp.png;press:Escape;wait:1"
  s+=";say:completequest 4 2;travel:5;expect:level=109;wait:3;say:capframe $tmp/a5-town.png"
  for n in Larzuk Malah Qual-Kehk Nihlathak; do
    s+=";move:npc=$n;until:NPC menu opened: npc=\"$n\",60;say:capframe $tmp/a5-$n.png;press:Escape;wait:1"
  done
  s+=";move:npc=Larzuk;until:NPC menu opened: npc=\"Larzuk\",60;menu:Trade;wait:2;say:capframe $tmp/a5-trade-Larzuk.png;press:Escape;wait:1"
  s+=";move:npc=Malah;until:NPC menu opened: npc=\"Malah\",60;menu:Trade;wait:2;say:capframe $tmp/a5-trade-Malah.png;press:Escape;wait:1"
  s+=";move:npc=Qual-Kehk;until:NPC menu opened: npc=\"Qual-Kehk\",60;menu:Hire;wait:2;say:capframe $tmp/a5-hire.png;press:Escape;wait:1"
  s+=";use:Waypoint;wait:2;say:capframe $tmp/a5-wp.png;press:Escape;wait:1"
  hop() { s+=";walkto:exit=$1;expect:level=$1;wait:2;say:capframe $tmp/a45-$1.png"; }
  play() { s+=";kill:near=30,60;loot:30,40;say:capframe $tmp/a45-$1-after.png"; }
  hop 110; play 110; hop 111; play 111
  echo "export OD2_AUTOGAME=\"$h/new/Hunter.d2s\" OD2_D2S_WRITEBACK=\"$h/wb\" OD2_REALMAPS=1 OD2_AUTOSPEED=3 OD2_POPULATE_DENSITY=25 OD2_AUTOSCRIPT_HEAL=1"
  echo "export OD2_AUTOSCRIPT='$s;exit'"
}
scenario_check() {
  grep -E "AUTOSCRIPT step [0-9]+ FAIL|AUTOSCRIPT RESULT|NPC menu opened|LEVEL CHANGE" $log.txt | cut -c1-200 | tail -50
  grep -q "AUTOSCRIPT RESULT PASS" $log.txt || { echo "FAIL: script did not pass"; fail=1; }
  # the waypoint panel shows string.tbl names, not the Levels.txt LevelName keys ("Rigid Highlands", "Crystalized Cavern Level 2")
  grep -E "WAYPOINT PANEL act=5" $log.txt | grep -q "Frigid Highlands" || { echo "FAIL: the Act 5 waypoint panel shows LevelName keys instead of the string.tbl names"; fail=1; }
}
scenario_timeout=1500
