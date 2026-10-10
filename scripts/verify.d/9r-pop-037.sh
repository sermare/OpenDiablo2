scenario_name="Act 1 population: Catacombs Level 4 (level 37) monsters are legal for its Levels.txt row and match its density"
source scripts/verify.d/lib/poplevel.sh
scenario_env() { poplevel_env 37; }
scenario_check() { poplevel_check 37 10; }
