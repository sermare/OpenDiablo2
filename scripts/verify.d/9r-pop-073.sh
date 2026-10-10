scenario_name="Act 2 population: Duriel's Lair (level 73) has no natural monsters (Populate = 0)"
source scripts/verify.d/lib/poplevel.sh
scenario_env() { poplevel_env 73; }
scenario_check() { poplevel_check 73 0; }
