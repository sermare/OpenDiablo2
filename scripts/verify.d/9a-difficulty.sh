scenario_name="difficulty (OD2_AUTODIFFICULTY=1: Nightmare chosen on the difficulty screen, pack scaled to the real tables, choice and per-difficulty progress in the exported .d2s)"
wb=$tmp/writeback-difficulty
scenario_env() {
  mkdir -p $wb
  # the sample save (progression 13) has Nightmare and Hell unlocked
  echo "export OD2_D2S_WRITEBACK=\"$wb\" OD2_AUTODIFFICULTY=1"
  echo 'export OD2_AUTOMONSTER="fallen1,pack" OD2_AUTOMONSTER_SECONDS=8'
}
scenario_check() {
  grep -E "DIFFICULTY|DIFFTEST|D2S EXPORT reparse" $log.txt | cut -c1-260 | head -20
  grep -q "DIFFICULTY screen .*unlocked=\[true true true\]" $log.txt || { echo "FAIL: difficulty screen did not unlock all three"; fail=1; }
  grep -q "DIFFICULTY chosen .*difficulty=Nightmare" $log.txt || { echo "FAIL: Nightmare not chosen"; fail=1; }
  grep -qE "DIFFTEST hero .*difficulty=1 \(Nightmare\) resist_penalty=-40 death_exp_penalty=5 merc_difficulty=2 client_difficulty=1" $log.txt \
    || { echo "FAIL: the game is not running in Nightmare"; fail=1; }
  grep -qE "DIFFTEST monster id=fallen1 difficulty=1 level=[0-9]+ hp=[1-9][0-9]* " $log.txt || { echo "FAIL: no Nightmare pack stats"; fail=1; }
  # fallen1 Nightmare by hand from patch_d2 (monlvl row 36 x monstats ratios):
  # defense 528*70/100=369, to-hit 738*90/100=664, damage 31*45/100=13..31*90/100=27, xp 2568*65/100=1669
  grep -qE "DIFFTEST monster id=fallen1 difficulty=1 level=36 hp=[0-9]+ defense=369 xp=1669 tc=.Act 1 .N. H2H A. a1=664/13-27" $log.txt \
    || { echo "FAIL: Nightmare fallen1 differs from the hand computation of the real tables"; fail=1; }
  grep -q 'DIFFTEST monster .*tc="[^"]*(N)' $log.txt || echo "note: no Nightmare treasure class in the pack lines"
  if grep -q "DIFFTEST monster .*difficulty=[02]" $log.txt; then echo "FAIL: a monster spawned at the wrong difficulty"; fail=1; fi
  # the export written for the run keeps Nightmare as the active difficulty
  grep -qE "D2S EXPORT reparse: .*difficulty=1 " $log.txt || { echo "FAIL: the exported .d2s lacks the Nightmare active byte"; fail=1; }
  # real-table hand computations and the unlock rules
  go test ./d2common/d2difficulty/ ./d2core/d2monsters/ 2>&1 | grep -v "ld: warning\|^# " | grep -v "^ok" && fail=1
}
