scenario_name="Act 1 population: Cave Level 2 (level 13) monsters are legal for its Levels.txt row and match its density"
source scripts/verify.d/lib/poplevel.sh
scenario_env() { poplevel_env 13; }
scenario_check() { poplevel_check 13 auto; }
