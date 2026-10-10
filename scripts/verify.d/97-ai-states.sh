scenario_name="monster AI states (forced fear / confuse / charm override the AI, are injected with the forcestate command and restore it; monsters fight monsters)"
scenario_env() { echo 'export OD2_AUTOAI="skeleton1,state=fear+confuse+charm" OD2_AUTOAI_SECONDS=22'; }
scenario_check() {
  grep -E "AUTOAI (start|inject|summary)|MONSTER (state|aistate)|forcestate:" $log.txt | cut -c1-220
  grep -q "AUTOAI start ref=skeleton1" $log.txt || { echo "FAIL: AUTOAI did not start"; fail=1; }
  for kind in fear confuse charm; do
    grep -q "AUTOAI inject: forcestate [0-9]* $kind " $log.txt || { echo "FAIL: $kind was not injected"; fail=1; }
    grep -q "MONSTER state .* kind=$kind " $log.txt || { echo "FAIL: no state line for $kind"; fail=1; }
  done
  # fear swaps the think function and gives the class AI back when it ends
  grep -q "MONSTER aistate .* from=Skeleton to=State11+fear" $log.txt || { echo "FAIL: fear did not override the AI"; fail=1; }
  grep -q "MONSTER aistate .* from=State11+fear to=Skeleton" $log.txt || { echo "FAIL: fear did not restore the AI"; fail=1; }
  grep -q "MONSTER aistate .* from=Skeleton to=Skeleton+confuse" $log.txt || { echo "FAIL: confuse not applied"; fail=1; }
  grep -q "MONSTER aistate .* from=Skeleton+confuse to=Skeleton" $log.txt || { echo "FAIL: confuse not restored"; fail=1; }
  grep -q "MONSTER aistate .* to=Skeleton+charm+allied" $log.txt || { echo "FAIL: charm not applied"; fail=1; }
  grep -q "MONSTER aistate .* from=Skeleton+charm+allied to=Skeleton" $log.txt || { echo "FAIL: charm not restored"; fail=1; }
  # monsters against monsters (confused or converted monster and the bystanders)
  grep -qE "MONSTER attack name=.* target=[^ ]+\([0-9]+\) " $log.txt || { echo "FAIL: no monster-versus-monster attack"; fail=1; }
  grep -q "AUTOAI summary" $log.txt || { echo "FAIL: no AUTOAI summary"; fail=1; }
  # archetype coverage (monster-ai-4): the subject runs an implemented AI, every monai.txt name has a Go think
  # function and the ported ground archetypes act (static tests, no game needed)
  grep -q "AUTOAI start ref=skeleton1 .*implemented=true" $log.txt || { echo "FAIL: subject AI not implemented"; fail=1; }
  go test ./d2common/d2monster/ -run 'TestMonaiTableCoverage|TestNoCommonMonsterIdles|TestMonsterAI4' -count=1 2>&1 | grep -v "ignoring duplicate libraries" | tail -5 | grep -q '^ok' || { echo "FAIL: monster AI archetype tests"; fail=1; }
  # faithful ports (feat/ai-faithful-a, monsters A..M): one in-game subject per family. Each runs its own short
  # game (one window at a time) and must start with the ported think function ("implemented=true" and the AI
  # name) and run without errors; the Go table tests cover the decisions themselves.
  local main_log=$log ref ai fam sub_cmd sub_slot
  for fam in baboon1:Baboon fingermage1:FingerMage foulcrow1:BloodHawk gargoyletrap:GargoyleTrap suckernest1:MosquitoNest deathmauler1:DeathMauler cr_lancer1:CorruptLancer; do
    ref=${fam%%:*}; ai=${fam##*:}
    sub_cmd=$tmp/97-sub-$ref.command log=$tmp/97-sub-$ref.log; rm -f $log
    {
      echo '#!/bin/zsh'
      echo "export OD2_PORT=$OD2_PORT"
      echo "export OD2_AUTOGAME=\"$save\" OD2_AUTOEXIT=1 OD2_AUTOTEST_MUTE=1 OD2_AUTOSPEED=4"
      echo "export OD2_AUTOAI=$ref OD2_AUTOAI_SECONDS=14"
      echo "$tmp/od2 2>&1 | tee $log"
    } > $sub_cmd
    chmod +x $sub_cmd
    sub_slot=$(./scripts/gameslot.sh acquire $$)
    launch_game $sub_cmd
    wait_run
    ./scripts/gameslot.sh release $sub_slot
    sed 's/\x1b\[[0-9;]*m//g' $log > $log.txt
    grep -E "AUTOAI (start|summary)" $log.txt | cut -c1-200
    grep -q "AUTOAI start ref=$ref .* ai=$ai implemented=true" $log.txt || { echo "FAIL: $ref did not start with the ported $ai AI"; fail=1; }
    grep -q "AUTOAI summary" $log.txt || { echo "FAIL: no AUTOAI summary for $ref"; fail=1; }
    # the fighters must have acted (attacks or skills in the summary); the nest and the statue are passive here
    case $ref in gargoyletrap|suckernest1) ;; *)
      grep "AUTOAI summary ref=$ref " $log.txt | grep -qE "attacks=[1-9]|skills=[1-9]" || { echo "FAIL: $ref never attacked or cast"; fail=1; } ;;
    esac
    grep -E "\[(ERROR|WARNING)\]|panic" $log.txt | grep -v "skipping missing" | head -3 | grep -q . && { echo "FAIL: errors in the $ref log"; fail=1; }
  done
  log=$main_log
  go test ./d2common/d2monster/ -run 'TestFaithfulA|TestAncientStatue' -count=1 2>&1 | grep -v "ignoring duplicate libraries" | tail -5 | grep -q '^ok' || { echo "FAIL: faithful AI port tests"; fail=1; }
}
