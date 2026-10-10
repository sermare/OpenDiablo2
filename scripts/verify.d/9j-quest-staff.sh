scenario_name="Quest walkthrough, Act 2: The Horadric Staff in the real world (Maggot Lair staff chest, Halls of the Dead cube chest, Sewers Level 3 scroll chest, Cain, cube, quest log, .d2s)"
# The sample hero (quests reset) opens the three quest chests in their dungeons, picks up the Staff of Kings, the
# Horadric Cube and the Horadric Scroll, shows them to Deckard Cain in Lut Gholein (the real town NPC), transmutes
# the staff in the cube (the Amulet of the Viper comes from the Claw Viper Temple altar, scenario 9j-quest-taintedsun;
# here it is given) and hears Cain's confirmation. Needs D2_TABLES + D2S_SAMPLE_BODY.
source scripts/quest_walk_lib.zsh
scenario_warnings_ok=1
scenario_env() {
  local s="$(qw_begin 2)say:setwaypoint 43 1;say:setwaypoint 48 1;say:setwaypoint 57 1;"
  # Far Oasis has no waypoint object in this world (docs/PLAYTEST.md): the way back to the Halls is on foot
  s+="$(qw_wp 43)$(qw_go 62 63 64)$(qw_chest 356)"
  s+="$(qw_go 63 62 43 42 56 57 60)$(qw_chest 354)"
  s+="$(qw_go 57)$(qw_wp 48)$(qw_go 49)$(qw_chest 355)$(qw_go 48)$(qw_wp 40)say:giveitem vip;"
  s+="$(qw_talk 'Deckard Cain' 10)$(qw_talk 'Deckard Cain' 10)$(qw_talk 'Deckard Cain' 10)$(qw_talk 'Deckard Cain' 10)"
  s+="$(qw_panel 2 2)say:cubeput msf;say:cubeput vip;say:transmute;wait:2;$(qw_panel 2 2)$(qw_talk 'Deckard Cain' 10)$(qw_panel 2 2)exit"
  qw_env staff "$s"
}
scenario_check() {
  grep -E "AUTOSCRIPT RESULT|QUEST object|QUEST SPEECH npc=\"Deckard Cain\"|QUESTPANEL act=2 quest=2|CUBE transmute" $log.txt | cut -c1-200
  qw_sample staff || { echo "SKIP: no revived sample hero in this run"; return; }
  grep -q "AUTOSCRIPT RESULT PASS" $log.txt || { echo "FAIL: the Horadric Staff walkthrough did not pass"; fail=1; }
  # the three quest objects exist in their levels and give their item once (QUEST EFFECT spawn-item), which is picked up
  for it in msf box tr1; do
    grep -q "QUEST EFFECT spawn-item code=$it " $log.txt || { echo "FAIL: the chest gave no $it"; fail=1; }
    grep -q "LOOT stored \"$it\"" $log.txt || { echo "FAIL: $it was not picked up"; fail=1; }
  done
  # Cain's lines in order: scroll (335), amulet (336), staff (337) and, after the cube made the staff, 339; the
  # message ids come with their Sounds.txt rows and the text of the string table
  for m in 335 336 337 339; do
    grep -qE "QUEST SPEECH npc=\"Deckard Cain\" class=244 msg=$m mode=[0-9] .* sound=[0-9]+ handle=ESOUND_CAIN_ACT2_Q2" $log.txt || { echo "FAIL: Cain did not speak message $m"; fail=1; }
  done
  grep -E "QUEST SPEECH npc=\"Deckard Cain\" class=244 msg=(335|336|337|339) " $log.txt | grep -q 'text=""' && { echo "FAIL: a Cain line of the Staff quest has no text"; fail=1; }
  grep -q "CUBE transmute .*Horadric Staff" $log.txt || { echo "FAIL: the cube did not make the Horadric Staff"; fail=1; }
  grep -q "QUEST A2Q2 slot=10 bits 0x[0-9a-f]*->0x[0-9a-f]* (Horadric Staff assembled)" $log.txt || { echo "FAIL: the quest did not see the staff"; fail=1; }
  # the quest log: page 3 once Cain heard every part, then page 4 (the staff must go to Tal Rasha's tomb) also after Cain's confirmation
  grep -q "QUESTPANEL act=2 quest=2 status=4 .*Tal Rasha" $log.txt || { echo "FAIL: panel page 4 text"; fail=1; }
  grep -q "QUESTPANEL act=2 quest=2 status=3 .*Horadric Cube" $log.txt || { echo "FAIL: panel page 3 text (every part reported)"; fail=1; }
  grep -q "QUESTPANEL act=2 quest=2 status=5" $log.txt && { echo "FAIL: the panel shows page 5 (take the artifacts to Cain) after Cain heard of every part"; fail=1; }
  # saved: slot 10 carries the quest bits (Cain's reports and the assembled staff) in the .d2s
  qw_laterquests
  qw_slot_has 10 0x0c78 || { echo "FAIL: quest slot 10 does not carry the Staff bits in the .d2s"; fail=1; }
  grep -E "AUTOSCRIPT step [0-9]+ FAIL" $log.txt | cut -c1-200
}
