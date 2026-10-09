scenario_name="skill pipeline (hero casts Fire Bolt at spawned zombies: mana, missiles, hits, kills)"
scenario_env() { echo 'export OD2_AUTOMONSTER="zombie1,2" OD2_AUTOMONSTER_SECONDS=30 OD2_AUTOCAST="Fire Bolt,4"'; }
scenario_check() {
  grep -E "AUTOCAST (start|summary)|CAST do|MISSILE hit" $log.txt | cut -c1-200 | head -12
  grep -qE "CAST do skill=\"Fire Bolt\" .*ok=true .*missiles=1 .*mana_paid=2.50" $log.txt || { echo "FAIL: no Fire Bolt cast that paid 2.50 mana"; fail=1; }
  grep -qE "MISSILE hit name=firebolt" $log.txt || { echo "FAIL: no Fire Bolt hit a monster"; fail=1; }
  grep -qE "AUTOCAST summary .* kills=[1-9]" $log.txt || { echo "FAIL: no kill by Fire Bolt"; fail=1; }
}
