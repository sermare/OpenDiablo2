scenario_name="Act 1 population: Hole Level 1 (level 11) monsters are legal for its Levels.txt row and match its density"
source scripts/verify.d/lib/poplevel.sh
scenario_env() { poplevel_env 11; }
scenario_check() { poplevel_check 11 auto; }
