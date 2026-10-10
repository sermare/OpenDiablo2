scenario_name="Act 1 population: Jail Level 3 (level 31) monsters are legal for its Levels.txt row and match its density"
source scripts/verify.d/lib/poplevel.sh
scenario_env() { poplevel_env 31; }
scenario_check() { poplevel_check 31 auto; }
