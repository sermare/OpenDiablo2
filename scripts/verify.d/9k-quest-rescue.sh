scenario_name="Quest walkthrough, Act 5: the first steps of Rescue on Mount Arreat in the real world (Qual-Kehk, Bloody Foothills, Frigid Highlands, quest log, .d2s)"
# The sample hero (quests reset) takes the quest from Qual-Kehk (his lines come newest quest first: Rite of Passage, then
# Rescue), walks through the Bloody Foothills into the Frigid Highlands (the quest moves on, the three barbarian cages are
# there). The prison doors that the quest wants killed are NOT placed in the generated Frigid Highlands (see docs/PLAYTEST.md),
# so the quest cannot go further; the check pins what works. Needs D2_TABLES + D2S_SAMPLE_BODY.
source scripts/quest_walk_lib.zsh
scenario_warnings_ok=1
scenario_env() {
  local s="$(qw_begin 5)$(qw_panel 5 2)$(qw_talk Qual-Kehk)$(qw_talk Qual-Kehk)$(qw_panel 5 2)$(qw_go 110 111)wait:2;$(qw_panel 5 2)exit"
  qw_env rescue "$s"
}
scenario_check() {
  grep -E "AUTOSCRIPT RESULT|QUEST A5Q2|QUEST SPEECH npc=\"Qual-Kehk\"|QUESTPANEL act=5 quest=2|LEVEL objects: .*Cage" $log.txt | cut -c1-220
  qw_sample rescue || { echo "SKIP: no revived sample hero in this run"; return; }
  grep -q "AUTOSCRIPT RESULT PASS" $log.txt || { echo "FAIL: the Rescue on Mount Arreat walkthrough did not pass"; fail=1; }
  for m in 20153 20096; do
    grep -qE "QUEST SPEECH npc=\"Qual-Kehk\" class=515 msg=$m mode=[0-9] .* sound=[0-9]+ handle=ESOUND_QUALKEHK" $log.txt || { echo "FAIL: Qual-Kehk did not speak message $m"; fail=1; }
  done
  grep -E "QUEST SPEECH npc=\"Qual-Kehk\" class=515 msg=(20153|20096) " $log.txt | grep -q 'text=""' && { echo "FAIL: a Qual-Kehk line has no text"; fail=1; }
  for t in "1->2" "2->3"; do
    grep -q "QUEST A5Q2 state $t" $log.txt || { echo "FAIL: no state change $t of A5Q2"; fail=1; }
  done
  grep -q "QUEST A5Q2 slot=36 bits 0x[0-9a-f]*->0x[0-9a-f]* (entered the area)" $log.txt || { echo "FAIL: the Frigid Highlands did not move the quest"; fail=1; }
  grep -q "LEVEL objects: .*Cage x3" $log.txt || { echo "FAIL: no barbarian cages in the Frigid Highlands"; fail=1; }
  grep -q 'QUESTPANEL act=5 quest=2 status=0 title="Rescue on Mount Arreat"' $log.txt || { echo "FAIL: panel before the quest"; fail=1; }
  grep -q "QUESTPANEL act=5 quest=2 status=1 .*Find the Soldiers in the Frigid Highlands" $log.txt || { echo "FAIL: panel page 1 text"; fail=1; }
  grep -q "QUESTPANEL act=5 quest=2 status=2 .*Rescue 15 more Soldiers" $log.txt || { echo "FAIL: panel page 2 text (the count)"; fail=1; }
  grep -q "QUESTPANEL .*text=\".*%d" $log.txt && { echo "FAIL: a quest log page shows a raw %d"; fail=1; }
  qw_laterquests
  qw_slot_has 36 0x001c || { echo "FAIL: quest slot 36 does not carry the started/left town/entered area bits in the .d2s"; fail=1; }
  grep -E "AUTOSCRIPT step [0-9]+ FAIL" $log.txt | cut -c1-200
}
