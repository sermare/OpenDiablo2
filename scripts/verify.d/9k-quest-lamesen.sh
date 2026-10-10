scenario_name="Quest walkthrough, Act 3: Lam Esen's Tome in the real world (Alkor, Kurast Bazaar, Ruined Temple, tome on the altar, Alkor's reward, quest log, .d2s)"
# The sample hero (quests reset) arrives in Kurast Docktown, takes the quest from Alkor, goes to Kurast Bazaar by
# waypoint and through the temple entrance into the Ruined Temple, operates the tome on its altar, picks up Lam Esen's
# Tome, walks back to the waypoint and hands the tome to Alkor (+5 stat points). Needs D2_TABLES + D2S_SAMPLE_BODY.
source scripts/quest_walk_lib.zsh
scenario_warnings_ok=1
scenario_env() {
  local s="$(qw_begin 3)say:setwaypoint 80 1;say:setwaypoint 75 1;$(qw_panel 3 1)$(qw_talk Alkor)$(qw_panel 3 1)"
  s+="wait:11;$(qw_wp 80)$(qw_go 94)$(qw_chest 193)$(qw_panel 3 1)$(qw_go 80)wait:11;$(qw_wp 75)$(qw_talk Alkor 10)$(qw_panel 3 1)exit"
  qw_env lamesen "$s"
}
scenario_check() {
  grep -E "AUTOSCRIPT RESULT|QUEST A3Q1|QUEST SPEECH npc=\"Alkor\"|QUESTPANEL act=3 quest=1|quest object|QUEST EFFECT" $log.txt | grep -v "log-update" | cut -c1-200
  qw_sample lamesen || { echo "SKIP: no revived sample hero in this run"; return; }
  grep -q "AUTOSCRIPT RESULT PASS" $log.txt || { echo "FAIL: the Lam Esen walkthrough did not pass"; fail=1; }
  # Alkor's lines (the quest line and the thanks) with their message ids and Sounds.txt rows
  for m in 549 564; do
    grep -qE "QUEST SPEECH npc=\"Alkor\" class=254 msg=$m mode=[0-9] .* sound=[0-9]+ handle=ESOUND_ALKOR_ACT3_Q1" $log.txt || { echo "FAIL: Alkor did not speak message $m"; fail=1; }
  done
  grep -q "LEVEL quest object id=193 name=\"Lam Esen's Tome\"" $log.txt || { echo "FAIL: no tome in the Ruined Temple"; fail=1; }
  grep -qE "QUEST object operated id=193|ACT3 .* operated \(id 193\)" $log.txt || { echo "FAIL: the tome was not operated"; fail=1; }
  grep -qE "QUEST EFFECT spawn-item code=bbb|ACT3 .* operated \(id 193\): gives \"bbb\"" $log.txt || { echo "FAIL: the tome object gave no item"; fail=1; }
  grep -q 'LOOT stored "bbb"' $log.txt || { echo "FAIL: Lam Esen's Tome was not picked up"; fail=1; }
  grep -q "QUEST EFFECT reward stat-points +5" $log.txt || { echo "FAIL: Alkor's reward (5 stat points)"; fail=1; }
  grep -q "QUEST EFFECT delete-item code=bbb" $log.txt || { echo "FAIL: Alkor kept no tome"; fail=1; }
  # the quest log: nothing before Alkor, a page text in the temple, the completed text after the reward
  grep -q 'QUESTPANEL act=3 quest=1 status=0 .*text=""' $log.txt || { echo "FAIL: panel before the quest"; fail=1; }
  grep -qE "QUESTPANEL act=3 quest=1 status=[1-9][0-9]* .*text=\"[A-Za-z]" $log.txt || { echo "FAIL: panel shows no page text"; fail=1; }
  grep -q "QUESTPANEL act=3 quest=1 status=-1 .*completed" $log.txt || { echo "FAIL: panel completed text"; fail=1; }
  qw_laterquests
  qw_slot_has 17 0x1001 || { echo "FAIL: quest slot 17 is not saved as done in the .d2s"; fail=1; }
  grep -E "AUTOSCRIPT step [0-9]+ FAIL" $log.txt | cut -c1-200
}
