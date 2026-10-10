scenario_name="Act 1 population: Stony Field (level 4) monsters are legal for its Levels.txt row and match its density"
source scripts/verify.d/lib/poplevel.sh
scenario_env() { poplevel_env 4; }
scenario_check() { poplevel_check 4 38; }
