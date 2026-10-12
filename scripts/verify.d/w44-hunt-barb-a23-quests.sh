scenario_name="w44 hunt: new Barbarian in the Act 2 and 3 quest areas the w41 hunt skipped (Maggot Lair, Claw Viper Temple, Arcane Sanctuary, Tal Rasha's tomb, Travincal, Durance of Hate), waypoint travel through portals"
scenario_turbo=1
scenario_warnings_ok=1
# Play-style hunt, wave 44. Portals are spawned next to the hero (spawnportal <level>) and used with use:Portal; frames are read by the hunter.
h=$tmp/w44a23
scenario_env() {
  mkdir -p $h/new $h/wb; rm -f $h/new/*.d2s(N) $h/wb/*.d2s(N)
  local cls=${OD2_HUNT_CLASS:-Barbarian}
  {
    echo '#!/bin/zsh'
    echo "export OD2_PORT=$OD2_PORT OD2_AUTONEWCHAR=$cls OD2_AUTONEWCHAR_NAME=Hunter OD2_D2S_WRITEBACK=$h/new OD2_AUTOEXIT=1"
    echo "$tmp/od2 > $h/newchar.log 2>&1"
  } > $h/newchar.command
  chmod +x $h/newchar.command; launch_game $h/newchar.command; wait_run
  local s="wait:1;say:levelup 45;say:setgold 20000;say:resetquests;say:completequest 1 6;travel:2;expect:level=40;wait:7;say:capframe $tmp/a2-town.png"
  local lv
  # Act 2 quest areas through portals: Maggot Lair 1, Claw Viper Temple 1, Arcane Sanctuary, Tal Rasha's Tomb, the Chamber
  for lv in 62 58 74 66 73; do
    s+=";say:spawnportal $lv;use:Portal;expect:level=$lv;wait:3;say:restorevitals;say:capframe $tmp/a2-$lv.png;kill:near=20,25;wait:1;say:capframe $tmp/a2-$lv-b.png"
    s+=";say:spawnportal 40;wait:1;use:Portal;expect:level=40;wait:7"
  done
  # Act 3: Travincal, Durance of Hate 1-3, Flayer dungeon, Spider cavern
  s+=";say:completequest 2 6;travel:3;expect:level=75;wait:7;say:capframe $tmp/a3-town.png"
  for lv in 83 100 101 102 88 85; do
    s+=";say:spawnportal $lv;use:Portal;expect:level=$lv;wait:3;say:restorevitals;say:capframe $tmp/a3-$lv.png;kill:near=20,25;wait:1;say:capframe $tmp/a3-$lv-b.png"
    s+=";say:spawnportal 75;wait:1;use:Portal;expect:level=75;wait:7"
  done
  echo "export OD2_AUTOGAME=\"$h/new/Hunter.d2s\" OD2_D2S_WRITEBACK=\"$h/wb\" OD2_REALMAPS=1 OD2_AUTOSPEED=3 OD2_POPULATE_DENSITY=15 OD2_AUTOSCRIPT_HEAL=1"
  echo "export OD2_AUTOSCRIPT='$s;exit'"
}
scenario_check() {
  grep -E "AUTOSCRIPT step [0-9]+ FAIL|AUTOSCRIPT RESULT|LEVEL CHANGE" $log.txt | cut -c1-200 | tail -40
  grep -q "AUTOSCRIPT RESULT PASS" $log.txt || { echo "FAIL: script did not pass"; fail=1; }
}
scenario_timeout=1800
