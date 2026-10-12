scenario_name="w44 save/load fidelity (a new hero of OD2_W44_CLASS plays Normal, Nightmare, Hell: level-ups, stats, skills, hotkeys, gear, belt, waypoints, quests, gold; Save and Exit, reload through the menus every time, and a last reload that changes nothing must save the very same hero)"
scenario_warnings_ok=1
scenario_timeout=600
# One class per run (OD2_W44_CLASS, default Sorceress). Four games in one process, all through the menus:
#   1 Normal:    6 level-ups, stat points on all four stats, 4 points on a skill, hotkeys, a weapon and a magic weapon,
#                belt potions, waypoints, quests, gold; Baal done so Nightmare opens
#   2 Nightmare: the same hero, more of everything, Baal done so Hell opens
#   3 Hell:      same again
#   4 Hell:      nothing is changed: the saved hero must equal the one of game 3
# Everything happens in a private config folder and save folder (OD2_CONFIG_DIR, OD2_D2S_DIR).
w44=$tmp/w44
w44_class=${OD2_W44_CLASS:-Sorceress}
w44_name=Wsave${w44_class[1,4]} # letters only: the game strips digits from names, and select: must find it
typeset -A w44_skill
w44_skill=(Amazon "Magic Arrow" Sorceress "Fire Bolt" Necromancer "Teeth" Paladin "Sacrifice" Barbarian "Bash" Druid "Firestorm" Assassin "Fire Trauma")
w44_play() { # $1 difficulty-specific extras
  local s="wait:2;say:levelup 6;wait:1;panel:character;wait:1"
  s+=";click:left@ui:221,151;wait:1;click:left@ui:221,151;wait:1;click:left@ui:221,213;wait:1;click:left@ui:221,299;wait:1;click:left@ui:221,299;wait:1;click:left@ui:221,361;wait:1;panel:close"
  s+=";panel:skills;wait:1;skill:spend=${w44_skill[$w44_class]};skill:spend=${w44_skill[$w44_class]};skill:spend=${w44_skill[$w44_class]};skill:spend=${w44_skill[$w44_class]};panel:close;wait:1"
  s+=";hotkey:F1=${w44_skill[$w44_class]}@left;skill:right=${w44_skill[$w44_class]}"
  s+=";say:giveitem hax;say:wield hax;say:giveitemq cap 4;say:useitem cap;say:setbelt vbl;say:givebelt hp1;say:givebelt mp1;say:givebelt hp2"
  s+=";say:setwaypoint 3 1;say:setwaypoint 4 1;say:setwaypoint 5 1;say:setwaypoint 6 1"
  s+=";say:completequest 1 1;say:completequest 1 2;say:completequest 1 3;say:setgold $2;wait:2"
  s+=";$1;wait:300" # the flow saves and exits; a script that ends first would end the process
  echo -n "$s"
}
scenario_env() {
  rm -rf $w44; mkdir -p $w44/saves $w44/cfg
  cp "${OD2_VERIFY_CONFIG_DIR:-$HOME/Library/Application Support/OpenDiablo2}/config.json" $w44/cfg/ 2>/dev/null
  echo "unset OD2_AUTOGAME OD2_D2S_WRITEBACK OD2_AUTOMAPSEED"
  echo "export OD2_CONFIG_DIR=\"$w44/cfg\" OD2_D2S_DIR=\"$w44/saves\""
  echo "export OD2_AUTOSCRIPT_1='$(w44_play "say:completequest 1 6;say:completequest 2 6;say:completequest 3 6;say:completequest 5 6" 1111)'"
  echo "export OD2_AUTOSCRIPT_2='$(w44_play "say:completequest 2 1;say:completequest 1 6;say:completequest 2 6;say:completequest 3 6;say:completequest 5 6" 2222)'"
  echo "export OD2_AUTOSCRIPT_3='$(w44_play "say:completequest 3 1" 3333)'"
  echo "export OD2_AUTOSCRIPT_4='wait:300'"
  local f="single,new,create:$w44_class:$w44_name,difficulty:0,scriptdone,wait:2,saveexit"
  f+=",select:$w44_name,play,difficulty:1,scriptdone,wait:2,saveexit"
  f+=",select:$w44_name,play,difficulty:2,scriptdone,wait:2,saveexit"
  f+=",select:$w44_name,play,difficulty:2,wait:8,saveexit,exit"
  echo "export OD2_AUTOFLOW=\"$f\""
}
scenario_check() {
  grep -E "NEWCHAR created|AUTOSCRIPT RESULT|AUTOSCRIPT step [0-9]+ FAIL|D2S EXPORT (path|reparse)|DIFFICULTY|SAVE DAMAGED|not loaded completely" $log.txt | cut -c1-420
  # autosaves and level changes also export: the last export of each game counts (a game starts at its difficulty: step)
  local -a last
  last=()
  local line g=0 cur=""
  while IFS= read -r line; do
    case "$line" in
      *"AUTOFLOW step"*" difficulty:"*) g=$((g+1)); last[$g]="" ;;
      *"D2S EXPORT reparse"*"name=$w44_name "*) [ $g -gt 0 ] && last[$g]="$line" ;;
    esac
  done < $log.txt
  [ ${#last} -eq 4 ] || { echo "FAIL: want 4 saves (one per game), got ${#last}"; fail=1; return; }
  key() { echo "$1" | grep -oE "(^|[ :])$2=[^ ]*" | head -1 | sed 's/^[ :]//'; }
  # game 1 values
  for kv in level=7 gold=1111 difficulty=0; do
    [ "$(key "${last[1]}" ${kv%%=*})" = "$kv" ] || { echo "FAIL: game 1 save: want $kv, got '$(key "${last[1]}" ${kv%%=*})'"; fail=1; }
  done
  [ "$(key "${last[2]}" difficulty)" = "difficulty=1" ] || { echo "FAIL: game 2 saved in difficulty $(key "${last[2]}" difficulty)"; fail=1; }
  [ "$(key "${last[3]}" difficulty)" = "difficulty=2" ] || { echo "FAIL: game 3 saved in difficulty $(key "${last[3]}" difficulty)"; fail=1; }
  for g in 1 2 3; do
    echo "game $g: $(key "${last[$g]}" level) $(key "${last[$g]}" gold) $(key "${last[$g]}" str) $(key "${last[$g]}" dex) $(key "${last[$g]}" vit) $(key "${last[$g]}" ene) $(key "${last[$g]}" spent) $(key "${last[$g]}" unused) $(key "${last[$g]}" hotkeys) $(key "${last[$g]}" belt) $(key "${last[$g]}" alldiff)"
  done
  # the skills and hotkeys of game 1 are still there in every later save
  for k in skills hotkeys; do
    [ "$(key "${last[1]}" $k)" = "$(key "${last[2]}" $k)" ] || echo "NOTE: $k game 1 '$(key "${last[1]}" $k)' vs game 2 '$(key "${last[2]}" $k)'"
  done
  # the Normal part of alldiff (first q/w pair) never changes after game 1, the Nightmare part never after game 2
  local a1 a2 a3
  a1=$(key "${last[1]}" alldiff | sed -E 's/alldiff=([^,]*),.*/\1/')
  a2=$(key "${last[2]}" alldiff | sed -E 's/alldiff=([^,]*),.*/\1/')
  a3=$(key "${last[3]}" alldiff | sed -E 's/alldiff=([^,]*),.*/\1/')
  [ "$a1" = "$a2" ] && [ "$a1" = "$a3" ] || { echo "FAIL: the Normal quests/waypoints changed after game 1: $a1 / $a2 / $a3"; fail=1; }
  a2=$(key "${last[2]}" alldiff | sed -E 's/alldiff=[^,]*,([^,]*),.*/\1/')
  a3=$(key "${last[3]}" alldiff | sed -E 's/alldiff=[^,]*,([^,]*),.*/\1/')
  [ "$a2" = "$a3" ] || { echo "FAIL: the Nightmare quests/waypoints changed after game 2: $a2 / $a3"; fail=1; }
  # the last game changed nothing: the whole line is the same
  [ "${last[3]}" = "${last[4]}" ] || { echo "FAIL: the reload that changed nothing saved a different hero"; echo " game 3: ${last[3]}"; echo " game 4: ${last[4]}"; fail=1; }
  grep -q "AUTOFLOW step [0-9]* exit" $log.txt || { echo "FAIL: the flow did not reach its end"; fail=1; }
  grep -qE "refusing to write an invalid save|SAVE DAMAGED|changed on disk" $log.txt && { echo "FAIL: the save layer refused or found damage"; fail=1; }
  ls -A $w44/saves | grep -q '\.tmp' && { echo "FAIL: temp file left in the save folder"; fail=1; }
  return 0
}
