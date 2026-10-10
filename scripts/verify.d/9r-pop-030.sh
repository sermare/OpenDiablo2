scenario_name="Act 1 population: Jail Level 2 (level 30) monsters are legal for its Levels.txt row and match its density"
source scripts/verify.d/lib/poplevel.sh
scenario_env() { poplevel_env 30; }
scenario_check() { poplevel_check 30 auto; }
