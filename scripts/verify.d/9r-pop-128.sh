scenario_name="Act 5 population: The Worldstone Keep Level 1 (level 128) monsters are legal for its Levels.txt row and match its density"
source scripts/verify.d/lib/poplevel.sh
scenario_env() { poplevel_env 128; }
scenario_check() { poplevel_check 128 auto; }
