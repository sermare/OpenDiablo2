scenario_name="Act 2 population: Arcane Sanctuary (level 74) monsters are legal for its Levels.txt row and match its density"
source scripts/verify.d/lib/poplevel.sh
scenario_env() { poplevel_env 74; }
scenario_check() { poplevel_check 74 178; }
