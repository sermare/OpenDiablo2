scenario_name="mercenary (merc spawns, fights, dies, is revived, follows; hirelings cast their skills.txt skills through the skill engine; real-map level change keeps its life and a dead merc stays behind until revived; header survives the export)"
wb=$tmp/writeback-merc
scenario_env() {
  mkdir -p $wb   # keeps the exported file out of the Saves folder
  # phases: spawn/fight/die/revive/follow with the save's merc; then OD2_AUTOMERC_TYPES hires a rogue archer (Fire Arrow, Inner Sight),
  # a desert mercenary (Prayer aura), an eastern sorcerer (Frozen Armor, Glacial Spike, Ice Blast) and a barbarian (Bash, Stun) and casts each
  # of their skills once; then OD2_AUTOMERC_LEVELS takes the hero by waypoint to the real DRLG map of Jail Level 1 (OD2_REALMAPS) with the
  # merc hurt, kills it there, goes back to the encampment by portal (the dead merc stays behind) and revives it
  # W37_SHOT=<file.png> (optional, by hand) saves a frame of the interface while the portrait shows the hurt merc
  if [ -n "${W37_SHOT:-}" ]; then echo "export OD2_AUTOSHOT=$W37_SHOT OD2_AUTOSHOT_DELAY=${W37_SHOT_DELAY:-14}"; fi
  echo "export OD2_AUTOMERC_UI=1 OD2_AUTOMERC=skeleton1,4 OD2_AUTOMERC_KILL=1 OD2_AUTOMERC_TYPES=0,6,16,24 OD2_AUTOMERC_MERCLEVEL=85 OD2_AUTOMERC_LEVELS=1 OD2_REALMAPS=1 OD2_D2S_WRITEBACK=\"$wb\""
}
scenario_check() {
  grep -E "AUTOMERC (summary|types|levels)|MERC (spawn|hire|death|revive|leaves|arrive|stays)|MERC leader follow|D2S EXPORT reparse" $log.txt | cut -c1-300
  for pat in "MERC spawn " "MERC attack .*hit=true" "MERC death " "MERC revive " "AUTOMERC follow dist=" "MERC leader follow kind=[01] " "AUTOMERC summary" \
    "D2S EXPORT reparse: .*merc=type"; do
    grep -qE "$pat" $log.txt || { echo "FAIL: no '$pat' in the merc log"; fail=1; }
  done

  # w42: the exe-ported Hireable AI follows through the Director's host hooks (MercLeaderOps): the hero walks 60 subtiles
  # away, the merc runs after it (kind 1 ring) and is back within the inner ring before the phase ends
  local fdist
  fdist=$(grep "AUTOMERC follow dist=" $log.txt | tail -1 | sed -E 's/.*follow dist=([0-9]+) .*/\1/')
  if [ -z "$fdist" ] || [ "$fdist" -gt 30 ]; then echo "FAIL: the merc did not catch up with the hero (last follow dist=$fdist)"; fail=1; fi

  # w37 merc interface: the hire lists are in the form of the exe's dialog, the portrait takes a potion and heals the merc,
  # gear dropped on it is worn, a too heavy item is refused, the right button opens the merc screen
  for pat in "AUTOMERC ui: hire seller=150 ok=true title=\"Your Gold: [0-9]+ +Hire which Mercenary\\?\" rows=[0-9]+ first=\".* - Lvl: +[0-9]+  Life: +[0-9]+  Def: +[0-9]+  Cost: +[0-9]+" \
    "AUTOMERC ui: hire seller=198 ok=true" "AUTOMERC ui: hire seller=252 ok=true" "AUTOMERC ui: hire seller=515 ok=true" \
    "MERC ui portrait shown tooltip=" "MERC ui potion drop handled=true cursor_empty=true" "MERC potion name=.*(instant=[1-9]|over_time=[1-9])" \
    "AUTOMERC ui: hire click had_merc=true confirm_open=true rows=2 title=\"This Mercenary will replace your current one\\.\"" \
    "AUTOMERC ui: hire click answered No: menu_open=false" \
    "MERC hire message code=12 text=\"You do not have enough gold for that\\.\"" "AUTOMERC ui: hire click broke: message_open=true rows=1" \
    "AUTOMERC ui: hire waiting open=true pending=true rows=0 text=\"Waiting for confirmation of transaction\\.\\.\\.\"" \
    "MERC txn reply kind=0 code=12 " "MERC txn reply kind=0 code=5 " "MERC txn reply kind=1 code=5 " \
    "AUTOMERC ui: hire wheel entries=([7-9]|10) first=1" \
    "AUTOMERC ui: hire title with stash=\"Your Gold: 1000000 +Hire which Mercenary\\?\"" \
    "AUTOMERC ui: hire paid from stash purse=0 stash=1000000->[0-9]+ merc_screen=true" \
    "MERC first hire: mercenary screen opened" "AUTOMERC revive txn: cost=[0-9]+ " \
    "MERC ui right click merc_screen_open=true" "MERC give decision=ok item=hp5 consumed=true" "MERC ui gear gear=cap@1" "MERC ui shield item=tow on_cursor=true"; do
    grep -qE "$pat" $log.txt || { echo "FAIL: no '$pat' in the merc log"; fail=1; }
  done
  grep -E "AUTOMERC ui:" $log.txt | cut -c1-260

  # skills.txt effects through the d2skills engine instead of own damage + a timed flag
  for pat in "AUTOMERC types: probe Inner Sight=ok Fire Arrow=ok" "AUTOMERC types: probe Jab=ok Prayer=ok" \
    "AUTOMERC types: probe Glacial Spike=ok Frozen Armor=ok Ice Blast=ok" "AUTOMERC types: probe Bash=ok Stun=ok" \
    "CAST start merc=.*skill=\"Fire Arrow\".* ok=true" "MISSILE create name=firearrow" "SKILL melee skill=\"Bash\"" \
    "STATE aura skill=\"Prayer\"" "STATE apply skill=\"Frozen Armor\" unit=merc" "STATE apply skill=\"Inner Sight\"" \
    "STATE aura end skill=\"Prayer\"" "MERC skill .*skill=\"Jab\".*engine=true"; do
    grep -qE "$pat" $log.txt || { echo "FAIL: no '$pat' in the merc log"; fail=1; }
  done

  # level change: the merc travels with the life it had, arrives next to the hero in the real map
  local left arrived
  left=$(grep -m1 "MERC leaves its level hp=" $log.txt | sed -E 's/.*hp=([0-9]+)\/.*/\1/')
  arrived=$(grep -m1 "MERC arrive level=29 " $log.txt | sed -E 's/.*carried hp=([0-9]+)\).*/\1/')
  if [ -z "$left" ] || [ -z "$arrived" ] || [ "$left" != "$arrived" ]; then
    echo "FAIL: the merc did not arrive with the life it left with (left=$left arrived=$arrived)"; fail=1
  fi
  grep -qE "MERC arrive level=29 hp=$left/" $log.txt || { echo "FAIL: the merc was respawned with another life than $left"; fail=1; }
  for pat in "AUTOMERC levels: arrived level=29 .*alive=true" "real maze: level 29 " "AUTOMERC levels: fight over level=29" \
    "MERC stays behind dead" "AUTOMERC levels: the dead merc stayed behind" "MERC revive: a new unit for the merc that stayed behind"; do
    grep -qE "$pat" $log.txt || { echo "FAIL: no '$pat' in the merc log"; fail=1; }
  done
  local hp
  hp=$(grep -m1 "AUTOMERC levels: revived level=1 " $log.txt | sed -E 's/.*merc hp=([0-9]+)\/([0-9]+) alive=true.*/\1 \2/')
  [ -n "$hp" ] && [ "${hp% *}" = "${hp#* }" ] || { echo "FAIL: the revived merc is not at full life (hp: $hp)"; fail=1; }
  if grep -E "a dead merc followed the hero|the merc did not arrive" $log.txt; then echo "FAIL: merc level-change error"; fail=1; fi
}
