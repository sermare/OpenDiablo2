scenario_name="w44 hunt: new Barbarian (OD2_HUNT_CLASS) shops in Act 1 like a human: gamble buy click, trade buy click, repair click, scrolls/tomes, identify, stash, portal to town and back"
scenario_turbo=1
scenario_warnings_ok=1
# Play-style hunt, wave 44. Clicks go through the input handlers (click:left@ui:x,y are pixels of the 800x600 interface column).
h=$tmp/w44shop
scenario_env() {
  mkdir -p $h/new $h/wb; rm -f $h/new/*.d2s(N) $h/wb/*.d2s(N)
  local cls=${OD2_HUNT_CLASS:-Barbarian}
  {
    echo '#!/bin/zsh'
    echo "export OD2_PORT=$OD2_PORT OD2_AUTONEWCHAR=$cls OD2_AUTONEWCHAR_NAME=Hunter OD2_D2S_WRITEBACK=$h/new OD2_AUTOEXIT=1"
    echo "$tmp/od2 > $h/newchar.log 2>&1"
  } > $h/newchar.command
  chmod +x $h/newchar.command; launch_game $h/newchar.command; wait_run
  local s="wait:1;say:levelup 20;say:setgold 30000"
  # gamble: open, buy the first item with a click, look at the cursor and the inventory
  s+=";move:npc=Gheed;until:NPC menu opened: npc=\"Gheed\",60;menu:Gamble;wait:2;say:capframe $tmp/gamble0.png"
  s+=";click:left@ui:124,150;wait:1;say:capframe $tmp/gamble-picked.png;click:left@ui:560,360;wait:1;say:capframe $tmp/gamble-placed.png;press:Escape;wait:1"
  # trade: buy a scroll from Akara with a click, drop it in the inventory
  s+=";move:npc=Akara;until:NPC menu opened: npc=\"Akara\",60;menu:Trade;wait:2;say:capframe $tmp/trade0.png;click:left@ui:124,150;wait:1;click:left@ui:560,400;wait:1;say:capframe $tmp/trade-bought.png;press:Escape;wait:1"
  # repair: Charsi with a worn weapon
  s+=";move:npc=Charsi;until:NPC menu opened: npc=\"Charsi\",60;menu:Trade;wait:2;click:left@ui:262,458;wait:1;say:capframe $tmp/repair.png;press:Escape;wait:1"
  # tome and scrolls: give, open inventory, read
  s+=";say:giveitem tbk;say:giveitem ibk;say:giveitem isc;say:giveitem tsc;say:giveitemq rin -4;panel:inventory;wait:1;say:capframe $tmp/inv-scrolls.png;panel:close"
  s+=";say:townportal paid;wait:3;say:capframe $tmp/tp.png;say:portals"
  s+=";click:left@object:267;wait:6;say:capframe $tmp/stash-use.png;panel:close"
  s+=";say:useitem isc;wait:1;say:capframe $tmp/idcursor.png;panel:close"
  s+=";walkto:exit=2;expect:level=2;wait:2;say:townportal paid;wait:3;say:capframe $tmp/tp2.png;wait:3;say:useportal;wait:4;say:capframe $tmp/tp-town.png;wait:6;say:useportal;wait:4;say:capframe $tmp/tp-back.png"
  echo "export OD2_AUTOGAME=\"$h/new/Hunter.d2s\" OD2_D2S_WRITEBACK=\"$h/wb\" OD2_REALMAPS=1 OD2_AUTOSPEED=3"
  echo "export OD2_AUTOSCRIPT='$s;exit'"
}
scenario_check() {
  grep -E "AUTOSCRIPT step [0-9]+ FAIL|AUTOSCRIPT RESULT|TRADE|GAMBLE|REPAIR|PORTAL|USEITEM|LEVEL CHANGE" $log.txt | cut -c1-200 | tail -50
  grep -q "AUTOSCRIPT RESULT PASS" $log.txt || { echo "FAIL: script did not pass"; fail=1; }
}
scenario_timeout=900
