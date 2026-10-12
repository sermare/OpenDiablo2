scenario_name="Quest log text per state (OD2_AUTOQUEST=questlog: opens the log on Den of Evil, Tools, Staff, Golden Bird and Siege in several states and logs the string key and text)"
scenario_env() {
  echo 'export OD2_AUTOQUEST=questlog'
}
scenario_check() {
  grep -E "AUTOQUEST (start|RESULT)|QUEST system started" $log.txt | cut -c1-200
  grep -q "AUTOQUEST RESULT PASS" $log.txt || { echo "FAIL: the quest log scenario did not pass"; fail=1; }
  grep -E "AUTOQUEST expect .*FAIL" $log.txt | cut -c1-200
  grep -E "AUTOQUEST questlog \[" $log.txt | cut -c1-260
  # every state logged its key and a non-empty text for the pages that have one
  for k in qstsa1q11 qstsa1q12 qstsa1q15 qstsComplete qstsa1q32 qstsa2q21 qstsa3q41 qsta5q11 qsta5q13 qsta5q14; do
    grep -qE "AUTOQUEST questlog \[.*key=$k " $log.txt || { echo "FAIL: key $k never shown"; fail=1; }
  done
  grep -E "AUTOQUEST questlog \[" $log.txt | grep -E "key=qst[a-zA-Z0-9]+ " | grep -q 'text=""' && { echo "FAIL: a quest log page with a key showed no text"; fail=1; }
}
