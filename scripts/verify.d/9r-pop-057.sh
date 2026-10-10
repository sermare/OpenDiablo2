scenario_name="Act 2 population: Halls of the Dead Level 2 (level 57) monsters are legal for its Levels.txt row and match its density"
source scripts/verify.d/lib/poplevel.sh
scenario_env() { poplevel_env 57; }
scenario_check() { poplevel_check 57 72; }
