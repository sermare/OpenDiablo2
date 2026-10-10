scenario_name="Act 2 population: Palace Cellar Level 2 (level 53) monsters are legal for its Levels.txt row and match its density"
source scripts/verify.d/lib/poplevel.sh
scenario_env() { poplevel_env 53; }
scenario_check() { poplevel_check 53 59; }
