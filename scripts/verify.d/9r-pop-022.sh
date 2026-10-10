scenario_name="Act 1 population: Tower Cellar Level 2 (level 22) monsters are legal for its Levels.txt row and match its density"
source scripts/verify.d/lib/poplevel.sh
scenario_env() { poplevel_env 22; }
scenario_check() { poplevel_check 22 auto; }
