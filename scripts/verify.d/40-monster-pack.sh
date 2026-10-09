scenario_name="monster pack (natural group spawns, nobody stacks badly, hero fights it)"
scenario_env() { echo 'export OD2_AUTOMONSTER="fallen1,pack" OD2_AUTOMONSTER_SECONDS=15'; }
scenario_check() {
  grep -E "MONSTER pack|AUTOMONSTER (summary|world)" $log.txt | cut -c1-200
  grep -q "MONSTER pack leader=" $log.txt || { echo "FAIL: no pack spawned"; fail=1; }
  # two monsters may share a cell for a moment under load; three means a real problem
  grep -q "AUTOMONSTER world .*max_stack=[12] " $log.txt || { echo "FAIL: monsters stacked (3+) or no world summary"; fail=1; }
}
