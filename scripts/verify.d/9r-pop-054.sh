scenario_name="Act 2 population: Palace Cellar Level 3 (level 54) monsters are legal for its Levels.txt row and match its density"
source scripts/verify.d/lib/poplevel.sh
scenario_env() { poplevel_env 54; }
scenario_check() { poplevel_check 54 64; }
