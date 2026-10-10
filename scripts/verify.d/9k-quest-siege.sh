scenario_name="Quest walkthrough, Act 5: Siege on Harrogath in the real world (Larzuk, Bloody Foothills, Shenk the Overseer, Larzuk's reward, quest log, .d2s)"
# The sample hero (quests reset, Normal monsters) arrives in Harrogath, takes the quest from Larzuk, walks into the Bloody
# Foothills, finds and kills Shenk the Overseer (the super unique of the DS1: the boss carries his own name now) and
# returns to Larzuk for the reward (the socket reward itself is the reward dialog, not tested here).
# Needs D2_TABLES + D2S_SAMPLE_BODY.
source scripts/quest_walk_lib.zsh
scenario_warnings_ok=1
scenario_env() {
  # (the Bloody Foothills are no waypoint level: the hero walks across the border and back)
  local s="$(qw_begin 5)$(qw_panel 5 1)$(qw_talk Larzuk)$(qw_panel 5 1)$(qw_go 110)kill:name=Shenk,300;$(qw_panel 5 1)"
  s+="$(qw_go 109)$(qw_talk Larzuk)$(qw_panel 5 1)exit"
  qw_env siege "$s"
}
scenario_check() {
  grep -E "AUTOSCRIPT RESULT|QUEST A5Q1|QUEST SPEECH npc=\"Larzuk\"|QUESTPANEL act=5 quest=1|MONSTER death name=Shenk|adopted DS1 super unique|QUEST EFFECT reward" $log.txt | cut -c1-200
  qw_sample siege || { echo "SKIP: no revived sample hero in this run"; return; }
  grep -q "AUTOSCRIPT RESULT PASS" $log.txt || { echo "FAIL: the Siege on Harrogath walkthrough did not pass"; fail=1; }
  # Larzuk's lines with their message ids and Sounds.txt rows: the quest line (20077) and the thanks (20090)
  for m in 20077 20090; do
    grep -qE "QUEST SPEECH npc=\"Larzuk\" class=511 msg=$m mode=[0-9] .* sound=[0-9]+ handle=ESOUND_LARZUK_ACT5_Q1" $log.txt || { echo "FAIL: Larzuk did not speak message $m"; fail=1; }
  done
  grep -q 'LEVEL npcs: .*Larzuk(511)' $log.txt || { echo "FAIL: no Larzuk in Harrogath"; fail=1; }
  # the world: the super unique of the Bloody Foothills is Shenk the Overseer; killing him is the goal
  grep -qE "adopted DS1 super unique Siege Boss \(Shenk the Overseer\)|POPULATE supers level 110: .*overseer1" $log.txt || { echo "FAIL: Shenk the Overseer is not placed in the Bloody Foothills"; fail=1; }
  grep -q "MONSTER death name=Shenk the Overseer" $log.txt || { echo "FAIL: Shenk was not killed"; fail=1; }
  for t in "1->2" "2->3" "3->4" "4->5"; do
    grep -q "QUEST A5Q1 state $t" $log.txt || { echo "FAIL: no state change $t of A5Q1"; fail=1; }
  done
  grep -q "QUEST EFFECT reward socket-quest" $log.txt || { echo "FAIL: Larzuk's socket reward"; fail=1; }
  # the quest log: the title only (the game's string table has no page text for this quest), then the completed text
  grep -q 'QUESTPANEL act=5 quest=1 status=0 title="Siege on Harrogath"' $log.txt || { echo "FAIL: panel before the quest"; fail=1; }
  grep -q "QUESTPANEL act=5 quest=1 status=-1 .*completed" $log.txt || { echo "FAIL: panel completed text"; fail=1; }
  qw_laterquests
  qw_slot_has 35 0x1001 || { echo "FAIL: quest slot 35 is not saved as done in the .d2s"; fail=1; }
  grep -E "AUTOSCRIPT step [0-9]+ FAIL" $log.txt | cut -c1-200
}
