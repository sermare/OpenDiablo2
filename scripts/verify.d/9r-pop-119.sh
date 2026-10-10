scenario_name="Act 5 population: Glacial Caves Level 2 (level 119) monsters are legal for its Levels.txt row and match its density"
source scripts/verify.d/lib/poplevel.sh
scenario_env() { poplevel_env 119; }
scenario_check() { poplevel_check 119 auto; }
