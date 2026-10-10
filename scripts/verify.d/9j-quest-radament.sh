scenario_name="Quest walkthrough, Act 2: Radament's Lair in the real world (Atma, Sewers 1-3, Radament, Book of Skill, Atma's reward, quest log, .d2s)"
# The level 94 sample hero with his quests reset travels to Lut Gholein, takes the quest from Atma, walks into the
# sewers by the manhole of the town, kills Radament (placed by the DS1 of Sewers Level 3), picks up the Book of Skill
# he drops, reads it (+1 skill point), walks back out and collects Atma's reward. Needs D2_TABLES + D2S_SAMPLE_BODY.
source scripts/quest_walk_lib.zsh
scenario_warnings_ok=1
scenario_env() {
  local s="$(qw_begin 2)$(qw_panel 2 1)$(qw_talk Atma)$(qw_panel 2 1)$(qw_go 47 48 49)kill:all,200;$(qw_panel 2 1)loot:30,40;say:useitem ass;"
  s+="$(qw_go 48 47 40)$(qw_talk Atma 10)$(qw_panel 2 1)exit"
  qw_env radament "$s"
}
scenario_check() {
  grep -E "AUTOSCRIPT RESULT|QUEST A2Q1 state|QUESTPANEL act=2 quest=1|USEITEM|QUEST SPEECH npc=\"Atma\"" $log.txt | cut -c1-210
  qw_sample radament || { echo "SKIP: no revived sample hero in this run"; return; }
  grep -q "AUTOSCRIPT RESULT PASS" $log.txt || { echo "FAIL: the Radament walkthrough did not pass"; fail=1; }
  # the speech: the quest line and the reward line carry their message ids and Sounds.txt rows
  for m in 304 334; do
    grep -qE "QUEST SPEECH npc=\"Atma\" class=176 msg=$m mode=[0-9] .* sound=[0-9]+ handle=ESOUND_ATMA" $log.txt || { echo "FAIL: Atma did not speak message $m"; fail=1; }
  done
  # the quest bits in the world: talk -> leave town -> kill -> reward
  for t in "1->2" "2->3" "3->4" "4->5"; do
    grep -q "QUEST A2Q1 state $t" $log.txt || { echo "FAIL: no state change $t of A2Q1"; fail=1; }
  done
  grep -q "MONSTER death name=Radament" $log.txt || { echo "FAIL: Radament was not killed"; fail=1; }
  grep -q "QUEST EFFECT spawn-item code=ass" $log.txt || { echo "FAIL: Radament dropped no Book of Skill"; fail=1; }
  grep -q 'LOOT stored "ass"' $log.txt || { echo "FAIL: the Book of Skill was not picked up"; fail=1; }
  grep -q "USEITEM code=ass took_effect=true" $log.txt || { echo "FAIL: the Book of Skill had no effect"; fail=1; }
  grep -q "QUEST EFFECT skill-point +1" $log.txt || { echo "FAIL: the Book of Skill gave no skill point"; fail=1; }
  # the quest log panel: page 1 after Atma, page 3 after the kill, the completed text after the reward
  grep -q 'QUESTPANEL act=2 quest=1 status=0 .*text=""' $log.txt || { echo "FAIL: panel before the quest"; fail=1; }
  grep -q "QUESTPANEL act=2 quest=1 status=1 .*Find Radament's Lair" $log.txt || { echo "FAIL: panel page 1 text"; fail=1; }
  grep -q "QUESTPANEL act=2 quest=1 status=3 .*Return to Atma" $log.txt || { echo "FAIL: panel page 3 text"; fail=1; }
  grep -q "QUESTPANEL act=2 quest=1 status=-1 .*completed" $log.txt || { echo "FAIL: panel completed text"; fail=1; }
  # saved: slot 9 of the exported .d2s is done (bit 0) and seen in the log (bit 12)
  qw_laterquests
  qw_slot_has 9 0x1001 || { echo "FAIL: quest slot 9 is not saved as done in the .d2s"; fail=1; }
  grep -E "AUTOSCRIPT step [0-9]+ FAIL" $log.txt | cut -c1-200
}
