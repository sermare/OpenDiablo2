scenario_name="Act 1 population: Barracks (level 28) monsters are legal for its Levels.txt row and match its density"
source scripts/verify.d/lib/poplevel.sh
scenario_env() { poplevel_env 28; }
scenario_check() { poplevel_check 28 17; }
