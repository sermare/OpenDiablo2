scenario_name="Act 1 population: Black Marsh (level 6) monsters are legal for its Levels.txt row and match its density"
source scripts/verify.d/lib/poplevel.sh
scenario_env() { poplevel_env 6; }
scenario_check() { poplevel_check 6 auto; }
