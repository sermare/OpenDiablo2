scenario_name="Act 1 population: Tower Cellar Level 1 (level 21) monsters are legal for its Levels.txt row and match its density"
source scripts/verify.d/lib/poplevel.sh
scenario_env() { poplevel_env 21; }
scenario_check() { poplevel_check 21 6; }
