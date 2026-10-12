# Shared by the w44-newchar-<class> scenarios: a hero made in the menu is played, saved and played again; the
# character panel, skill panel, inventory panel and skill bar are logged in both games and compared with the class'
# CharStats.txt row (and the exe's new-character code: PLAYER_InitBaseStatsFromClass 0x56e5d0, PLAYER_GiveStartingItems
# 0x532960, SERVER_CreateFreshPlayer 0x52ff20).
#
#   newchar_env <Class> <Name>
#   newchar_check <life> <mana> <stamina> <str> <dex> <vit> <ene> <worn regexp> <inventory items> <belt items> <left skill> [ammo] [right skill]
scenario_warnings_ok=1
scenario_timeout=300
newchar_env() {
  local class=$1 name=$2
  sd=$tmp/newchar
  rm -rf $sd; mkdir -p $sd/saves $sd/cfg
  cp "${OD2_VERIFY_CONFIG_DIR:-$HOME/Library/Application Support/OpenDiablo2}/config.json" $sd/cfg/ 2>/dev/null
  echo "unset OD2_AUTOGAME OD2_D2S_WRITEBACK OD2_AUTOMAPSEED"
  echo "export OD2_CONFIG_DIR=\"$sd/cfg\" OD2_D2S_DIR=\"$sd/saves\""
  echo "export OD2_AUTOSCRIPT='wait:3;panel:character;wait:1;panel:close;panel:skills;wait:1;panel:close;panel:inventory;wait:1;panel:close;wait:60'"
  local f="single,new,create:$class:$name,difficulty:0,wait:45,saveexit"
  f="$f,select:$name,play,wait:45,saveexit,exit"
  echo "export OD2_AUTOFLOW=\"$f\""
}
newchar_check() {
  local life=$1 mana=$2 stam=$3 str=$4 dex=$5 vit=$6 ene=$7 worn=$8 invn=$9 belt=${10} left=${11} ammo=${12:-0} right=${13:-Attack}
  grep -E "NEWCHAR created|PANEL (character|skills|inventory)|containers loaded|SKILLBAR state" $log.txt | cut -c1-330
  local n
  n=$(grep -c "PANEL character: level=1 name=" $log.txt)
  [ "$n" -ge 2 ] || { echo "FAIL: want the character panel of both games, got $n"; fail=1; return; }
  # both games: level 1, the class attributes, life/mana/stamina of PLAYER_InitBaseStatsFromClass
  n=$(grep "PANEL character: level=1 name=" $log.txt | grep -cE " str=$str dex=$dex vit=$vit ene=$ene hp=$life/$life mana=$mana/$mana stamina=$stam/$stam exp=0 .*statpoints=0 skillpoints=0 ")
  [ "$n" -ge 2 ] || { echo "FAIL: want str/dex/vit/ene $str/$dex/$vit/$ene hp $life mana $mana stamina $stam, 0 stat and skill points in both games, got $n"; fail=1; }
  n=$(grep "PANEL inventory:" $log.txt | grep -cE "worn=\[$worn")
  [ "$n" -ge 2 ] || { echo "FAIL: want worn items matching '$worn' in both games, got $n"; fail=1; }
  n=$(grep -cE "containers loaded: .*inventory=$invn belt_items=$belt" $log.txt)
  [ "$n" -ge 2 ] || { echo "FAIL: want inventory=$invn belt_items=$belt in both games, got $n"; fail=1; }
  # the thrown javelin stack is a full stack (PLAYER_CreateStartingItem 0x5326c0: quantity = max stack); 0 for the others
  n=$(grep "PANEL inventory:" $log.txt | grep -cE "ammo=$ammo([^0-9]|\$)")
  [ "$n" -ge 2 ] || { echo "FAIL: want ammo=$ammo in both games, got $n"; fail=1; }
  grep -E "PANEL inventory:" $log.txt | grep -qE "worn=\[[^]]*(Crown|Plate|Boots|Ring|Amulet|Cap|Gloves)" && { echo "FAIL: placeholder gear is worn"; fail=1; }
  n=$(grep "PANEL skills active:" $log.txt | grep -cE "left=$left\(.* right=$right")
  [ "$n" -ge 2 ] || { echo "FAIL: want the left skill $left and the right skill $right in both games, got $n"; fail=1; }
  ls $sd/saves/*.d2s >/dev/null 2>&1 || { echo "FAIL: no .d2s"; fail=1; }
  grep -q "AUTOFLOW step [0-9]* exit" $log.txt || { echo "FAIL: the flow did not reach its end"; fail=1; }
}
