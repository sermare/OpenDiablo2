scenario_name="Act 1 population: Underground Passage Level 2 (level 14) monsters are legal for its Levels.txt row and match its density"
source scripts/verify.d/lib/poplevel.sh
scenario_env() { poplevel_env 14; }
scenario_check() { poplevel_check 14 7; }
