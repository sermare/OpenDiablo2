scenario_name="Act 5 population: Glacial Caves Level 1 (level 118) monsters are legal for its Levels.txt row and match its density"
source scripts/verify.d/lib/poplevel.sh
scenario_env() { poplevel_env 118; }
scenario_check() { poplevel_check 118 auto; }
