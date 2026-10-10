scenario_name="Act 1 population: Pit Level 2 (level 16) monsters are legal for its Levels.txt row and match its density"
source scripts/verify.d/lib/poplevel.sh
scenario_env() { poplevel_env 16; }
scenario_check() { poplevel_check 16 auto; }
