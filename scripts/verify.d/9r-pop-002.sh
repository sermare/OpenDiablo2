scenario_name="Act 1 population: Blood Moor (level 2) monsters are legal for its Levels.txt row and match its density"
source scripts/verify.d/lib/poplevel.sh
scenario_env() { poplevel_env 2; }
scenario_check() { poplevel_check 2 23; }
