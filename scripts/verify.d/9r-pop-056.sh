scenario_name="Act 2 population: Halls of the Dead Level 1 (level 56) monsters are legal for its Levels.txt row and match its density"
source scripts/verify.d/lib/poplevel.sh
scenario_env() { poplevel_env 56; }
scenario_check() { poplevel_check 56 auto; }
