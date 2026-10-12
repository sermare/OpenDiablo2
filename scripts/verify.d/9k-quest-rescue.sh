scenario_name="Quest walkthrough, Act 5: the first steps of Rescue on Mount Arreat in the real world (Qual-Kehk, Bloody Foothills, Frigid Highlands, quest log, .d2s)"
# The sample hero (quests reset) takes the quest from Qual-Kehk (the Rite of Passage line is not heard: that quest opens after Betrayal of Harrogath), travels to the Frigid Highlands by waypoint (the quest moves on, the three barbarian cages are
# there), kills the prison doors on the cages (monster class 434, placed on the cage objects when the level is built:
# no DS1 monster marker can place them), walks back by the waypoint and takes Qual-Kehk's reward (runes by the number
# of freed barbarians, not a hire). Needs D2_TABLES + D2S_SAMPLE_BODY.
source scripts/quest_walk_lib.zsh
scenario_warnings_ok=1
scenario_env() {
  # (the Act 5 quests open one after the other, exe sequence filters: the Siege on Harrogath is done first)
  local s="$(qw_begin 5)say:completequest 5 1;$(qw_panel 5 2)$(qw_talk Qual-Kehk)$(qw_panel 5 2)say:setwaypoint 111 1;wait:11;$(qw_wp 111)$(qw_panel 5 2)skill:right=Fire Bolt;kill:name=prison door,300;$(qw_panel 5 2)$(qw_wp 109)$(qw_talk Qual-Kehk)$(qw_panel 5 2)exit"
  qw_env rescue "$s"
}
scenario_check() {
  grep -E "AUTOSCRIPT RESULT|PRISONDOOR|MONSTER death name=Prison|QUEST EFFECT give-item|QUEST A5Q2|QUEST SPEECH npc=\"Qual-Kehk\"|QUESTPANEL act=5 quest=2|LEVEL objects: .*Cage|RESCUE" $log.txt | cut -c1-220
  qw_sample rescue || { echo "SKIP: no revived sample hero in this run"; return; }
  grep -q "AUTOSCRIPT RESULT PASS" $log.txt || { echo "FAIL: the Rescue on Mount Arreat walkthrough did not pass"; fail=1; }
  grep -qE "QUEST SPEECH npc=\"Qual-Kehk\" class=515 msg=20096 mode=[0-9] .* sound=[0-9]+ handle=ESOUND_QUALKEHK" $log.txt || { echo "FAIL: Qual-Kehk did not speak message 20096"; fail=1; }
  grep -q "QUEST SPEECH npc=\"Qual-Kehk\" class=515 msg=20153 " $log.txt && { echo "FAIL: Qual-Kehk offered the Rite of Passage before Betrayal of Harrogath was done"; fail=1; }
  grep -E "QUEST SPEECH npc=\"Qual-Kehk\" class=515 msg=(20153|20096) " $log.txt | grep -q 'text=""' && { echo "FAIL: a Qual-Kehk line has no text"; fail=1; }
  for t in "1->2" "2->3" "3->4" "4->5"; do
    grep -q "QUEST A5Q2 state $t" $log.txt || { echo "FAIL: no state change $t of A5Q2"; fail=1; }
  done
  grep -q "QUEST A5Q2 slot=36 bits 0x[0-9a-f]*->0x[0-9a-f]* (entered the area)" $log.txt || { echo "FAIL: the Frigid Highlands did not move the quest"; fail=1; }
  grep -q "LEVEL objects: .*Cage x3" $log.txt || { echo "FAIL: no barbarian cages in the Frigid Highlands"; fail=1; }
  grep -q "PRISONDOOR placed 3 prison door" $log.txt || { echo "FAIL: no prison doors on the barbarian cages"; fail=1; }
  [ "$(grep -c 'MONSTER death name=Prison Door' $log.txt)" -ge 3 ] || { echo "FAIL: the three prison doors were not destroyed"; fail=1; }
  [ "$(grep -c 'RESCUE door died: 5 prisoner(s) freed' $log.txt)" -ge 3 ] || { echo "FAIL: the three prison doors did not free five barbarians each"; fail=1; }
  grep -q "RESCUE door died: .* (total 15 of 15)" $log.txt || { echo "FAIL: the freed count did not reach 15 of 15 barbarians"; fail=1; }
  [ "$(grep -c 'RESCUE prisoner [0-9]* reached the portal and left' $log.txt)" -ge 3 ] || { echo "FAIL: the freed barbarians did not walk to the portal and leave"; fail=1; }
  # (no barbarian-hire reward exists, the hire gate is slot 36 bit 0 (exe); Qual-Kehk pays runes by the freed count, 15 = Tal, Ral, Ort)
  for r in r07 r08 r09; do
    grep -q "QUEST EFFECT give-item code=$r " $log.txt || { echo "FAIL: Qual-Kehk paid no rune $r"; fail=1; }
  done
  grep -q "QUEST A5Q2 slot=36 bits .* (all fifteen barbarians freed)" $log.txt || { echo "FAIL: the freed-barbarian count was not recorded"; fail=1; }
  grep -q 'QUESTPANEL act=5 quest=2 status=0 title="Rescue on Mount Arreat"' $log.txt || { echo "FAIL: panel before the quest"; fail=1; }
  grep -q "QUESTPANEL act=5 quest=2 status=1 .*Find the Soldiers in the Frigid Highlands" $log.txt || { echo "FAIL: panel page 1 text"; fail=1; }
  # w37 (exe 0x5860f0, VERIFIED): entering level 111/112 moves the record but the log keeps page 1 while no page is pushed, so
  # the panel opened in the Frigid Highlands still reads page 1; the "Rescue %d more Soldiers" count page is covered by
  # d2player TestFillCount / d2quest w37_acts45_test
  [ "$(grep -c 'QUESTPANEL act=5 quest=2 status=1 .*Find the Soldiers in the Frigid Highlands' $log.txt)" -ge 2 ] || { echo "FAIL: panel page 1 text not kept in the Frigid Highlands"; fail=1; }
  grep -q "QUESTPANEL act=5 quest=2 status=3 .*Return to Qual-Kehk" $log.txt || { echo "FAIL: panel page 3 text"; fail=1; }
  grep -q "QUESTPANEL .*text=\".*%d" $log.txt && { echo "FAIL: a quest log page shows a raw %d"; fail=1; }
  qw_laterquests
  # (the payment clears the progress bits: done, primary goal and completion-seen remain)
  qw_slot_has 36 0x3001 || { echo "FAIL: quest slot 36 does not carry the done, goal and completion-seen bits (0x3001) in the .d2s"; fail=1; }
  grep -E "AUTOSCRIPT step [0-9]+ FAIL" $log.txt | cut -c1-200
}
