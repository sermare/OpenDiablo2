scenario_name="quests of Acts 2-5 (OD2_AUTOQUEST=acts2-5: one or more quests per act, bits, NPC speech + Sounds rows, rewards, .d2s persistence)"
scenario_env() {
  echo 'export OD2_AUTOQUEST=acts2-5'
}
scenario_check() {
  grep -E "AUTOQUEST (start|RESULT)|QUEST system started" $log.txt | cut -c1-200
  grep -q "AUTOQUEST RESULT PASS" $log.txt || { echo "FAIL: the Acts 2-5 quest scenario did not pass"; fail=1; }
  grep -E "AUTOQUEST expect .*FAIL" $log.txt | cut -c1-200
  # every stage logs the bit changes of its quest (QUEST ... bits before->after)
  for q in A2Q3 A2Q4 A2Q6 A3Q1 A3Q3 A4Q1 A4Q2 A5Q1 A5Q5; do
    grep -qE "QUEST $q slot=[0-9]+ bits 0x[0-9a-f]+->0x[0-9a-f]+ " $log.txt || { echo "FAIL: no bit changes logged for $q"; fail=1; }
  done
  # the speech lines carry the message id and the Sounds.txt row: one quest-giver line per act
  for m in 348 373 377 430 442 450 549 564 587 593 664 670 681 20077 20153; do
    grep -qE "QUEST SPEECH .* msg=$m mode=[0-9] .* sound=[0-9]+ handle=" $log.txt || { echo "FAIL: message $m was not spoken"; fail=1; }
  done
  grep -E "QUEST SPEECH" $log.txt | grep -E "msg=(430|549|664|20077) " | head -4 | cut -c1-230
  # rewards and act travel
  grep -qE "QUEST EFFECT reward stat-points \+5" $log.txt || { echo "FAIL: Lam Esen's stat points"; fail=1; }
  grep -qE "QUEST EFFECT reward hire-ironwolves" $log.txt || { echo "FAIL: Asheara's reward"; fail=1; }
  grep -qE "QUEST EFFECT skill-point \+2" $log.txt || { echo "FAIL: Fallen Angel's skill points"; fail=1; }
  grep -qE "QUEST EFFECT reward socket-quest" $log.txt || { echo "FAIL: Larzuk's reward"; fail=1; }
  grep -qE "QUEST EFFECT unlock act (3|5)" $log.txt || { echo "FAIL: no act unlock"; fail=1; }
  # the quest bits reached the .d2s the game wrote back: bit 0 (done) of the completed slots, and the Act 2 and Act 4 words
  last=$(grep -E "D2S EXPORT reparse" $log.txt | tail -1)
  echo "$last" | grep -oE "laterquests=\[[^]]*\]"
  for s in 11 12 14 15 17 19 25 26 28 35 39; do
    echo "$last" | grep -qE "laterquests=\[([^]]* )?$s:0x[0-9a-f]{3}[13579bdf]" || { echo "FAIL: quest slot $s is not saved as done in the .d2s"; fail=1; }
  done
}
