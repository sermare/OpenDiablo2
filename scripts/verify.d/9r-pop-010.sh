scenario_name="Act 1 population: Underground Passage Level 1 (level 10) monsters are legal for its Levels.txt row and match its density"
source scripts/verify.d/lib/poplevel.sh
scenario_env() { poplevel_env 10; }
scenario_check() { poplevel_check 10 32; }
