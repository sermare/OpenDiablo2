scenario_name="Act 2 population: Palace Cellar Level 1 (level 52) monsters are legal for its Levels.txt row and match its density"
source scripts/verify.d/lib/poplevel.sh
scenario_env() { poplevel_env 52; }
scenario_check() { poplevel_check 52 auto; }
