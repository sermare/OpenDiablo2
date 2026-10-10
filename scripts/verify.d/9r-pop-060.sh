scenario_name="Act 2 population: Halls of the Dead Level 3 (level 60) monsters are legal for its Levels.txt row and match its density"
source scripts/verify.d/lib/poplevel.sh
scenario_env() { poplevel_env 60; }
scenario_check() { poplevel_check 60 auto; }
