scenario_name="Act 1 population: Catacombs Level 2 (level 35) monsters are legal for its Levels.txt row and match its density"
source scripts/verify.d/lib/poplevel.sh
scenario_env() { poplevel_env 35; }
scenario_check() { poplevel_check 35 auto; }
