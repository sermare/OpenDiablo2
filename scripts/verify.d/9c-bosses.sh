scenario_name="boss encounters (Duriel tomb, Mephisto wake, Diablo seals, Baal throne waves: triggers, spawns, state changes, quest bits set by the kills)"
scenario_env() { echo 'export OD2_AUTOBOSS=duriel,mephisto,diablo,baal'; }
scenario_check() {
  grep -E "AUTOBOSS (start|step|RESULT|QUEST|FAIL)|BOSS .*(trigger|killed|wave [0-9] spawned|morphs)" $log.txt | cut -c1-210
  grep -q "AUTOBOSS RESULT PASS" $log.txt || { echo "FAIL: the boss scenario did not pass"; fail=1; }
  grep -E "AUTOBOSS FAIL" $log.txt | cut -c1-200

  # Duriel: the orifice needs the staff, the portal opens, one Duriel in the lair, Tyrael after the kill
  grep -q "BOSS duriel trigger: orifice operated without the Horadric Staff" $log.txt || { echo "FAIL: duriel: orifice without staff"; fail=1; }
  grep -q "BOSS duriel trigger: Horadric Staff placed" $log.txt || { echo "FAIL: duriel: staff trigger"; fail=1; }
  grep -q "BOSS duriel action: portal portal to Duriel's Lair" $log.txt || { echo "FAIL: duriel: portal"; fail=1; }
  grep -q "BOSS duriel trigger: lair entered -> state in-lair" $log.txt || { echo "FAIL: duriel: lair"; fail=1; }
  grep -q "BOSS duriel action: spawn-monster Duriel" $log.txt || { echo "FAIL: duriel: spawn"; fail=1; }
  grep -q "BOSS duriel action: spawn-npc Tyrael" $log.txt || { echo "FAIL: duriel: Tyrael"; fail=1; }
  grep -qE "AUTOBOSS QUEST boss=duriel slot=14 .*primary_goal=true" $log.txt || { echo "FAIL: duriel: quest bit"; fail=1; }

  # Mephisto: asleep while the hero is far, wakes when he comes close
  grep -q "BOSS mephisto trigger: Durance of Hate level 3 entered -> state asleep" $log.txt || { echo "FAIL: mephisto: sleep"; fail=1; }
  grep -q "BOSS mephisto state -> awake (hero within the wake range)" $log.txt || { echo "FAIL: mephisto: wake"; fail=1; }
  grep -q "BOSS mephisto action: portal red portal" $log.txt || { echo "FAIL: mephisto: red portal"; fail=1; }
  grep -qE "AUTOBOSS QUEST boss=mephisto slot=22 .*primary_goal=true" $log.txt || { echo "FAIL: mephisto: quest bit"; fail=1; }

  # Diablo: five seals, the three seal bosses, then one Diablo
  grep -q "BOSS diablo trigger: seal boss Grand Vizier of Chaos dead" $log.txt || { echo "FAIL: diablo: vizier"; fail=1; }
  grep -q "BOSS diablo trigger: seal boss Lord De Seis dead" $log.txt || { echo "FAIL: diablo: de seis"; fail=1; }
  grep -q "BOSS diablo trigger: seal boss Infector of Souls dead" $log.txt || { echo "FAIL: diablo: infector"; fail=1; }
  [ "$(grep -c 'BOSS diablo trigger: all 5 seals open and 3 seal bosses dead' $log.txt)" = 1 ] || { echo "FAIL: diablo: arrival is not a one-shot"; fail=1; }
  grep -q "BOSS diablo action: spawn-monster Diablo" $log.txt || { echo "FAIL: diablo: spawn"; fail=1; }
  grep -qE "AUTOBOSS QUEST boss=diablo slot=26 .*primary_goal=true" $log.txt || { echo "FAIL: diablo: quest bit"; fail=1; }

  # Baal: five waves with the verified groups, the morph, the portal, Baal in the chamber
  for w in 1 2 3 4 5; do
    grep -qE "BOSS baal wave $w spawned" $log.txt || { echo "FAIL: baal: wave $w"; fail=1; }
  done
  grep -q "BOSS baal: the throne morphs into Baal" $log.txt || { echo "FAIL: baal: morph"; fail=1; }
  grep -q "AUTOBOSS baal left the level through the portal" $log.txt || { echo "FAIL: baal: did not walk into the Worldstone portal"; fail=1; }
  grep -q "BOSS baal trigger: Worldstone Chamber entered" $log.txt || { echo "FAIL: baal: chamber"; fail=1; }
  grep -qE "AUTOBOSS QUEST boss=baal slot=40 .*primary_goal=true" $log.txt || { echo "FAIL: baal: quest bit"; fail=1; }
  grep -q "MONSTER aistate .*to=BaalThrone" $log.txt || grep -q "ai=BaalThrone" $log.txt || { echo "FAIL: baal: the throne AI did not run"; fail=1; }
}
