scenario_name="front-end flow (OD2_AUTOFLOW: main menu -> create -> play -> Save and Exit -> reload -> delete; .d2s.bak and atomic write)"
# Everything happens in temp copies: a private config dir (own Saves folder for the .od2 files) and a
# private save folder that stands in for the player's Diablo II Save folder. Nothing real is written.
scenario_warnings_ok=1
flowdir=$tmp/flowsaves
sample_name=$(dd if="$D2S_SAMPLE_BODY" bs=1 skip=20 count=16 2>/dev/null | tr -d '\0')
scenario_env() {
  mkdir -p $flowdir $tmp/flowcfg
  cp "$D2S_SAMPLE_BODY" $flowdir/Sample.d2s
  cp "$HOME/Library/Application Support/OpenDiablo2/config.json" $tmp/flowcfg/ 2>/dev/null
  echo "unset OD2_AUTOGAME OD2_D2S_WRITEBACK"
  echo "export OD2_CONFIG_DIR=\"$tmp/flowcfg\" OD2_D2S_DIR=\"$flowdir\""
  # 1 refused name, 1 hardcore Necromancer, a first save with 777 gold, back to the list, play it again (gold survived),
  # play the real level-94 save and change its gold, play it again, then delete the new character
  local steps="single,new,create:Sorceress:x,create:Necromancer:Flowy:hardcore"
  steps+=",gold:777,wait:2,saveexit,select:Flowy,play,wait:2,saveexit"
  steps+=",select:$sample_name,play,difficulty:0,gold:31337,wait:2,saveexit"
  steps+=",select:$sample_name,play,difficulty:0,wait:2,saveexit"
  steps+=",delete:Flowy,exit"
  echo "export OD2_AUTOFLOW=\"$steps\""
}
scenario_check() {
  grep -E "AUTOFLOW|NEWCHAR|CHARSELECT deleted|D2S EXPORT reparse" $log.txt | cut -c1-260
  grep -qE "NEWCHAR refused \"x\"" $log.txt || { echo "FAIL: the one-letter name was not refused"; fail=1; }
  grep -qE "NEWCHAR created Flowy class=Necromancer expansion=true hardcore=true d2s=$flowdir/Flowy.d2s" $log.txt || { echo "FAIL: Flowy not created in the save folder"; fail=1; }
  grep -qE "CHARSELECT slot=[0-9]+ name=\"Flowy\" class=Necromancer level=1 hardcore=true expansion=true ladder=false dead=false imported=true" $log.txt || { echo "FAIL: Flowy not listed after creation"; fail=1; }
  grep -qE "CHARSELECT slot=[0-9]+ name=\"$sample_name\" class=[A-Za-z]+ level=[0-9]+" $log.txt || { echo "FAIL: the real save is not listed"; fail=1; }
  [ "$(grep -c 'AUTOFLOW saveexit gold=777' $log.txt)" -ge 2 ] || { echo "FAIL: Flowy's gold did not survive Save and Exit + reload"; fail=1; }
  grep -qE "D2S EXPORT reparse: name=Flowy .*hardcore=true .*checksum=ok" $log.txt || { echo "FAIL: Flowy's save does not re-parse"; fail=1; }
  grep -qE "D2S EXPORT reparse: name=$sample_name .*gold=31337 .*checksum=ok" $log.txt || { echo "FAIL: the real save was not written back with the new gold"; fail=1; }
  grep -qE "AUTOFLOW saveexit gold=31337" $log.txt || { echo "FAIL: the changed gold was not there after the reload"; fail=1; }
  cmp -s "$D2S_SAMPLE_BODY" $flowdir/Sample.d2s.bak || { echo "FAIL: Sample.d2s.bak is not the untouched original"; fail=1; }
  cmp -s "$D2S_SAMPLE_BODY" $flowdir/Sample.d2s && { echo "FAIL: Sample.d2s was not rewritten"; fail=1; }
  [ -n "$(find $flowdir -name "*.tmp")" ] && { echo "FAIL: temporary file left behind"; fail=1; }
  [ -f $flowdir/Flowy.d2s ] && { echo "FAIL: Flowy.d2s still there after the delete"; fail=1; }
  [ -f $flowdir/Flowy.d2s.deleted ] || { echo "FAIL: deleted Flowy.d2s was not kept as .deleted"; fail=1; }
  grep -qE "AUTOFLOW step [0-9]+ exit" $log.txt || { echo "FAIL: the flow did not reach its last step"; fail=1; }
  grep -E "\[ERROR\]|panic" $log.txt | grep -v "skipping missing" && { echo "FAIL: errors in the log"; fail=1; }
}
