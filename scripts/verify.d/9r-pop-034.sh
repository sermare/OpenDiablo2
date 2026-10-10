scenario_name="Act 1 population: Catacombs Level 1 (level 34) monsters are legal for its Levels.txt row and match its density"
source scripts/verify.d/lib/poplevel.sh
scenario_env() { poplevel_env 34; }
scenario_check() { poplevel_check 34 auto; }
