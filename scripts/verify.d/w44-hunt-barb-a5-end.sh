scenario_name="w44 hunt: new Barbarian (OD2_HUNT_CLASS) in Act 5: waypoint panel names, Arreat Summit Ancients, Worldstone Keep, Throne of Destruction, Worldstone Chamber"
scenario_turbo=1
scenario_warnings_ok=1
# Play-style hunt, wave 44. Frames are read by the hunter (Read tool) from $tmp, never committed.
h=$tmp/w44a5end
scenario_env() {
  mkdir -p $h/new $h/wb; rm -f $h/new/*.d2s(N) $h/wb/*.d2s(N)
  local cls=${OD2_HUNT_CLASS:-Barbarian}
  {
    echo '#!/bin/zsh'
    echo "export OD2_PORT=$OD2_PORT OD2_AUTONEWCHAR=$cls OD2_AUTONEWCHAR_NAME=Hunter OD2_D2S_WRITEBACK=$h/new OD2_AUTOEXIT=1"
    echo "$tmp/od2 > $h/newchar.log 2>&1"
  } > $h/newchar.command
  chmod +x $h/newchar.command; launch_game $h/newchar.command; wait_run
  local s="wait:1;say:levelup 60;say:resetquests;say:completequest 1 6;travel:2;expect:level=40;wait:2;say:completequest 2 6;travel:3;expect:level=75;wait:2;say:completequest 3 6;travel:4;expect:level=103;wait:2;say:completequest 4 2;travel:5;expect:level=109;wait:3"
  s+=";say:completequest 5 1;say:completequest 5 2;say:completequest 5 3;say:completequest 5 4;wait:3"
  # the waypoint panel of Harrogath: names are string.tbl texts (Frigid Highlands, Glacial Trail, ...)
  s+=";use:Waypoint;wait:2;say:capframe $tmp/a5-wp.png;press:Escape;wait:1"
  s+=";say:spawnportal 120;use:Portal;expect:level=120;wait:3;say:restorevitals;say:capframe $tmp/summit.png"
  s+=";say:ritealtar;wait:2;say:capframe $tmp/rite-wake.png;wait:3;say:capframe $tmp/rite-fight.png"
  s+=";say:restorevitals;say:ritekill;wait:3;say:capframe $tmp/rite-done.png;say:ritedoor;wait:2"
  s+=";say:spawnportal 128;use:Portal;expect:level=128;wait:3;say:capframe $tmp/wsk1.png"
  s+=";walkto:exit=129;expect:level=129;wait:3;say:capframe $tmp/wsk2.png;use:Waypoint;wait:2;say:capframe $tmp/wsk2-wp.png;press:Escape;wait:1"
  s+=";walkto:exit=130;expect:level=130;wait:3;say:capframe $tmp/wsk3.png"
  s+=";walkto:exit=131;expect:level=131;wait:3;say:capframe $tmp/throne.png"
  echo "export OD2_AUTOGAME=\"$h/new/Hunter.d2s\" OD2_D2S_WRITEBACK=\"$h/wb\" OD2_REALMAPS=1 OD2_AUTOSPEED=3 OD2_POPULATE_DENSITY=10 OD2_AUTOSCRIPT_HEAL=1"
  echo "export OD2_AUTOSCRIPT='$s;exit'"
}
scenario_check() {
  grep -E "AUTOSCRIPT step [0-9]+ FAIL|AUTOSCRIPT RESULT|WAYPOINT PANEL|RITE |LEVEL CHANGE" $log.txt | cut -c1-260 | tail -40
  grep -q "AUTOSCRIPT RESULT PASS" $log.txt || { echo "FAIL: script did not pass"; fail=1; }
  grep -qE "WAYPOINT PANEL act=5 .*111:Frigid Highlands.*115:Glacial Trail" $log.txt || { echo "FAIL: the Act 5 waypoint panel does not show the string.tbl names"; fail=1; }
}
scenario_timeout=1500
