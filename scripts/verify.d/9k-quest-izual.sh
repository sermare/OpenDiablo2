scenario_name="Quest walkthrough, Act 4: The Fallen Angel in the real world (Tyrael, Outer Steppes, Plains of Despair, Izual, Tyrael's reward, quest log, .d2s)"
# The sample hero (quests reset, Normal monsters) arrives in the Pandemonium Fortress, talks to Tyrael (his lines come
# newest quest first: Terror's End, The Fallen Angel, the Act 4 welcome), walks through the Outer Steppes into the Plains of
# Despair, kills everything there including Izual (placed by the DS1) and returns to Tyrael for the 2 skill points.
# Needs D2_TABLES + D2S_SAMPLE_BODY.
source scripts/quest_walk_lib.zsh
scenario_warnings_ok=1
scenario_env() {
  local s="$(qw_begin 4)$(qw_panel 4 1)$(qw_talk Tyrael)$(qw_talk Tyrael)$(qw_talk Tyrael)$(qw_panel 4 1)$(qw_go 104 105)kill:all,250;$(qw_panel 4 1)"
  s+="$(qw_go 104 103)$(qw_talk Tyrael)$(qw_talk Tyrael)$(qw_panel 4 1)exit"
  qw_env izual "$s"
}
scenario_check() {
  grep -E "AUTOSCRIPT RESULT|QUEST A4Q1|QUEST SPEECH npc=\"Tyrael\"|QUESTPANEL act=4 quest=1|MONSTER death name=Izual|QUEST EFFECT skill" $log.txt | cut -c1-200
  qw_sample izual || { echo "SKIP: no revived sample hero in this run"; return; }
  grep -q "AUTOSCRIPT RESULT PASS" $log.txt || { echo "FAIL: the Izual walkthrough did not pass"; fail=1; }
  # Tyrael's lines with their message ids and Sounds.txt rows: the quest line (670), the thanks (676), the others
  for m in 664 670 676 681; do
    grep -qE "QUEST SPEECH npc=\"Tyrael\" class=367 msg=$m mode=[0-9] .* sound=[0-9]+ handle=ESOUND_TYRAEL" $log.txt || { echo "FAIL: Tyrael did not speak message $m"; fail=1; }
  done
  grep -E "QUEST SPEECH npc=\"Tyrael\" class=367 msg=(664|670|676|681) " $log.txt | grep -q 'text=""' && { echo "FAIL: a Tyrael line of the Act 4 quests has no text"; fail=1; }
  # the quest bits in the world: talk (2), leave town (3), enter the Plains, kill Izual (goal), reward
  for t in "1->2" "2->3" "3->4" "4->5"; do
    grep -q "QUEST A4Q1 state $t" $log.txt || { echo "FAIL: no state change $t of A4Q1"; fail=1; }
  done
  grep -q "MONSTER death name=Izual" $log.txt || { echo "FAIL: Izual was not killed (he is not in the Plains of Despair)"; fail=1; }
  grep -q "QUEST EFFECT skill-point +2" $log.txt || { echo "FAIL: Tyrael's reward (2 skill points)"; fail=1; }
  # the quest log: nothing, "Look for Izual", "See Tyrael for reward", the completed text
  grep -q 'QUESTPANEL act=4 quest=1 status=0 .*text=""' $log.txt || { echo "FAIL: panel before the quest"; fail=1; }
  grep -q "QUESTPANEL act=4 quest=1 status=1 .*Look for Izual" $log.txt || { echo "FAIL: panel page 1 text"; fail=1; }
  grep -q "QUESTPANEL act=4 quest=1 status=3 .*See Tyrael" $log.txt || { echo "FAIL: panel page 3 text"; fail=1; }
  grep -q "QUESTPANEL act=4 quest=1 status=-1 .*completed" $log.txt || { echo "FAIL: panel completed text"; fail=1; }
  qw_laterquests
  qw_slot_has 25 0x1001 || { echo "FAIL: quest slot 25 is not saved as done in the .d2s"; fail=1; }
  grep -E "AUTOSCRIPT step [0-9]+ FAIL" $log.txt | cut -c1-200
}
