scenario_name="Act 2 population: Stony Tomb Level 2 (level 59) monsters are legal for its Levels.txt row and match its density"
source scripts/verify.d/lib/poplevel.sh
scenario_env() { poplevel_env 59; }
scenario_check() { poplevel_check 59 auto; }
