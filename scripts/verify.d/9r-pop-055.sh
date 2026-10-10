scenario_name="Act 2 population: Stony Tomb Level 1 (level 55) monsters are legal for its Levels.txt row and match its density"
source scripts/verify.d/lib/poplevel.sh
scenario_env() { poplevel_env 55; }
scenario_check() { poplevel_check 55 auto; }
