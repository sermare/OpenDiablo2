scenario_name="Act 2 quest speech (OD2_AUTOQUEST=act2: Radament, Tainted Sun, Horadric Staff, Arcane Sanctuary, Summoner, Seven Tombs; Warriv, Fara, Drognan, Greiz, Jerhyn, Elzix, Lysander...)"
scenario_env() {
  echo 'export OD2_AUTOQUEST=act2'
}
scenario_check() {
  grep -E "AUTOQUEST (start|RESULT)|QUEST system started" $log.txt | cut -c1-200
  grep -q "AUTOQUEST RESULT PASS" $log.txt || { echo "FAIL: the Act 2 quest scenario did not pass"; fail=1; }
  grep -E "AUTOQUEST expect .*FAIL" $log.txt | cut -c1-200
  for q in A2Q1 A2Q2 A2Q3 A2Q4 A2Q5 A2Q6; do
    grep -qE "QUEST $q slot=[0-9]+ bits 0x[0-9a-f]+->0x[0-9a-f]+ " $log.txt || { echo "FAIL: no bit changes logged for $q"; fail=1; }
  done
  # the speech of every Act 2 NPC of the quest line (message ids of the speech tables)
  for m in 304 334 335 336 337 338 339 348 373 377 430 442 444 445 446 447 449 450 452; do
    grep -qE "QUEST SPEECH .* msg=$m mode=[0-9] .* sound=[0-9]+ handle=" $log.txt || { echo "FAIL: message $m was not spoken"; fail=1; }
  done
  # the speakers: Warriv, Fara, Drognan, Greiz, Jerhyn, Elzix, Lysander, Atma, Cain, Meshif
  for npc in Drognan Jerhyn Atma Cain Meshif Warriv; do
    grep -E "QUEST SPEECH npc=\"[^\"]*$npc" $log.txt | grep -q . || { echo "FAIL: $npc never spoke"; fail=1; }
  done
  grep -E "QUEST SPEECH" $log.txt | grep -E "msg=(348|373|442) " | cut -c1-260
  grep -E "QUEST SPEECH" $log.txt | grep -v 'text=""' | wc -l | sed 's/^ */spoken lines with a text: /'
  # the quest bits reached the .d2s the game wrote back (Act 2 slots 9..14, the Act 2 finished word in slot 15)
  last=$(grep -E "D2S EXPORT reparse" $log.txt | tail -1)
  echo "$last" | cut -c1-300
}
