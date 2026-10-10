scenario_name="Quest walkthrough, Act 2: The Tainted Sun in the real world (Drognan, Lost City, Valley of Snakes, Claw Viper Temple altar, Amulet of the Viper, quest log, .d2s)"
# The sample hero (quests reset) travels to the Lost City by waypoint (the sun darkens), walks home to Drognan,
# goes back through the Valley of Snakes into the Claw Viper Temple, operates the Tainted Sun altar on level 2 of the
# temple and picks up the Amulet of the Viper it gives. Needs D2_TABLES + D2S_SAMPLE_BODY.
source scripts/quest_walk_lib.zsh
scenario_warnings_ok=1
scenario_env() {
  # the sun darkens in the Lost City; Drognan tells the hero about it when he comes back (the quest log shows
  # nothing before that), then the hero goes to the altar
  local s="$(qw_begin 2)say:setwaypoint 44 1;$(qw_panel 2 3)$(qw_wp 44)$(qw_go 43 42 41 40)$(qw_talk Drognan)$(qw_panel 2 3)"
  s+="wait:11;$(qw_wp 44)$(qw_go 45 58 61)$(qw_chest 'Tainted Sun Altar')$(qw_panel 2 3)exit"
  qw_env taintedsun "$s"
}
scenario_check() {
  grep -E "AUTOSCRIPT RESULT|QUEST A2Q3|QUEST SPEECH npc=\"Drognan\"|QUESTPANEL act=2 quest=3|quest object" $log.txt | cut -c1-200
  qw_sample taintedsun || { echo "SKIP: no revived sample hero in this run"; return; }
  grep -q "AUTOSCRIPT RESULT PASS" $log.txt || { echo "FAIL: the Tainted Sun walkthrough did not pass"; fail=1; }
  grep -qE "QUEST SPEECH npc=\"Drognan\" class=177 msg=348 mode=[0-9] .* sound=[0-9]+ handle=ESOUND_DROGNAN" $log.txt || { echo "FAIL: Drognan did not speak message 348"; fail=1; }
  grep -q "QUEST A2Q3 state 0->1" $log.txt || { echo "FAIL: the Lost City did not darken the sun (state 0->1)"; fail=1; }
  grep -q "LEVEL quest object id=149 name=\"Tainted Sun Altar\"" $log.txt || { echo "FAIL: no Tainted Sun altar in the Claw Viper Temple"; fail=1; }
  grep -q "QUEST object operated id=149" $log.txt || { echo "FAIL: the altar was not operated"; fail=1; }
  grep -q "QUEST EFFECT spawn-item code=vip" $log.txt || { echo "FAIL: the altar gave no Amulet of the Viper"; fail=1; }
  grep -q 'LOOT stored "vip"' $log.txt || { echo "FAIL: the Amulet of the Viper was not picked up"; fail=1; }
  # the quest log: nothing before Drognan, a page after the talk, the goal page after the altar
  grep -q 'QUESTPANEL act=2 quest=3 status=0 .*text=""' $log.txt || { echo "FAIL: panel before the quest"; fail=1; }
  grep -qE "QUESTPANEL act=2 quest=3 status=[1-9][0-9]* .*text=\"[A-Za-z]" $log.txt || { echo "FAIL: panel shows no page text for the quest"; fail=1; }
  qw_laterquests
  qw_slot_has 11 0x2000 || { echo "FAIL: quest slot 11 does not carry the goal bit in the .d2s"; fail=1; }
  grep -E "AUTOSCRIPT step [0-9]+ FAIL" $log.txt | cut -c1-200
}
