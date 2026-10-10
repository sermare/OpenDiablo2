scenario_name="Act 1 population: Catacombs Level 3 (level 36) monsters are legal for its Levels.txt row and match its density"
source scripts/verify.d/lib/poplevel.sh
scenario_env() { poplevel_env 36; }
scenario_check() { poplevel_check 36 36; }
