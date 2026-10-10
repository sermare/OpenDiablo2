scenario_name="Act 1 population: Jail Level 1 (level 29) monsters are legal for its Levels.txt row and match its density"
source scripts/verify.d/lib/poplevel.sh
scenario_env() { poplevel_env 29; }
scenario_check() { poplevel_check 29 24; }
