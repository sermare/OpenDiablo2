scenario_name="Act 1 population: Cold Plains (level 3) monsters are legal for its Levels.txt row and match its density"
source scripts/verify.d/lib/poplevel.sh
scenario_env() { poplevel_env 3; }
scenario_check() { poplevel_check 3 28; }
