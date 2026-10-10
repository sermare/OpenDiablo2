scenario_name="Act 5 population: The Worldstone Keep Level 3 (level 130) monsters are legal for its Levels.txt row and match its density"
source scripts/verify.d/lib/poplevel.sh
scenario_env() { poplevel_env 130; }
scenario_check() { poplevel_check 130 26; }
