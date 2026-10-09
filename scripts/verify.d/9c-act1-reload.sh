scenario_name="Act 1 reload (the .d2s the 9b playthrough exported is loaded again: same hero, quests, Akara's talk afterwards)"
a1=$tmp/act1
scenario_env() {
  rm -f $a1/reload.d2s; mkdir -p $a1/wb2
  # without the 9b playthrough (a filtered run) there is nothing to reload: use the sample
  if [ -s $a1/wb/Playtest.d2s ]; then cp $a1/wb/Playtest.d2s $a1/reload.d2s; else cp "$D2S_SAMPLE_BODY" $a1/reload.d2s; echo "# no playthrough .d2s" >&2; fi
  echo "export OD2_AUTOGAME=\"$a1/reload.d2s\" OD2_D2S_WRITEBACK=\"$a1/wb2\" OD2_REALMAPS=1"
  echo "export OD2_AUTOSCRIPT='wait:1;panel:quest;wait:1;say:capframe $tmp/act1-reload-quests.png;panel:close;move:npc=Akara;until:NPC menu opened: npc=\"Akara\",60;menu:Talk;wait:3;exit'"
}
scenario_check() {
  grep -E "HERO state|QUEST system started|QUEST LOG act=1 quest=1|NPC menu opened|QUEST SPEECH|AUTOSCRIPT RESULT|D2S EXPORT reparse" $log.txt | cut -c1-260 | head -14
  grep -q "AUTOSCRIPT RESULT PASS" $log.txt || { echo "FAIL: the reload run did not pass"; fail=1; }
  [ -s $tmp/act1-reload-quests.png ] || { echo "FAIL: no screenshot of the quest log"; fail=1; }
  [ -s $a1/play.reparse ] || { echo "SKIP: no 9b playthrough in this run, nothing to compare"; return; }
  before=$(cat $a1/play.reparse)
  after=$(grep "D2S EXPORT reparse" $log.txt | tail -1)
  # the same hero: every value of the reparse line that the game itself does not change by standing in town
  for key in name class level exp gold str dex vit ene difficulty act items equipped waypoints spent unused; do
    b=$(echo "$before" | grep -oE "(^|[ :])$key=[^ ]*" | head -1 | sed 's/^[ :]//')
    a=$(echo "$after" | grep -oE "(^|[ :])$key=[^ ]*" | head -1 | sed 's/^[ :]//')
    [ "$a" = "$b" ] || { echo "FAIL: $key differs after reload: before '$b' after '$a'"; fail=1; }
  done
  # the quests: the Den of Evil is still done (opening the quest log marks the completion as seen, which
  # changes the log flags of the slot, so the whole word is not compared)
  echo "$after" | grep -oE "act1quests=\[[^]]*\]"
  echo "$after" | grep -qE "act1quests=\[([^]]* )?1:0x[0-9a-f]{3}[13579bdf]" || { echo "FAIL: the Den of Evil is not done after the reload"; fail=1; }
  # the hero is the one that left the game
  if [ -s $a1/play.state ]; then
    s=$(grep -oE "level=[0-9]+ exp=[0-9]+ skillpoints=[0-9]+" $a1/play.state | head -1)
    r=$(grep "HERO state at start" $log.txt | grep -oE "level=[0-9]+ exp=[0-9]+ skillpoints=[0-9]+" | head -1)
    [ "$s" = "$r" ] || { echo "FAIL: the hero at start of the reload ($r) is not the one who left ($s)"; fail=1; }
  fi
  # Den of Evil is done in the quest log (status 3 = completed) and Akara has nothing more to say about it
  grep -qE "QUEST LOG act=1 quest=1 status=3" $log.txt || { echo "FAIL: the quest log does not show the Den of Evil as completed"; fail=1; }
  grep -qE "QUEST SPEECH npc=\"Akara\" .* msg=(64|65) " $log.txt && { echo "FAIL: Akara repeats the Den of Evil after it is done"; fail=1; }
}
