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
}
