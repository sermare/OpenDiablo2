scenario_name="quest rewards, act-travel hooks, quest cube recipes and the Pandemonium event (OD2_AUTOUBER: Larzuk/Anya/Malah/Alkor rewards, Hellforge drops, Khalim's Will + Staff, keys -> uber areas -> organs -> Tristram)"
scenario_env() { echo 'export OD2_AUTOUBER=rewards,travel,cube,uber'; }
scenario_check() {
  grep -E "AUTOUBER (start|RESULT)|AUTOUBER expect .*FAIL" $log.txt | cut -c1-200
  grep -q "AUTOUBER RESULT PASS" $log.txt || { echo "FAIL: the quest reward / Pandemonium scenario did not pass"; fail=1; }

  # rewards are applied to the hero, not only logged
  grep -qE "QUEST EFFECT reward life-boost value=20 .* max life [0-9]+ -> [0-9]+" $log.txt || { echo "FAIL: Potion of Life not applied"; fail=1; }
  grep -qE "QUEST EFFECT reward resist-bonus value=10 .* resistances \+10 now fire=" $log.txt || { echo "FAIL: Malah's scroll not applied"; fail=1; }
  grep -qE "REWARD socket item=lsd sockets=[1-6] " $log.txt || { echo "FAIL: Larzuk did not socket the sword"; fail=1; }
  grep -qE "REWARD personalize item=cap name=" $log.txt || { echo "FAIL: Anya did not personalise the helm"; fail=1; }
  grep -qE "QUEST EFFECT reward hire-barbarians .* mercenaries of barbarians are hirable" $log.txt || { echo "FAIL: Qual-Kehk's reward"; fail=1; }
  grep -qE "QUEST DROP hfh from \"Hephasto the Armorer\"" $log.txt || { echo "FAIL: Hellforge Hammer drop"; fail=1; }
  grep -qE "QUEST DROP mss from \"Mephisto\"" $log.txt || { echo "FAIL: Mephisto Soulstone drop"; fail=1; }

  # act travel reaches the quest system
  for t in "1 -> 2 via=act:npc" "2 -> 3 via=act:npc" "3 -> 4 via=portal" "4 -> 5 via=act:talk"; do
    grep -q "QUEST travel act $t" $log.txt || { echo "FAIL: no quest travel line for $t"; fail=1; }
  done

  # the cube recipes of the quests
  grep -q 'CUBE transmute "Staff of Kings + Viper amulet -> Horadric Staff"' $log.txt || { echo "FAIL: Horadric Staff recipe"; fail=1; }
  grep -q "CUBE transmute \"Khalim Flail + Heart + Eye + Brain -> Khalim's Will\"" $log.txt || { echo "FAIL: Khalim's Will recipe"; fail=1; }

  # Pandemonium: three portals to the uber areas, their bosses (real monstats rows, own AIs), the organs, Tristram
  for lvl in 133 134 135 136; do
    grep -q "UBER portal .* opened to level $lvl" $log.txt || { echo "FAIL: no portal to level $lvl"; fail=1; }
  done
  grep -q "OBJECT spawned portal to level 136" $log.txt || { echo "FAIL: the Tristram portal object was not spawned"; fail=1; }
  for k in uberandariel:Andariel uberduriel:Duriel uberizual:UberIzual ubermephisto:UberMephisto uberdiablo:UberDiablo uberbaal:UberBaal; do
    grep -qE "UBER spawn name=.* key=${k%%:*} ai=${k##*:} level=1(33|34|35|36)" $log.txt || { echo "FAIL: ${k%%:*} was not spawned with AI ${k##*:}"; fail=1; }
  done
  for organ in dhn bey mbr; do
    grep -q "UBER drop $organ " $log.txt || { echo "FAIL: organ $organ not dropped"; fail=1; }
  done
  grep -q "BOSS uber trigger: all three Tristram bosses dead -> the event is won" $log.txt || { echo "FAIL: the event was not won"; fail=1; }
  grep -q "UBER drop std (Standard of Heroes)" $log.txt || { echo "FAIL: no final reward"; fail=1; }
}
