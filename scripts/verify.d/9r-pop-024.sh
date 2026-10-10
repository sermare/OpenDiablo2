scenario_name="Act 1 population: Tower Cellar Level 4 (level 24) monsters are legal for its Levels.txt row and match its density"
source scripts/verify.d/lib/poplevel.sh
scenario_env() { poplevel_env 24; }
scenario_check() { poplevel_check 24 auto; }
