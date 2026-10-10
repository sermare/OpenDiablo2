scenario_name="Act 4 population: Plains of Despair (level 105) monsters are legal for its Levels.txt row and match its density"
source scripts/verify.d/lib/poplevel.sh
scenario_env() { poplevel_env 105; }
scenario_check() { poplevel_check 105 auto; }
