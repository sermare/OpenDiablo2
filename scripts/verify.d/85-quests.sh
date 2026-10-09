scenario_name="quest system (Act 1 quest line driven by OD2_AUTOQUEST: bits, NPC speech + Sounds rows, rewards, .d2s persistence)"
scenario_env() {
  echo 'export OD2_AUTOQUEST=act1'
}
scenario_check() {
  grep -E "AUTOQUEST (start|RESULT)|QUEST system started" $log.txt | cut -c1-200
  grep -q "AUTOQUEST RESULT PASS" $log.txt || { echo "FAIL: the Act 1 quest scenario did not pass"; fail=1; }
  grep -E "AUTOQUEST expect .*FAIL" $log.txt | cut -c1-200
  # every stage logs the bit changes of its quest (QUEST ... bits before->after)
  for q in A1Q1 A1Q2 A1Q3 A1Q4 A1Q5 A1Q6; do
    grep -qE "QUEST $q slot=[0-9]+ bits 0x[0-9a-f]+->0x[0-9a-f]+ " $log.txt || { echo "FAIL: no bit changes logged for $q"; fail=1; }
  done
  # the speech lines carry the message id and the Sounds.txt row
  for m in 64 76 81 92 97 112 118 146 163 166 183; do
    grep -qE "QUEST SPEECH .* msg=$m mode=[0-9] .* sound=[0-9]+ handle=" $log.txt || { echo "FAIL: message $m was not spoken"; fail=1; }
  done
  grep -E "QUEST SPEECH" $log.txt | head -4 | cut -c1-230
  # the reward effects
  grep -qE "QUEST EFFECT skill-point \+1" $log.txt || { echo "FAIL: no skill point for Den of Evil"; fail=1; }
  grep -qE "QUEST EFFECT hire-rogues" $log.txt || { echo "FAIL: Kashya's reward"; fail=1; }
  grep -qE "QUEST EFFECT imbue-available" $log.txt || { echo "FAIL: Charsi's imbue"; fail=1; }
  # the quest bits reached the .d2s the game wrote back (bit 0 of slots 1..6, act 1 finished word in slot 7)
  last=$(grep -E "D2S EXPORT reparse" $log.txt | tail -1)
  echo "$last" | grep -oE "act1quests=\[[^]]*\]"
  for s in 1 2 3 4 5 6 7; do
    echo "$last" | grep -qE "act1quests=\[([^]]* )?$s:0x[0-9a-f]{3}[13579bdf]" || { echo "FAIL: quest slot $s is not saved as done in the .d2s"; fail=1; }
  done
}
