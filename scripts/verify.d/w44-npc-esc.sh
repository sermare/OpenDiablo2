scenario_name="w44 NPC Esc: Esc in an NPC menu is its Leave row (no game menu); Esc during NPC speech cuts the voice and subtitle and is swallowed (exe key table 0x721940 / dialog table 0x7264c0); the next Esc opens the game menu"
scenario_turbo=1
scenario_warnings_ok=1
h1=$tmp/w44esc
scenario_env() {
  mkdir -p $h1/new $h1/wb; rm -f $h1/new/*.d2s(N) $h1/wb/*.d2s(N)
  {
    echo '#!/bin/zsh'
    echo "export OD2_PORT=$OD2_PORT OD2_AUTONEWCHAR=Druid OD2_AUTONEWCHAR_NAME=Hunter OD2_D2S_WRITEBACK=$h1/new OD2_AUTOEXIT=1"
    echo "$tmp/od2 > $h1/newchar.log 2>&1"
  } > $h1/newchar.command
  chmod +x $h1/newchar.command; launch_game $h1/newchar.command; wait_run
  local s="wait:1"
  # 1 Esc in the NPC menu = Leave: menu closes, no game menu
  s+=';move:npc=Akara;until:NPC menu opened: npc="Akara",60;press:Escape;wait:1'
  # 2 Esc during the speech: swallowed and cuts the line, no game menu; then Esc opens the game menu, Esc closes it
  s+=';move:npc=Akara;until:NPC menu opened: npc="Akara",60;menu:Talk;wait:1;press:Escape;wait:1;press:Escape;wait:1;press:Escape;wait:1'
  echo "export OD2_AUTOGAME=\"$h1/new/Hunter.d2s\" OD2_D2S_WRITEBACK=\"$h1/wb\" OD2_AUTOSPEED=1"
  echo "export OD2_AUTOSCRIPT='$s;exit'"
}
scenario_check() {
  grep -E "AUTOSCRIPT step [0-9]+ FAIL|AUTOSCRIPT RESULT|NPC menu: Cancel|NPC speech skipped|INPUT panels after Escape" $log.txt | cut -c1-200 | tail -20
  grep -q "AUTOSCRIPT RESULT PASS" $log.txt || { echo "FAIL: script did not pass"; fail=1; }
  grep -q "NPC menu: Cancel" $log.txt || { echo "FAIL: Esc did not choose Leave in the NPC menu"; fail=1; }
  [ "$(grep -c 'NPC speech skipped by key' $log.txt)" = 1 ] || { echo "FAIL: Esc during speech must cut the line exactly once"; fail=1; }
  # menu=true must show exactly once (the Esc after the cut), never on the Esc in the NPC menu or on the Esc that cut the speech
  [ "$(grep 'INPUT panels after Escape' $log.txt | grep -c 'menu=true')" = 1 ] || { echo "FAIL: the game menu must open exactly once (only the Esc with no NPC menu / speech)"; fail=1; }
}
