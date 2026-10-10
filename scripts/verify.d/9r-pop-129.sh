scenario_name="Act 5 population: The Worldstone Keep Level 2 (level 129) monsters are legal for its Levels.txt row and match its density"
source scripts/verify.d/lib/poplevel.sh
scenario_env() { poplevel_env 129; }
scenario_check() { poplevel_check 129 auto; }
